package tac

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/model"
)

func TestCodegenReportInventoriesOnlyEmittedStorage(t *testing.T) {
	x := Operand{Kind: Symbol, Value: "s1.x", Type: model.TypeInteger}
	temp := Operand{Kind: Temporary, Value: "t0", Type: model.TypeInteger}
	p := &Program{PeakTemporaries: 1, Instructions: []Instruction{{Op: Add, Target: temp, Left: x, Right: &x}, {Op: Copy, Target: x, Left: temp}}}
	scopes := model.ScopeSnapshots{
		{ID: 2, Symbols: model.Symbols{{Name: "unused", Kind: model.SymbolVariable}}},
		{ID: 1, Symbols: model.Symbols{{Name: "x", Kind: model.SymbolVariable, Type: model.Type{Kind: model.TypeInteger}}}},
	}
	before, _ := json.Marshal([]any{p, scopes})
	r := BuildCodegenReport(p, scopes)
	if len(r.Spaces) != 2 || r.Spaces[0].ID != "globals" || r.Spaces[0].Cells != 1 || r.Spaces[1].ID != "main" || r.Spaces[1].Cells != 1 {
		t.Fatalf("invented or missing storage: %+v", r.Spaces)
	}
	if !reflect.DeepEqual(r.Spaces[0].Slots[0].Operand, &x) || r.Spaces[1].Slots[0].Operand.Type != "" || !r.Spaces[1].Slots[0].Dynamic {
		t.Fatal("operand identity or reused temporary typing changed")
	}
	if len(r.Symbols[1].Locations) != 0 || len(r.Symbols[1].Labels) != 0 {
		t.Fatal("unemitted source symbol must not get fabricated storage")
	}
	after, _ := json.Marshal([]any{p, scopes})
	if !bytes.Equal(before, after) {
		t.Fatal("report mutated its inputs")
	}
	first, _ := json.Marshal(r)
	scopes[0], scopes[1] = scopes[1], scopes[0]
	second, _ := json.Marshal(BuildCodegenReport(p, scopes))
	if !bytes.Equal(first, second) {
		t.Fatal("scope traversal order changes report")
	}
	if BuildCodegenReport(nil, scopes) != nil {
		t.Fatal("nil TAC must not yield a partial report")
	}
}

func TestCodegenReportRetainsAllActualLocations(t *testing.T) {
	x := Operand{Kind: Symbol, Value: "s2.x", Type: model.TypeInteger}
	frame := FrameLayout{Cells: 2, Slots: []FrameSlot{{Offset: 0, Size: 1, Role: "continuation"}, {Offset: 1, Size: 1, Role: "parameter", Operand: &x}}}
	p := &Program{Procedures: []Procedure{{Label: "s1.f", ScopeID: 2, Frame: frame}}, Environments: []EnvironmentLayout{{ScopeID: 2, ParentID: 1, Bindings: []Operand{x}}}}
	scopes := model.ScopeSnapshots{{ID: 2, Symbols: model.Symbols{{Name: "x", Kind: model.SymbolParameter}}}}
	before, _ := json.Marshal([]any{p, scopes})
	r := BuildCodegenReport(p, scopes)
	want := []StorageLocation{{Space: "frame:s1.f", Offset: 1, Size: 1, Role: "parameter"}, {Space: "environment:s2", Offset: 0, Size: 1, Role: "binding"}}
	if !reflect.DeepEqual(r.Symbols[0].Locations, want) || !reflect.DeepEqual(r.Spaces[0].Slots, frame.Slots) || r.Spaces[1].Parent != "environment:s1" {
		t.Fatalf("staging/retention layout: %+v", r)
	}
	after, _ := json.Marshal([]any{p, scopes})
	if !bytes.Equal(before, after) {
		t.Fatal("report mutated frame/environment inputs")
	}
}

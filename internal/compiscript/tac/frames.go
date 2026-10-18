package tac

import (
	"fmt"
	"sort"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// FrameLayout uses abstract cells, not byte offsets or a target-machine ABI.
type FrameLayout struct {
	Cells int         `json:"cells"`
	Slots []FrameSlot `json:"slots"`
}

type FrameSlot struct {
	Offset  int      `json:"offset"`
	Size    int      `json:"size"`
	Role    string   `json:"role"`
	Operand *Operand `json:"operand,omitempty"`
	Dynamic bool     `json:"dynamic,omitempty"`
}

type Procedure struct {
	Label           string         `json:"label"`
	ScopeID         int            `json:"scopeId"`
	Span            ast.Span       `json:"span"`
	Parameters      []Operand      `json:"parameters"`
	ResultKind      model.TypeKind `json:"resultKind"`
	Frame           FrameLayout    `json:"frame"`
	Instructions    []Instruction  `json:"instructions"`
	PeakTemporaries int            `json:"peakTemporaries"`
	Globals         []Operand      `json:"globals,omitempty"`
	LexicalScope    int            `json:"lexicalScope,omitempty"`
	Captures        []Operand      `json:"captures,omitempty"`
	Receiver        *Operand       `json:"receiver,omitempty"`
	ResultClass     string         `json:"resultClass,omitempty"`
	ResultShape     string         `json:"resultShape,omitempty"`
}

func symbolOperand(scope int, symbol model.Symbol) Operand {
	return Operand{Kind: Symbol, Value: fmt.Sprintf("s%d.%s", scope, symbol.Name), Type: symbol.Type.Kind}
}

func (l *lowerer) layout(p *Procedure) bool {
	add := func(role string, operand *Operand, dynamic bool) {
		p.Frame.Slots = append(p.Frame.Slots, FrameSlot{Offset: p.Frame.Cells, Size: 1, Role: role, Operand: operand, Dynamic: dynamic})
		p.Frame.Cells++
	}
	add("continuation", nil, false)
	add("controlLink", nil, false)
	add("globalAccessLink", nil, false)
	if callValue(p.ResultKind) {
		add("result", nil, false)
	}
	if p.Receiver != nil {
		add("parameter", p.Receiver, false)
	}
	for _, parameter := range p.Parameters {
		parameter := parameter
		add("parameter", &parameter, false)
	}
	type local struct {
		operand Operand
		span    ast.Span
	}
	var locals []local
	owned := map[int]bool{p.ScopeID: true}
	// Scope inventories need not arrive in parent-before-child order.
	for changed := true; changed; {
		changed = false
		for _, scope := range l.scopes {
			if owned[scope.ParentID] && !owned[scope.ID] {
				if scope.Kind == model.ScopeFunction {
					continue
				}
				owned[scope.ID] = true
				changed = true
			}
		}
	}
	for _, scope := range l.scopes {
		if !owned[scope.ID] {
			continue
		}
		for _, symbol := range scope.Symbols {
			if symbol.Kind == model.SymbolClass || symbol.Kind == model.SymbolField || symbol.Kind == model.SymbolMethod {
				continue
			}
			if symbol.Kind == model.SymbolParameter || symbol.Kind == model.SymbolFunction {
				continue
			}
			o := l.storageIdentity(symbolOperand(scope.ID, symbol))
			if !storedValue(o.Type) || (symbol.Kind != model.SymbolVariable && symbol.Kind != model.SymbolConstant) {
				return l.unsupported(symbol.Span, "unsupported function-owned binding")
			}
			if symbol.Type.Kind == model.TypeList {
				o = l.storageIdentity(l.listStorageType(scope.ID, o, symbol.Type))
			}
			if symbol.Type.Kind == model.TypeClass {
				o.Class = l.classForType(scope.ID, symbol.Type.Name)
				o = l.storageIdentity(o)
			}
			locals = append(locals, local{o, symbol.Span})
		}
	}
	sort.Slice(locals, func(i, j int) bool {
		if locals[i].span.Start.Offset != locals[j].span.Start.Offset {
			return locals[i].span.Start.Offset < locals[j].span.Start.Offset
		}
		return locals[i].operand.Value < locals[j].operand.Value
	})
	for _, local := range locals {
		operand := local.operand
		add("local", &operand, false)
	}
	for i := 0; i < p.PeakTemporaries; i++ {
		// Types belong to occurrences, never to a reused temporary cell.
		operand := Operand{Kind: Temporary, Value: fmt.Sprintf("t%d", i)}
		add("temporary", &operand, true)
	}
	return true
}

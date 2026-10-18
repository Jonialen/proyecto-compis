package compiscript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

// Decode the public JSON contract, rather than requiring the implementation type.
type symbolReport struct {
	Unit    string `json:"unit"`
	Symbols []struct {
		ID        string       `json:"id"`
		ScopeID   int          `json:"scopeId"`
		Symbol    model.Symbol `json:"symbol"`
		Locations []struct {
			Space  string `json:"space"`
			Offset int    `json:"offset"`
			Size   int    `json:"size"`
			Role   string `json:"role"`
		} `json:"locations"`
		Labels []string `json:"labels"`
	} `json:"symbols"`
	Spaces []struct {
		ID    string          `json:"id"`
		Kind  string          `json:"kind"`
		Cells int             `json:"cells"`
		Slots []tac.FrameSlot `json:"slots"`
	} `json:"spaces"`
	Labels []struct {
		Name  string   `json:"name"`
		Kind  string   `json:"kind"`
		Owner string   `json:"owner"`
		Span  ast.Span `json:"span"`
	} `json:"labels"`
}

func assertSymbolReport(t *testing.T, report CompilationReport) {
	t.Helper()
	encoded, _ := json.Marshal(report)
	var envelope struct {
		Codegen *symbolReport `json:"codegen"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Codegen == nil {
		t.Fatal("successful compilation must expose codegen report")
	}
	r := envelope.Codegen
	if r.Unit != "abstract-cell" {
		t.Fatalf("unit=%q", r.Unit)
	}
	spaces := map[string]int{}
	for i, space := range r.Spaces {
		if _, duplicate := spaces[space.ID]; duplicate {
			t.Fatalf("duplicate space %s", space.ID)
		}
		spaces[space.ID] = i
		if space.Cells != len(space.Slots) {
			t.Fatalf("space size: %+v", space)
		}
		for n, slot := range space.Slots {
			if slot.Offset != n || slot.Size != 1 {
				t.Fatalf("cell layout: %+v", space)
			}
		}
	}
	checkSlots := func(id string, slots []tac.FrameSlot) {
		t.Helper()
		i, ok := spaces[id]
		if !ok || !reflect.DeepEqual(r.Spaces[i].Slots, slots) {
			t.Fatalf("%s does not reuse actual layout: %+v", id, r.Spaces)
		}
	}
	for _, p := range report.TAC.Procedures {
		checkSlots("frame:"+p.Label, p.Frame.Slots)
	}
	for _, e := range report.TAC.Environments {
		var slots []tac.FrameSlot
		for i, operand := range e.Bindings {
			o := operand
			slots = append(slots, tac.FrameSlot{Offset: i, Size: 1, Role: "binding", Operand: &o})
		}
		if len(slots) > 0 {
			checkSlots(fmt.Sprintf("environment:s%d", e.ScopeID), slots)
		}
	}
	for _, c := range report.TAC.Classes {
		i, ok := spaces["instance:"+c.ID]
		if !ok {
			t.Fatalf("missing class %s", c.ID)
		}
		space := r.Spaces[i]
		if len(space.Slots) != len(c.Fields) {
			t.Fatal("field count")
		}
		for n, f := range c.Fields {
			s := space.Slots[n]
			if s.Offset != f.Offset || s.Operand == nil || s.Operand.Value != f.ID {
				t.Fatalf("inherited field mismatch: %+v vs %+v", s, f)
			}
		}
	}
	labels := map[string]bool{}
	for _, l := range r.Labels {
		labels[l.Owner+":"+l.Name] = true
	}
	for _, c := range report.TAC.Classes {
		if !labels["instance:"+c.ID+":"+c.ID] {
			t.Fatalf("missing class layout label %s", c.ID)
		}
		for _, m := range c.Methods {
			if !labels["instance:"+c.ID+":"+m.Callee] || !labels["frame:"+m.Callee+":"+m.Callee] {
				t.Fatalf("method dispatch label not emitted: %+v", m)
			}
		}
		if !labels["frame:"+c.Initializer+":"+c.Initializer] {
			t.Fatalf("initializer label not emitted: %s", c.Initializer)
		}
	}
	checkInstructions := func(owner string, instructions []tac.Instruction) {
		for _, in := range instructions {
			if in.Op == tac.Label && !labels[owner+":"+in.Label] {
				t.Fatalf("missing emitted control label %s", in.Label)
			}
			for _, o := range symbolInstructionOperands(in) {
				if o.Kind != tac.Temporary && o.Kind != tac.Symbol {
					continue
				}
				found := false
				for _, space := range r.Spaces {
					if o.Kind == tac.Temporary && space.ID != owner {
						continue
					}
					for _, s := range space.Slots {
						if s.Operand != nil && s.Operand.Kind == o.Kind && s.Operand.Value == o.Value {
							found = true
						}
					}
				}
				if !found {
					t.Fatalf("emitted operand has no storage: %s in %s", o, owner)
				}
			}
		}
	}
	checkInstructions("main", report.TAC.Instructions)
	for _, p := range report.TAC.Procedures {
		if !labels["frame:"+p.Label+":"+p.Label] {
			t.Fatalf("missing procedure label %s", p.Label)
		}
		checkInstructions("frame:"+p.Label, p.Instructions)
	}
	wantSymbols := 0
	for _, scope := range report.Scopes {
		for _, symbol := range scope.Symbols {
			wantSymbols++
			id := fmt.Sprintf("s%d.%s", scope.ID, symbol.Name)
			found := false
			for _, entry := range r.Symbols {
				if entry.ID != id {
					continue
				}
				found = true
				if entry.ScopeID != scope.ID || !reflect.DeepEqual(entry.Symbol, symbol) {
					t.Fatalf("source identity lost: %s", id)
				}
				for _, loc := range entry.Locations {
					i, ok := spaces[loc.Space]
					if !ok {
						t.Fatal("unknown location space")
					}
					s := r.Spaces[i].Slots[loc.Offset]
					if s.Operand == nil || s.Operand.Value != id || s.Size != loc.Size || s.Role != loc.Role {
						t.Fatalf("invented location for %s", id)
					}
				}
				// Every retained/staging instance must be represented, not just one.
				for _, space := range r.Spaces {
					for _, slot := range space.Slots {
						if slot.Operand == nil || slot.Operand.Value != id {
							continue
						}
						matched := false
						for _, loc := range entry.Locations {
							if loc.Space == space.ID && loc.Offset == slot.Offset {
								matched = true
							}
						}
						if !matched {
							t.Fatalf("missing location %s in %s", id, space.ID)
						}
					}
				}
				if symbol.Kind == model.SymbolFunction || symbol.Kind == model.SymbolMethod || symbol.Kind == model.SymbolClass {
					if len(entry.Locations) != 0 || len(entry.Labels) == 0 {
						t.Fatalf("non-storage symbol %s: %+v", id, entry)
					}
					for _, label := range entry.Labels {
						if !labels["frame:"+label+":"+label] && !labels["instance:"+label+":"+label] {
							t.Fatalf("symbol linked to invented label %s", label)
						}
					}
				}
			}
			if !found {
				t.Fatalf("missing source symbol %s", id)
			}
		}
	}
	if len(r.Symbols) != wantSymbols {
		t.Fatalf("symbol count=%d want=%d", len(r.Symbols), wantSymbols)
	}
}

func symbolInstructionOperands(i tac.Instruction) []tac.Operand {
	out := []tac.Operand{i.Target, i.Left}
	for _, p := range []*tac.Operand{i.Right, i.ReturnValue, i.Receiver, i.Index} {
		if p != nil {
			out = append(out, *p)
		}
	}
	return append(out, i.Arguments...)
}

func TestCompilationSymbols(t *testing.T) {
	for _, name := range []string{"numeric", "conditions", "loops", "switch", "functions", "recursion", "closures", "objects", "collections"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("../../testdata/compiscript/tac/" + name + ".cps")
			if err != nil {
				t.Fatal(err)
			}
			before := append([]byte(nil), source...)
			program, _ := frontend.Parse(source)
			scopes, _ := semantic.Analyze(program)
			immutable, _ := json.Marshal([]any{program, scopes})
			generated, diagnostics := tac.Generate(program, scopes)
			if len(diagnostics) != 0 {
				t.Fatal(diagnostics)
			}
			report := Compile(source)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatal(report.Diagnostics)
			}
			assertSymbolReport(t, report)
			actual, _ := json.Marshal(report.TAC)
			want, _ := json.Marshal(generated)
			if !bytes.Equal(actual, want) || report.TAC.String() != generated.String() {
				t.Fatal("Generate/TAC changed")
			}
			first, _ := json.Marshal(report)
			second, _ := json.Marshal(Compile(source))
			after, _ := json.Marshal([]any{program, scopes})
			if !bytes.Equal(first, second) || !bytes.Equal(source, before) || !bytes.Equal(immutable, after) || !reflect.DeepEqual(report.Scopes, Analyze(source).Scopes) {
				t.Fatal("determinism or immutability")
			}
		})
	}
	assertSymbolReport(t, Compile([]byte(`const x=2; {let x=3; x=x+1;} let y=x+4;`)))
}

func TestCompilationSymbolsAtomicity(t *testing.T) {
	for _, source := range []string{`let x=;`, `let x=missing;`, `let x=1; print(x);`} {
		r := Compile([]byte(source))
		encoded, _ := json.Marshal(r)
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(encoded, &fields)
		if r.TAC != nil || len(r.Diagnostics) == 0 || fields["codegen"] != nil {
			t.Fatalf("partial report: %s", encoded)
		}
		if !reflect.DeepEqual(r.Scopes, Analyze([]byte(source)).Scopes) {
			t.Fatal("rejected scopes changed")
		}
	}
}

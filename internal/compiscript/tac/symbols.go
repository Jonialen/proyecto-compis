package tac

import (
	"fmt"
	"sort"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// CodegenReport describes static layouts, never live addresses or heap identities.
// All values (including references) occupy one abstract cell. Labels occupy none.
type CodegenReport struct {
	Unit    string          `json:"unit"`
	Symbols []CodegenSymbol `json:"symbols"`
	Spaces  []StorageSpace  `json:"spaces"`
	Labels  []CodegenLabel  `json:"labels"`
}

type CodegenSymbol struct {
	ID        string            `json:"id"`
	ScopeID   int               `json:"scopeId"`
	Symbol    model.Symbol      `json:"symbol"`
	Locations []StorageLocation `json:"locations"`
	Labels    []string          `json:"labels"`
}

type StorageLocation struct {
	Space  string `json:"space"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
	Role   string `json:"role"`
}

type StorageSpace struct {
	ID      string      `json:"id"`
	Kind    string      `json:"kind"`
	ScopeID int         `json:"scopeId,omitempty"`
	Parent  string      `json:"parent,omitempty"`
	Cells   int         `json:"cells"`
	Slots   []FrameSlot `json:"slots"`
}

type CodegenLabel struct {
	Name  string   `json:"name"`
	Kind  string   `json:"kind"`
	Owner string   `json:"owner"`
	Span  ast.Span `json:"span"`
}

// BuildCodegenReport inventories emitted operands and reuses the authoritative
// frame/environment/class layouts. It neither analyzes source nor changes TAC.
func BuildCodegenReport(p *Program, scopes model.ScopeSnapshots) *CodegenReport {
	if p == nil {
		return nil
	}
	r := &CodegenReport{Unit: "abstract-cell", Symbols: []CodegenSymbol{}, Spaces: []StorageSpace{}, Labels: []CodegenLabel{}}
	locations := map[string][]StorageLocation{}
	labels := map[string][]string{}
	allocated := map[string]bool{}
	addSpace := func(s StorageSpace) {
		if s.Slots == nil {
			s.Slots = []FrameSlot{}
		}
		r.Spaces = append(r.Spaces, s)
		for _, slot := range s.Slots {
			if slot.Operand == nil || slot.Operand.Kind != Symbol {
				continue
			}
			id := slot.Operand.Value
			allocated[id] = true
			locations[id] = append(locations[id], StorageLocation{Space: s.ID, Offset: slot.Offset, Size: slot.Size, Role: slot.Role})
		}
	}
	for _, procedure := range p.Procedures {
		addSpace(StorageSpace{ID: "frame:" + procedure.Label, Kind: "frame", ScopeID: procedure.ScopeID, Cells: procedure.Frame.Cells, Slots: append([]FrameSlot(nil), procedure.Frame.Slots...)})
		labels[procedure.Label] = append(labels[procedure.Label], procedure.Label)
		r.Labels = append(r.Labels, CodegenLabel{Name: procedure.Label, Kind: "procedure", Owner: "frame:" + procedure.Label, Span: procedure.Span})
	}
	for _, env := range p.Environments {
		s := StorageSpace{ID: fmt.Sprintf("environment:s%d", env.ScopeID), Kind: "environment", ScopeID: env.ScopeID, Cells: len(env.Bindings)}
		if env.ParentID != 0 {
			s.Parent = fmt.Sprintf("environment:s%d", env.ParentID)
		}
		for i, operand := range env.Bindings {
			o := operand
			s.Slots = append(s.Slots, FrameSlot{Offset: i, Size: 1, Role: "binding", Operand: &o})
		}
		addSpace(s)
	}
	for _, class := range p.Classes {
		s := StorageSpace{ID: "instance:" + class.ID, Kind: "instance", ScopeID: class.ScopeID, Cells: len(class.Fields)}
		if class.Parent != "" {
			s.Parent = "instance:" + class.Parent
		}
		for _, field := range class.Fields {
			o := Operand{Kind: Symbol, Value: field.ID, Type: field.Type.Kind, Class: field.Class, Shape: field.Shape}
			s.Slots = append(s.Slots, FrameSlot{Offset: field.Offset, Size: 1, Role: "field", Operand: &o})
		}
		addSpace(s)
		labels[class.ID] = []string{class.ID, class.Initializer}
		r.Labels = append(r.Labels, CodegenLabel{Name: class.ID, Kind: "classLayout", Owner: s.ID, Span: class.Span})
		for _, method := range class.Methods {
			r.Labels = append(r.Labels, CodegenLabel{Name: method.Callee, Kind: "method", Owner: s.ID, Span: method.Span})
		}
	}
	// Global cells are the emitted symbols not already owned by a frame, lexical
	// environment or instance. Main block bindings retain their semantic identity.
	globals := map[string]Operand{}
	mainTemps := map[string]bool{}
	scan := func(owner string, instructions []Instruction) {
		for _, instruction := range instructions {
			if instruction.Op == Label {
				r.Labels = append(r.Labels, CodegenLabel{Name: instruction.Label, Kind: "control", Owner: owner, Span: instruction.Span})
			}
			operands := []Operand{instruction.Target, instruction.Left}
			for _, o := range []*Operand{instruction.Right, instruction.ReturnValue, instruction.Receiver, instruction.Index} {
				if o != nil {
					operands = append(operands, *o)
				}
			}
			operands = append(operands, instruction.Arguments...)
			for _, o := range operands {
				if o.Kind == Symbol && !allocated[o.Value] {
					globals[o.Value] = o
				}
				if owner == "main" && o.Kind == Temporary {
					mainTemps[o.Value] = true
				}
			}
		}
	}
	scan("main", p.Instructions)
	for _, procedure := range p.Procedures {
		scan("frame:"+procedure.Label, procedure.Instructions)
		for _, o := range procedure.Globals {
			if !allocated[o.Value] {
				globals[o.Value] = o
			}
		}
	}
	global := StorageSpace{ID: "globals", Kind: "global"}
	var ids []string
	for id := range globals {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for i, id := range ids {
		o := globals[id]
		global.Slots = append(global.Slots, FrameSlot{Offset: i, Size: 1, Role: "binding", Operand: &o})
	}
	global.Cells = len(global.Slots)
	addSpace(global)
	main := StorageSpace{ID: "main", Kind: "main"}
	// Temporary numbering comes from the emitter; a cell has no permanent type.
	for i := 0; i < p.PeakTemporaries; i++ {
		id := fmt.Sprintf("t%d", i)
		if mainTemps[id] {
			o := Operand{Kind: Temporary, Value: id}
			main.Slots = append(main.Slots, FrameSlot{Offset: len(main.Slots), Size: 1, Role: "temporary", Operand: &o, Dynamic: true})
		}
	}
	main.Cells = len(main.Slots)
	addSpace(main)
	for _, scope := range scopes {
		for _, symbol := range scope.Symbols {
			id := symbolOperand(scope.ID, symbol).Value
			entry := CodegenSymbol{ID: id, ScopeID: scope.ID, Symbol: symbol, Locations: []StorageLocation{}, Labels: []string{}}
			entry.Locations = append(entry.Locations, locations[id]...)
			entry.Labels = append(entry.Labels, labels[id]...)
			r.Symbols = append(r.Symbols, entry)
		}
	}
	sort.Slice(r.Symbols, func(i, j int) bool { return r.Symbols[i].ID < r.Symbols[j].ID })
	return r
}

package compiscript

import (
	"fmt"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

type testEnvironment struct {
	layout tac.EnvironmentLayout
	parent *testEnvironment
	cells  map[string]frameValue
}

type testClosure struct {
	procedure   *tac.Procedure
	environment *testEnvironment
	receiver    *testObject
}

func testOperandScope(o tac.Operand) int {
	if o.Kind != tac.Symbol {
		return 0
	}
	id, _ := strconv.Atoi(strings.TrimPrefix(strings.SplitN(o.Value, ".", 2)[0], "s"))
	return id
}

func (f *testFrame) loadEnvironments(layouts []tac.EnvironmentLayout) error {
	f.layouts = map[int]tac.EnvironmentLayout{}
	f.environments = map[int]*testEnvironment{}
	for _, layout := range layouts {
		if layout.ScopeID <= 0 {
			return fmt.Errorf("invalid environment scope")
		}
		if _, ok := f.layouts[layout.ScopeID]; ok {
			return fmt.Errorf("duplicate environment layout")
		}
		seen := map[string]bool{}
		for _, o := range layout.Bindings {
			if testOperandScope(o) != layout.ScopeID || seen[o.Value] {
				return fmt.Errorf("invalid environment binding")
			}
			seen[o.Value] = true
		}
		f.layouts[layout.ScopeID] = layout
	}
	for _, layout := range layouts {
		seen := map[int]bool{}
		for id := layout.ScopeID; id != 0; {
			if seen[id] {
				return fmt.Errorf("cyclic environment layout")
			}
			seen[id] = true
			parent, ok := f.layouts[id]
			if !ok {
				return fmt.Errorf("undefined parent environment layout")
			}
			id = parent.ParentID
		}
	}
	return nil
}

func (f *testFrame) findEnvironment(id int) *testEnvironment {
	if env := f.environments[id]; env != nil {
		return env
	}
	for env := f.lexical; env != nil; env = env.parent {
		if env.layout.ScopeID == id {
			return env
		}
	}
	return nil
}

func environmentBinding(env *testEnvironment, o tac.Operand) bool {
	for _, binding := range env.layout.Bindings {
		if binding == o {
			return true
		}
	}
	return false
}

func (f *testFrame) enterTestEnvironment(id int) error {
	layout, ok := f.layouts[id]
	if !ok {
		return fmt.Errorf("undefined environment layout")
	}
	parent := f.findEnvironment(layout.ParentID)
	if layout.ParentID != 0 && parent == nil {
		return fmt.Errorf("undefined parent environment")
	}
	// A function may enter its own scope and descendants, never a foreign scope.
	if f.procedure != nil {
		owned := false
		for scope := id; scope != 0; scope = f.layouts[scope].ParentID {
			if scope == f.procedure.ScopeID {
				owned = true
				break
			}
		}
		if !owned {
			return fmt.Errorf("incorrect environment ownership")
		}
	}
	env := &testEnvironment{layout: layout, parent: parent, cells: map[string]frameValue{}}
	if layout.ParentID == 0 {
		env.cells = f.globals
	}
	if f.procedure != nil && id == f.procedure.ScopeID {
		for _, o := range f.procedure.Parameters {
			offset, err := f.ownedOffset(o)
			if err != nil {
				return err
			}
			value, ok := f.cells[offset].(frameValue)
			if !ok || !environmentBinding(env, o) {
				return fmt.Errorf("undefined environment parameter")
			}
			env.cells[o.Value] = value
		}
		if o := f.procedure.Receiver; o != nil {
			offset, err := f.ownedOffset(*o)
			if err != nil {
				return err
			}
			value, ok := f.cells[offset].(frameValue)
			if !ok || !environmentBinding(env, *o) {
				return fmt.Errorf("undefined receiver parameter")
			}
			env.cells[o.Value] = value
		}
	}
	f.environments[id] = env
	return nil
}

func (f *testFrame) readEnvironment(o tac.Operand) (bool, frameValue, error) {
	if o.Kind != tac.Symbol || len(f.layouts) == 0 {
		return false, frameValue{}, nil
	}
	env := f.findEnvironment(testOperandScope(o))
	if env == nil {
		return true, frameValue{}, fmt.Errorf("undefined operand environment: %s", o)
	}
	if !environmentBinding(env, o) {
		return true, frameValue{}, fmt.Errorf("environment binding out of bounds: %s", o)
	}
	v, ok := env.cells[o.Value]
	if !ok {
		return true, v, fmt.Errorf("undefined environment cell: %s", o)
	}
	if v.kind != o.Type {
		return true, v, fmt.Errorf("environment type mismatch")
	}
	if err := f.checkObjectType(o, v); err != nil {
		return true, v, err
	}
	return true, v, nil
}

func (f *testFrame) writeEnvironment(o tac.Operand, v frameValue) (bool, error) {
	if o.Kind != tac.Symbol || len(f.layouts) == 0 {
		return false, nil
	}
	// Formal arguments are staged in existing frame cells before the explicit
	// function-scope entry moves them into retained lexical cells.
	if f.procedure != nil && f.pc == 0 && f.findEnvironment(f.procedure.ScopeID) == nil {
		if f.procedure.Receiver != nil && *f.procedure.Receiver == o {
			return false, nil
		}
		for _, parameter := range f.procedure.Parameters {
			if parameter == o {
				return false, nil
			}
		}
	}
	env := f.findEnvironment(testOperandScope(o))
	if env == nil {
		return true, fmt.Errorf("undefined destination environment")
	}
	if !environmentBinding(env, o) {
		return true, fmt.Errorf("environment binding out of bounds: %s", o)
	}
	env.cells[o.Value] = v
	return true, nil
}

func checkClosureSignature(i tac.Instruction, p *tac.Procedure) error {
	s := i.Signature
	if s == nil || s.Kind != model.TypeFunction || s.Result == nil || s.Result.Kind != p.ResultKind || len(s.Params) != len(p.Parameters) {
		return fmt.Errorf("indirect call signature mismatch")
	}
	for j, parameter := range p.Parameters {
		if parameter.Type != s.Params[j].Kind || parameter.Shape != testListShape(s.Params[j]) {
			return fmt.Errorf("indirect call signature mismatch")
		}
	}
	if p.ResultShape != testListShape(*s.Result) {
		return fmt.Errorf("indirect call signature mismatch")
	}
	if len(i.TypeClasses) != 0 {
		if len(i.TypeClasses) != len(p.Parameters)+1 {
			return fmt.Errorf("indirect call signature mismatch")
		}
		for j, parameter := range p.Parameters {
			if i.TypeClasses[j] != parameter.Class {
				return fmt.Errorf("indirect call signature mismatch")
			}
		}
		if i.TypeClasses[len(p.Parameters)] != p.ResultClass {
			return fmt.Errorf("indirect call signature mismatch")
		}
	} else {
		if p.ResultClass != "" {
			return fmt.Errorf("indirect call signature mismatch")
		}
		for _, parameter := range p.Parameters {
			if parameter.Class != "" {
				return fmt.Errorf("indirect call signature mismatch")
			}
		}
	}
	if (p.ResultKind == model.TypeNull && i.Target.Kind != "") || (p.ResultKind != model.TypeNull && i.Target.Type != p.ResultKind) {
		return fmt.Errorf("indirect call result mismatch")
	}
	return nil
}

func (f *testFrame) checkCallRequirements(p *tac.Procedure, lexical *testEnvironment) error {
	if p.LexicalScope != 0 && (lexical == nil || lexical.layout.ScopeID != p.LexicalScope) {
		return fmt.Errorf("invalid call lexical environment")
	}
	for _, o := range p.Globals {
		v, ok := f.globals[o.Value]
		if !ok || v.kind != o.Type {
			return fmt.Errorf("call requires initialized global binding %s", o.Value)
		}
	}
	for _, o := range p.Captures {
		var env *testEnvironment
		for e := lexical; e != nil; e = e.parent {
			if e.layout.ScopeID == testOperandScope(o) {
				env = e
				break
			}
		}
		if env == nil || !environmentBinding(env, o) {
			return fmt.Errorf("undefined capture environment")
		}
		v, ok := env.cells[o.Value]
		if !ok || v.kind != o.Type {
			return fmt.Errorf("call requires initialized captured binding %s", o.Value)
		}
	}
	return nil
}

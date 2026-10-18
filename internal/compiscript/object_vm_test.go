package compiscript

import (
	"fmt"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

type testObject struct {
	layout       *tac.ClassLayout
	fields       map[int]frameValue
	environments map[int]*testEnvironment
}

func (f *testFrame) classIs(actual, expected string) bool {
	seen := map[string]bool{}
	for actual != "" && !seen[actual] {
		if actual == expected {
			return true
		}
		seen[actual] = true
		c := f.classes[actual]
		if c == nil {
			return false
		}
		actual = c.Parent
	}
	return false
}
func (f *testFrame) checkObjectType(o tac.Operand, v frameValue) error {
	if o.Type == model.TypeList && v.value != nil {
		list, ok := v.value.(*testList)
		if !ok || list == nil || !f.listTypeFits(o, list.typeOf) {
			return fmt.Errorf("list type mismatch")
		}
		return nil
	}
	if o.Type != model.TypeClass || v.value == nil {
		return nil
	}
	object, ok := v.value.(*testObject)
	if !ok || object == nil || object.layout == nil || !f.classIs(object.layout.ID, o.Class) {
		return fmt.Errorf("nominal object type mismatch")
	}
	return nil
}

func (f *testFrame) loadClasses(layouts []tac.ClassLayout, procedures map[string]*tac.Procedure) error {
	f.classes = map[string]*tac.ClassLayout{}
	for i := range layouts {
		c := &layouts[i]
		if c.ID == "" || f.classes[c.ID] != nil {
			return fmt.Errorf("invalid class identity")
		}
		f.classes[c.ID] = c
	}
	for _, c := range layouts {
		seen := map[string]bool{}
		for id := c.ID; id != ""; {
			if seen[id] {
				return fmt.Errorf("cyclic class layout")
			}
			seen[id] = true
			parent := f.classes[id]
			if parent == nil {
				return fmt.Errorf("undefined base class")
			}
			id = parent.Parent
		}
		init := procedures[c.Initializer]
		if init == nil || init.Receiver == nil || init.Receiver.Class != c.ID || init.LexicalScope != c.LexicalScope {
			return fmt.Errorf("undefined class initializer")
		}
		fields := map[string]bool{}
		for offset, field := range c.Fields {
			if field.Offset != offset || field.ID == "" || fields[field.ID] || !f.classIs(c.ID, field.Owner) {
				return fmt.Errorf("invalid field offset/owner")
			}
			fields[field.ID] = true
		}
		if c.Parent != "" {
			parent := f.classes[c.Parent]
			if len(parent.Fields) > len(c.Fields) {
				return fmt.Errorf("invalid inherited fields")
			}
			for i, field := range parent.Fields {
				if field.ID != c.Fields[i].ID || field.Offset != c.Fields[i].Offset {
					return fmt.Errorf("invalid inherited field offset")
				}
			}
		}
		methods := map[string]bool{}
		for _, method := range c.Methods {
			p := procedures[method.Callee]
			if methods[method.Name] || p == nil || p.Receiver == nil || p.Receiver.Class != method.Owner || !f.classIs(c.ID, method.Owner) {
				return fmt.Errorf("invalid method metadata")
			}
			methods[method.Name] = true
		}
	}
	return nil
}

func (f *testFrame) objectInstruction(i tac.Instruction, procedures map[string]*tac.Procedure) error {
	if i.Op == tac.NewObject {
		c := f.classes[i.Class]
		if c == nil {
			return fmt.Errorf("undefined object class")
		}
		object := &testObject{layout: c, fields: map[int]frameValue{}, environments: map[int]*testEnvironment{}}
		for current := c; current != nil; current = f.classes[current.Parent] {
			env := f.findEnvironment(current.LexicalScope)
			if env == nil {
				return fmt.Errorf("undefined class lexical environment")
			}
			object.environments[current.LexicalScope] = env
		}
		return f.write(i.Target, frameValue{model.TypeClass, object})
	}
	v, err := f.read(i.Left)
	if err != nil {
		return err
	}
	if v.value == nil {
		return fmt.Errorf("null receiver")
	}
	object, ok := v.value.(*testObject)
	if !ok || v.kind != model.TypeClass || !f.classIs(object.layout.ID, i.Class) {
		return fmt.Errorf("invalid object receiver")
	}
	if i.Op == tac.BindMethod {
		label := i.Callee
		if label == "" {
			for _, method := range object.layout.Methods {
				if method.Name == i.Member {
					label = method.Callee
					break
				}
			}
		} else {
			c := f.classes[i.Class]
			if c == nil || label != c.Initializer {
				return fmt.Errorf("invalid initializer selection")
			}
		}
		p := procedures[label]
		if p == nil || p.Receiver == nil {
			return fmt.Errorf("undefined object method")
		}
		env := object.environments[p.LexicalScope]
		if env == nil {
			return fmt.Errorf("undefined method environment")
		}
		return f.write(i.Target, frameValue{model.TypeFunction, testClosure{procedure: p, environment: env, receiver: object}})
	}
	c := f.classes[i.Class]
	if c == nil || i.Offset == nil || *i.Offset < 0 || *i.Offset >= len(c.Fields) || *i.Offset >= len(object.layout.Fields) {
		return fmt.Errorf("field offset out of bounds")
	}
	field := c.Fields[*i.Offset]
	if field.ID != i.Member || object.layout.Fields[*i.Offset].ID != field.ID {
		return fmt.Errorf("invalid field identity")
	}
	if i.Op == tac.LoadField {
		value, ok := object.fields[*i.Offset]
		if !ok {
			return fmt.Errorf("undefined field %s", field.ID)
		}
		return f.write(i.Target, value)
	}
	if i.Right == nil {
		return fmt.Errorf("missing field value")
	}
	value, err := f.read(*i.Right)
	if err != nil {
		return err
	}
	if !field.Mutable {
		if f.procedure == nil || f.procedure.Label != f.classes[field.Owner].Initializer {
			return fmt.Errorf("immutable field")
		}
		if _, set := object.fields[*i.Offset]; set {
			return fmt.Errorf("immutable field already initialized")
		}
	}
	if value.kind == model.TypeNull && (field.Type.Kind == model.TypeClass || field.Type.Kind == model.TypeString || field.Type.Kind == model.TypeList) {
		value.kind = field.Type.Kind
	}
	if value.kind != field.Type.Kind {
		return fmt.Errorf("field type mismatch")
	}
	if err := f.checkObjectType(tac.Operand{Type: field.Type.Kind, Class: field.Class, Shape: field.Shape}, value); err != nil {
		return err
	}
	object.fields[*i.Offset] = value
	return nil
}

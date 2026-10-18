package compiscript

import (
	"fmt"
	"math"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// Each dynamic allocation owns a distinct reference, including empty lists.
// This bounded test oracle is not a product interpreter.
type testList struct {
	typeOf tac.Operand
	values []frameValue
}

func testListElement(o tac.Operand) tac.Operand {
	kind, rest, _ := strings.Cut(o.Shape, "/")
	return tac.Operand{Type: model.TypeKind(kind), Shape: rest, Class: o.Class}
}

func testListShape(t model.Type) string {
	if t.Kind != model.TypeList || t.Element == nil {
		return ""
	}
	s := string(t.Element.Kind)
	if child := testListShape(*t.Element); child != "" {
		s += "/" + child
	}
	return s
}

func (f *testFrame) listTypeFits(expected, actual tac.Operand) bool {
	if expected.Type == model.TypeList && actual.Type == model.TypeList {
		return expected.Shape != "" && actual.Shape != "" && f.listTypeFits(testListElement(expected), testListElement(actual))
	}
	if expected.Type == model.TypeFloat && actual.Type == model.TypeInteger {
		return true
	}
	if expected.Type != actual.Type {
		return false
	}
	if expected.Type == model.TypeClass {
		return f.classIs(actual.Class, expected.Class)
	}
	return true
}

func (f *testFrame) collectionValue(expected tac.Operand, value frameValue) (frameValue, error) {
	if value.kind == model.TypeNull && (expected.Type == model.TypeClass || expected.Type == model.TypeString || expected.Type == model.TypeList) {
		value.kind = expected.Type
	}
	if expected.Type == model.TypeFloat && value.kind == model.TypeInteger {
		value.kind = model.TypeFloat
	}
	if value.kind != expected.Type {
		return value, fmt.Errorf("list element type mismatch")
	}
	if err := f.checkObjectType(expected, value); err != nil {
		return value, err
	}
	return value, nil
}

func (f *testFrame) collectionInstruction(i tac.Instruction) error {
	if i.Op == tac.NewList {
		if i.Target.Type != model.TypeList || i.Target.Shape == "" {
			return fmt.Errorf("invalid list allocation type")
		}
		values := make([]frameValue, len(i.Arguments))
		for j, arg := range i.Arguments {
			value, err := f.read(arg)
			if err != nil {
				return err
			}
			values[j], err = f.collectionValue(testListElement(i.Target), value)
			if err != nil {
				return err
			}
		}
		return f.write(i.Target, frameValue{model.TypeList, &testList{typeOf: i.Target, values: values}})
	}
	value, err := f.read(i.Left)
	if err != nil {
		return err
	}
	if value.value == nil {
		return fmt.Errorf("null list")
	}
	list, ok := value.value.(*testList)
	if !ok || list == nil || value.kind != model.TypeList {
		return fmt.Errorf("invalid list reference")
	}
	if i.Op == tac.ListLength {
		return f.write(i.Target, frameValue{model.TypeInteger, float64(len(list.values))})
	}
	if i.Index == nil {
		return fmt.Errorf("missing list index")
	}
	index, err := f.read(*i.Index)
	if err != nil {
		return err
	}
	n, ok := index.value.(float64)
	if index.kind != model.TypeInteger || !ok || math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n {
		return fmt.Errorf("invalid list index")
	}
	if n < 0 || n >= float64(len(list.values)) {
		return fmt.Errorf("list index out of bounds")
	}
	if i.Op == tac.LoadIndex {
		v, err := f.collectionValue(i.Target, list.values[int(n)])
		if err != nil {
			return err
		}
		return f.write(i.Target, v)
	}
	if i.Right == nil {
		return fmt.Errorf("missing list value")
	}
	v, err := f.read(*i.Right)
	if err != nil {
		return err
	}
	v, err = f.collectionValue(testListElement(i.Left), v)
	if err != nil {
		return err
	}
	list.values[int(n)] = v
	return nil
}

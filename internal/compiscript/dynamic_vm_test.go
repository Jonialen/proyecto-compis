package compiscript

import (
	"fmt"
	"math"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

func (f *testFrame) checkedFrameValue(expected tac.Operand, v frameValue) (frameValue, error) {
	if expected.Type == model.TypeError {
		return v, nil
	}
	// A typed reference view of null remains the null value, not an instance of
	// its annotation. In particular it may cross an unknown null-result boundary.
	if expected.Type == model.TypeNull && v.value == nil && (v.kind == model.TypeString || v.kind == model.TypeClass || v.kind == model.TypeList) {
		v.kind = model.TypeNull
	}
	if expected.Type == model.TypeFloat && v.kind == model.TypeInteger {
		v.kind = model.TypeFloat
	}
	if v.kind == model.TypeNull && (expected.Type == model.TypeClass || expected.Type == model.TypeList || expected.Type == model.TypeString) {
		v.kind = expected.Type
	}
	if v.kind != expected.Type {
		return v, languageFaultf("dynamic value type mismatch: expected %s, got %s", expected.Type, v.kind)
	}
	if err := f.checkObjectType(expected, v); err != nil {
		return v, err
	}
	return v, nil
}

func (f *testFrame) dynamicMember(i tac.Instruction, procedures map[string]*tac.Procedure) error {
	v, err := f.read(i.Left)
	if err != nil {
		return err
	}
	if v.value == nil {
		return languageFaultf("null receiver")
	}
	object, ok := v.value.(*testObject)
	if v.kind != model.TypeClass || !ok || object == nil || object.layout == nil {
		return languageFaultf("invalid object receiver")
	}
	for _, field := range object.layout.Fields {
		if field.Name != i.Member {
			continue
		}
		if i.Op == tac.DynamicMember {
			value, exists := object.fields[field.Offset]
			if !exists {
				return languageFaultf("undefined field %s", field.ID)
			}
			return f.write(i.Target, value)
		}
		if i.Right == nil {
			return fmt.Errorf("missing field value")
		}
		if !field.Mutable {
			return languageFaultf("immutable field")
		}
		value, err := f.read(*i.Right)
		if err != nil {
			return err
		}
		value, err = f.checkedFrameValue(tac.Operand{Type: field.Type.Kind, Class: field.Class, Shape: field.Shape}, value)
		if err != nil {
			return err
		}
		object.fields[field.Offset] = value
		return f.write(i.Target, value)
	}
	if i.Op == tac.DynamicMember {
		for _, method := range object.layout.Methods {
			if method.Name != i.Member {
				continue
			}
			p := procedures[method.Callee]
			if p == nil || object.environments[p.LexicalScope] == nil {
				return fmt.Errorf("undefined object method")
			}
			return f.write(i.Target, frameValue{model.TypeFunction, testClosure{procedure: p, environment: object.environments[p.LexicalScope], receiver: object}})
		}
	}
	return languageFaultf("missing object member %s", i.Member)
}

// Reject malformed instruction contracts before executing any source handler.
// Runtime tagged checks concern language values, not an escape from IR validity.
func validateTestInstruction(i tac.Instruction) error {
	numeric := func(k model.TypeKind) bool { return k == model.TypeInteger || k == model.TypeFloat }
	switch i.Op {
	case tac.Copy, tac.CheckValue, tac.Declare, tac.Print, tac.Label, tac.Jump, tac.BranchTrue, tac.Call, tac.Return, tac.EnterEnvironment, tac.MakeClosure, tac.NewObject, tac.LoadField, tac.StoreField, tac.BindMethod, tac.NewList, tac.ListLength, tac.LoadIndex, tac.StoreIndex, tac.DynamicMember, tac.DynamicStore, tac.PushHandler, tac.PopHandler, tac.CatchException:
		return nil
	case tac.CallIndirect:
		if i.Guarded && (i.Signature != nil || i.Target.Type != model.TypeError) {
			return fmt.Errorf("invalid guarded call metadata")
		}
		return nil
	case tac.Negate, tac.IntegerToFloat, tac.LogicalNot:
		if i.Right != nil {
			return fmt.Errorf("invalid unary instruction")
		}
		if i.Guarded {
			return nil
		}
		if i.Op == tac.LogicalNot && i.Left.Type == model.TypeBoolean && i.Target.Type == model.TypeBoolean {
			return nil
		}
		if i.Op == tac.IntegerToFloat && i.Left.Type == model.TypeInteger && i.Target.Type == model.TypeFloat {
			return nil
		}
		if i.Op == tac.Negate && numeric(i.Left.Type) && i.Left.Type == i.Target.Type {
			return nil
		}
	case tac.Add, tac.Subtract, tac.Multiply, tac.Divide, tac.Remainder, tac.Equal, tac.NotEqual, tac.Less, tac.LessEqual, tac.Greater, tac.GreaterEqual:
		if i.Right == nil {
			return fmt.Errorf("missing binary operand")
		}
		if i.Guarded {
			return nil
		}
		if i.Op == tac.Equal || i.Op == tac.NotEqual {
			if i.Target.Type == model.TypeBoolean {
				return nil
			}
		} else if i.Op == tac.Add && i.Left.Type == model.TypeString && i.Right.Type == model.TypeString && i.Target.Type == model.TypeString {
			return nil
		} else if numeric(i.Left.Type) && i.Left.Type == i.Right.Type {
			comparison := i.Op == tac.Less || i.Op == tac.LessEqual || i.Op == tac.Greater || i.Op == tac.GreaterEqual
			if (comparison && i.Target.Type == model.TypeBoolean) || (!comparison && i.Target.Type == i.Left.Type) {
				return nil
			}
		}
	default:
		return fmt.Errorf("unknown opcode %s", i.Op)
	}
	return fmt.Errorf("invalid instruction type contract: %s", i.Op)
}

func frameReferenceEqual(a, b frameValue) bool {
	if a.value == nil || b.value == nil {
		return a.value == nil && b.value == nil
	}
	if a.kind != b.kind {
		return false
	}
	switch x := a.value.(type) {
	case float64:
		y, ok := b.value.(float64)
		return ok && x == y
	case bool:
		y, ok := b.value.(bool)
		return ok && x == y
	case string:
		y, ok := b.value.(string)
		return ok && x == y
	case *testList:
		y, ok := b.value.(*testList)
		return ok && x == y
	case *testObject:
		y, ok := b.value.(*testObject)
		return ok && x == y
	case testClosure:
		y, ok := b.value.(testClosure)
		return ok && x == y
	case *testException:
		y, ok := b.value.(*testException)
		return ok && x == y
	}
	return false
}

func (f *testFrame) dynamicOperation(i tac.Instruction) error {
	a, err := f.read(i.Left)
	if err != nil {
		return err
	}
	var b frameValue
	if i.Right != nil {
		b, err = f.read(*i.Right)
		if err != nil {
			return err
		}
	}
	result := frameValue{kind: model.TypeBoolean}
	if i.Op == tac.LogicalNot {
		v, ok := a.value.(bool)
		if a.kind != model.TypeBoolean || !ok {
			return languageFaultf("invalid boolean negation")
		}
		result.value = !v
		return f.write(i.Target, result)
	}
	if i.Op == tac.Equal || i.Op == tac.NotEqual {
		if i.Right == nil {
			return fmt.Errorf("missing binary operand")
		}
		if (a.kind == model.TypeFloat || a.kind == model.TypeInteger) && (b.kind == model.TypeFloat || b.kind == model.TypeInteger) {
			a.kind, b.kind = model.TypeFloat, model.TypeFloat
		}
		if a.kind != b.kind && a.value != nil && b.value != nil {
			return languageFaultf("invalid binary operand types")
		}
		result.value = frameReferenceEqual(a, b)
		if i.Op == tac.NotEqual {
			result.value = !result.value.(bool)
		}
		return f.write(i.Target, result)
	}
	if i.Op == tac.Add && a.kind == model.TypeString && b.kind == model.TypeString {
		x, xok := a.value.(string)
		y, yok := b.value.(string)
		if !xok || !yok {
			return languageFaultf("null string operand")
		}
		return f.write(i.Target, frameValue{model.TypeString, x + y})
	}
	x, xok := a.value.(float64)
	if !xok || (a.kind != model.TypeInteger && a.kind != model.TypeFloat) {
		return languageFaultf("invalid numeric operand")
	}
	if i.Op == tac.Negate {
		return f.write(i.Target, frameValue{a.kind, -x})
	}
	if i.Right == nil {
		return fmt.Errorf("missing binary operand")
	}
	y, yok := b.value.(float64)
	if !yok || (b.kind != model.TypeInteger && b.kind != model.TypeFloat) {
		return languageFaultf("invalid numeric operand")
	}
	result.kind = model.TypeInteger
	if a.kind == model.TypeFloat || b.kind == model.TypeFloat || i.Op == tac.Divide {
		result.kind = model.TypeFloat
	}
	switch i.Op {
	case tac.Add:
		result.value = x + y
	case tac.Subtract:
		result.value = x - y
	case tac.Multiply:
		result.value = x * y
	case tac.Divide:
		if y == 0 {
			return languageFaultf("division by zero")
		}
		result.value = x / y
	case tac.Remainder:
		if y == 0 {
			return languageFaultf("remainder by zero")
		}
		result.value = math.Mod(x, y)
	case tac.Less:
		result = frameValue{model.TypeBoolean, x < y}
	case tac.LessEqual:
		result = frameValue{model.TypeBoolean, x <= y}
	case tac.Greater:
		result = frameValue{model.TypeBoolean, x > y}
	case tac.GreaterEqual:
		result = frameValue{model.TypeBoolean, x >= y}
	default:
		return fmt.Errorf("unknown guarded opcode %s", i.Op)
	}
	return f.write(i.Target, result)
}

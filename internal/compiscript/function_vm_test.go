package compiscript

import (
	"fmt"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// This bounded oracle executes abstract cells, not a product runtime or byte ABI.
// Unlike runConditions, it isolates every invocation and checks frame ownership.
type frameValue struct {
	kind  model.TypeKind
	value any
}
type frameContinuation struct {
	caller *testFrame
	pc     int
	target tac.Operand
}
type testFrame struct {
	procedure *tac.Procedure
	tape      []tac.Instruction
	labels    map[string]int
	pc        int
	depth     int
	cells     []any
	offsets   map[string]int
	main      map[string]frameValue
	globals   map[string]frameValue
	mainPeak  int
}

func (f *testFrame) globalStore() (map[string]frameValue, error) {
	if f.procedure == nil {
		return f.globals, nil
	}
	store, ok := f.cells[2].(map[string]frameValue)
	if !ok {
		return nil, fmt.Errorf("broken global access link")
	}
	return store, nil
}

func (f *testFrame) ownedOffset(o tac.Operand) (int, error) {
	offset, owned := f.offsets[string(o.Kind)+":"+o.Value]
	if !owned {
		return 0, fmt.Errorf("incorrect frame ownership: %s", o)
	}
	slot := f.procedure.Frame.Slots[offset]
	if o.Kind == tac.Temporary {
		if slot.Role != "temporary" || !slot.Dynamic {
			return 0, fmt.Errorf("invalid temporary ownership")
		}
	} else if (slot.Role != "parameter" && slot.Role != "local") || slot.Operand.Type != o.Type {
		return 0, fmt.Errorf("invalid local ownership/type")
	}
	return offset, nil
}

func (f *testFrame) mainTemporary(o tac.Operand) error {
	if o.Kind != tac.Temporary {
		return fmt.Errorf("invalid main operand")
	}
	n, err := strconv.Atoi(strings.TrimPrefix(o.Value, "t"))
	if err != nil || o.Value != fmt.Sprintf("t%d", n) || n < 0 || n >= f.mainPeak {
		return fmt.Errorf("incorrect main temporary ownership")
	}
	return nil
}

func frameLabels(tape []tac.Instruction) (map[string]int, error) {
	labels := map[string]int{}
	for pc, i := range tape {
		if i.Op == tac.Label {
			if _, exists := labels[i.Label]; exists || i.Label == "" {
				return nil, fmt.Errorf("duplicate or empty label")
			}
			labels[i.Label] = pc
		}
	}
	for _, i := range tape {
		if i.Op == tac.Jump || i.Op == tac.BranchTrue {
			if _, exists := labels[i.Label]; !exists {
				return nil, fmt.Errorf("undefined label")
			}
		}
	}
	return labels, nil
}

func (f *testFrame) globalOperand(o tac.Operand) bool {
	if f.procedure == nil {
		return o.Kind == tac.Symbol
	}
	for _, global := range f.procedure.Globals {
		if global == o {
			return true
		}
	}
	return false
}

func (f *testFrame) read(o tac.Operand) (frameValue, error) {
	if o.Kind == tac.Literal {
		if o.Type == model.TypeBoolean {
			v, e := strconv.ParseBool(o.Value)
			return frameValue{o.Type, v}, e
		}
		if o.Type == model.TypeInteger || o.Type == model.TypeFloat {
			v, e := strconv.ParseFloat(o.Value, 64)
			return frameValue{o.Type, v}, e
		}
		return frameValue{}, fmt.Errorf("unsupported literal")
	}
	var value frameValue
	var found bool
	if f.globalOperand(o) {
		store, err := f.globalStore()
		if err != nil {
			return value, err
		}
		value, found = store[o.Value]
	} else if f.procedure == nil {
		if err := f.mainTemporary(o); err != nil {
			return value, err
		}
		value, found = f.main[o.Value]
	} else {
		offset, err := f.ownedOffset(o)
		if err != nil {
			return value, err
		}
		value, found = f.cells[offset].(frameValue)
	}
	if !found {
		return value, fmt.Errorf("undefined slot: %s", o)
	}
	if value.kind != o.Type {
		return value, fmt.Errorf("operand type mismatch: %s", o)
	}
	return value, nil
}

func (f *testFrame) write(o tac.Operand, value frameValue) error {
	if (o.Kind != tac.Symbol && o.Kind != tac.Temporary) || o.Type != value.kind {
		return fmt.Errorf("invalid destination: %s", o)
	}
	if f.globalOperand(o) {
		store, err := f.globalStore()
		if err != nil {
			return err
		}
		store[o.Value] = value
		return nil
	}
	if f.procedure == nil {
		if err := f.mainTemporary(o); err != nil {
			return err
		}
		f.main[o.Value] = value
		return nil
	}
	offset, err := f.ownedOffset(o)
	if err != nil {
		return err
	}
	f.cells[offset] = value
	return nil
}

func newTestFrame(p *tac.Procedure, globals map[string]frameValue, continuation frameContinuation) (*testFrame, error) {
	labels, err := frameLabels(p.Instructions)
	if err != nil {
		return nil, err
	}
	f := &testFrame{procedure: p, tape: p.Instructions, labels: labels, cells: make([]any, p.Frame.Cells), offsets: map[string]int{}, globals: globals, depth: continuation.caller.depth + 1}
	if f.depth > 64 {
		return nil, fmt.Errorf("call depth budget exceeded")
	}
	if p.Frame.Cells != len(p.Frame.Slots) || p.Frame.Cells < 3 {
		return nil, fmt.Errorf("invalid frame size")
	}
	for i, slot := range p.Frame.Slots {
		if slot.Offset != i || slot.Size != 1 {
			return nil, fmt.Errorf("invalid cell offset/size")
		}
		if slot.Operand != nil {
			o := *slot.Operand
			key := string(o.Kind) + ":" + o.Value
			if _, exists := f.offsets[key]; exists {
				return nil, fmt.Errorf("duplicate owned operand")
			}
			if slot.Role == "temporary" && (!slot.Dynamic || o.Type != "") {
				return nil, fmt.Errorf("temporary cell must be dynamic")
			}
			f.offsets[key] = i
		}
	}
	for i, role := range []string{"continuation", "controlLink", "globalAccessLink"} {
		if p.Frame.Slots[i].Role != role {
			return nil, fmt.Errorf("invalid control cell")
		}
	}
	f.cells[0], f.cells[1], f.cells[2] = continuation, continuation.caller, globals
	return f, nil
}

func runFunctionFrames(program *tac.Program) (map[string]any, error) {
	globals := map[string]frameValue{}
	values := map[string]any{}
	labels, err := frameLabels(program.Instructions)
	if err != nil {
		return values, err
	}
	f := &testFrame{tape: program.Instructions, labels: labels, main: map[string]frameValue{}, globals: globals, mainPeak: program.PeakTemporaries}
	procedures := map[string]*tac.Procedure{}
	for i := range program.Procedures {
		p := &program.Procedures[i]
		if _, exists := procedures[p.Label]; exists {
			return values, fmt.Errorf("duplicate procedure")
		}
		procedures[p.Label] = p
	}
	for steps := 0; ; steps++ {
		if steps >= 10000 {
			return values, fmt.Errorf("instruction budget exceeded")
		}
		if f.pc >= len(f.tape) {
			if f.procedure != nil {
				return values, fmt.Errorf("procedure fell off tape")
			}
			for name, value := range globals {
				values[name] = value.value
			}
			return values, nil
		}
		i := f.tape[f.pc]
		f.pc++
		switch i.Op {
		case tac.Label:
			continue
		case tac.Jump:
			f.pc = f.labels[i.Label]
			continue
		case tac.BranchTrue:
			value, e := f.read(i.Left)
			if e != nil {
				return values, e
			}
			if value.kind != model.TypeBoolean {
				return values, fmt.Errorf("nonboolean branch")
			}
			if value.value.(bool) {
				f.pc = f.labels[i.Label]
			}
			continue
		case tac.Call:
			p, exists := procedures[i.Callee]
			if !exists {
				return values, fmt.Errorf("undefined procedure")
			}
			if len(i.Arguments) != len(p.Parameters) {
				return values, fmt.Errorf("argument count mismatch")
			}
			// All arguments read from the caller BEFORE constructing the callee.
			args := make([]frameValue, len(i.Arguments))
			for j, arg := range i.Arguments {
				args[j], err = f.read(arg)
				if err != nil {
					return values, err
				}
			}
			callee, e := newTestFrame(p, globals, frameContinuation{f, f.pc, i.Target})
			if e != nil {
				return values, e
			}
			for j, formal := range p.Parameters {
				if e := callee.write(formal, args[j]); e != nil {
					return values, e
				}
			}
			f = callee
			continue
		case tac.Return:
			if f.procedure == nil {
				return values, fmt.Errorf("return outside frame")
			}
			continuation, ok := f.cells[0].(frameContinuation)
			if !ok || f.cells[1] != continuation.caller {
				return values, fmt.Errorf("broken control/continuation")
			}
			if f.procedure.ResultKind == model.TypeNull {
				if i.ReturnValue != nil || continuation.target.Kind != "" {
					return values, fmt.Errorf("invalid null return")
				}
			} else {
				if i.ReturnValue == nil {
					return values, fmt.Errorf("missing scalar return")
				}
				value, e := f.read(*i.ReturnValue)
				if e != nil {
					return values, e
				}
				if value.kind != f.procedure.ResultKind || f.procedure.Frame.Slots[3].Role != "result" {
					return values, fmt.Errorf("invalid result cell")
				}
				f.cells[3] = value
				if e := continuation.caller.write(continuation.target, f.cells[3].(frameValue)); e != nil {
					return values, e
				}
			}
			f = continuation.caller
			f.pc = continuation.pc
			continue
		}
		left, e := f.read(i.Left)
		if e != nil {
			return values, e
		}
		var right frameValue
		if i.Right != nil {
			right, e = f.read(*i.Right)
			if e != nil {
				return values, e
			}
		}
		// Both operands are now values; a destination can safely reuse either cell.
		result := frameValue{kind: i.Target.Type}
		switch i.Op {
		case tac.Copy, tac.Negate:
			if left.kind != result.kind {
				return values, fmt.Errorf("missing explicit conversion")
			}
		case tac.IntegerToFloat:
			if left.kind != model.TypeInteger || result.kind != model.TypeFloat {
				return values, fmt.Errorf("invalid widening")
			}
		case tac.LogicalNot:
			if left.kind != model.TypeBoolean || result.kind != model.TypeBoolean {
				return values, fmt.Errorf("invalid boolean negation")
			}
		default:
			if i.Right == nil || left.kind != right.kind {
				return values, fmt.Errorf("invalid binary operand types")
			}
		}
		switch i.Op {
		case tac.Copy, tac.IntegerToFloat:
			result.value = left.value
		case tac.LogicalNot:
			result.value = !left.value.(bool)
		case tac.Negate:
			result.value = -left.value.(float64)
		case tac.Equal:
			result.value = left.value == right.value
		case tac.NotEqual:
			result.value = left.value != right.value
		case tac.Less:
			result.value = left.value.(float64) < right.value.(float64)
		case tac.LessEqual:
			result.value = left.value.(float64) <= right.value.(float64)
		case tac.Greater:
			result.value = left.value.(float64) > right.value.(float64)
		case tac.GreaterEqual:
			result.value = left.value.(float64) >= right.value.(float64)
		case tac.Add:
			result.value = left.value.(float64) + right.value.(float64)
		case tac.Subtract:
			result.value = left.value.(float64) - right.value.(float64)
		case tac.Multiply:
			result.value = left.value.(float64) * right.value.(float64)
		case tac.Divide:
			if right.value.(float64) == 0 {
				return values, fmt.Errorf("division by zero")
			}
			result.value = left.value.(float64) / right.value.(float64)
		default:
			return values, fmt.Errorf("unknown opcode %s", i.Op)
		}
		if e := f.write(i.Target, result); e != nil {
			return values, e
		}
	}
}

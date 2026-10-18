package compiscript

import (
	"fmt"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/tac"
)

// This marker is oracle state, not a language value or fabricated default.
// Its typed cell proves activation to existing dependency guards; read faults.
type testUndefined struct{}

func undefinedFrameValue(v frameValue) bool {
	_, undefined := v.value.(testUndefined)
	return undefined
}

// Reference formatting exposes stable declaration IDs, never physical addresses.
// Lists are bounded and cycle guarded; objects/functions are opaque references.
func renderFrameValue(value frameValue) string {
	budget := 64
	seen := map[*testList]bool{}
	var render func(frameValue, int) string
	render = func(v frameValue, depth int) string {
		if budget <= 0 || depth >= 8 {
			return "..."
		}
		budget--
		if v.value == nil {
			return "null"
		}
		switch x := v.value.(type) {
		case string:
			return x
		case bool:
			return strconv.FormatBool(x)
		case float64:
			return strconv.FormatFloat(x, 'g', -1, 64)
		case *testObject:
			return "<object " + x.layout.ID + ">"
		case testClosure:
			return "<function " + x.procedure.Label + ">"
		case *testList:
			if seen[x] {
				return "<cycle>"
			}
			seen[x] = true
			defer delete(seen, x)
			parts := make([]string, 0, len(x.values))
			for _, element := range x.values {
				parts = append(parts, render(element, depth+1))
				if budget <= 0 {
					break
				}
			}
			return "[" + strings.Join(parts, ", ") + "]"
		default:
			return "<invalid>"
		}
	}
	return render(value, 0)
}

func (f *testFrame) declareUndefined(o tac.Operand) error {
	if o.Kind != tac.Symbol {
		return fmt.Errorf("declaration requires a binding")
	}
	if len(f.layouts) != 0 {
		env := f.findEnvironment(testOperandScope(o))
		if env == nil || !environmentBinding(env, o) {
			return fmt.Errorf("undefined declaration environment")
		}
		env.cells[o.Value] = frameValue{kind: o.Type, value: testUndefined{}}
		return nil
	}
	if f.globalOperand(o) {
		store, err := f.globalStore()
		if err != nil {
			return err
		}
		store[o.Value] = frameValue{kind: o.Type, value: testUndefined{}}
		return nil
	}
	offset, err := f.ownedOffset(o)
	if err != nil {
		return err
	}
	f.cells[offset] = nil
	return nil
}

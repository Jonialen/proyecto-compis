package compiscript

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// runConditions is a bounded test oracle, not a source-language runtime. Operand
// reads precede writes so it can detect incorrect reuse at recursive joins.
func runConditions(program *tac.Program) (map[string]any, int, error) {
	values := map[string]any{}
	labels := map[string]int{}
	for pc, instruction := range program.Instructions {
		if instruction.Op == "label" {
			label := instruction.Label
			if _, exists := labels[label]; exists || label == "" {
				return values, 0, fmt.Errorf("duplicate or empty label %q", label)
			}
			labels[label] = pc
		}
	}
	for _, instruction := range program.Instructions {
		if instruction.Op == "jump" || instruction.Op == "branchTrue" {
			if _, exists := labels[instruction.Label]; !exists {
				return values, 0, fmt.Errorf("undefined label")
			}
		}
	}
	read := func(operand tac.Operand) (any, error) {
		if operand.Kind != tac.Literal {
			value, exists := values[operand.Value]
			if !exists {
				return nil, fmt.Errorf("undefined operand %s", operand)
			}
			_, boolean := value.(bool)
			if boolean != (operand.Type == model.TypeBoolean) {
				return nil, fmt.Errorf("operand type mismatch %s", operand)
			}
			return value, nil
		}
		if operand.Type == model.TypeBoolean {
			return strconv.ParseBool(operand.Value)
		}
		return strconv.ParseFloat(operand.Value, 64)
	}
	divisions := 0
	for pc, steps := 0, 0; pc < len(program.Instructions); pc, steps = pc+1, steps+1 {
		if steps >= 10000 {
			return values, divisions, fmt.Errorf("instruction budget exceeded")
		}
		i := program.Instructions[pc]
		if i.Op == "label" {
			continue
		}
		if i.Op == "jump" {
			pc = labels[i.Label]
			continue
		}
		left, err := read(i.Left)
		if err != nil {
			return values, divisions, err
		}
		if i.Op == "branchTrue" {
			condition, ok := left.(bool)
			if !ok {
				return values, divisions, fmt.Errorf("nonboolean branch")
			}
			if condition {
				pc = labels[i.Label]
			}
			continue
		}
		var right any
		if i.Right != nil {
			right, err = read(*i.Right)
			if err != nil {
				return values, divisions, err
			}
		}
		var result any
		switch i.Op {
		case "copy", "i2f":
			result = left
		case "!":
			result = !left.(bool)
		case "neg":
			result = -left.(float64)
		case "==":
			result = left == right
		case "!=":
			result = left != right
		case "<":
			result = left.(float64) < right.(float64)
		case "<=":
			result = left.(float64) <= right.(float64)
		case ">":
			result = left.(float64) > right.(float64)
		case ">=":
			result = left.(float64) >= right.(float64)
		case "+":
			result = left.(float64) + right.(float64)
		case "-":
			result = left.(float64) - right.(float64)
		case "*":
			result = left.(float64) * right.(float64)
		case "/":
			divisions++
			if right.(float64) == 0 {
				return values, divisions, fmt.Errorf("division by zero")
			}
			result = left.(float64) / right.(float64)
		default:
			return values, divisions, fmt.Errorf("unknown opcode %s", i.Op)
		}
		values[i.Target.Value] = result
	}
	return values, divisions, nil
}

func TestConditionOracleRejectsMalformedInstructions(t *testing.T) {
	for _, tt := range []struct {
		name         string
		instructions []tac.Instruction
		want         string
	}{
		{"duplicate label", []tac.Instruction{{Op: tac.Label, Label: "L0"}, {Op: tac.Label, Label: "L0"}}, "duplicate or empty label"},
		{"undefined target", []tac.Instruction{{Op: tac.Jump, Label: "missing"}}, "undefined label"},
		{"undefined temporary", []tac.Instruction{{Op: tac.Copy, Left: tac.Operand{Kind: tac.Temporary, Value: "t0", Type: model.TypeBoolean}}}, "undefined operand"},
		{"nonboolean branch", []tac.Instruction{{Op: tac.Label, Label: "L0"}, {Op: tac.BranchTrue, Label: "L0", Left: tac.Operand{Kind: tac.Literal, Value: "1", Type: model.TypeInteger}}}, "nonboolean branch"},
		{"bounded cycle", []tac.Instruction{{Op: tac.Label, Label: "L0"}, {Op: tac.Jump, Label: "L0"}}, "instruction budget exceeded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runConditions(&tac.Program{Instructions: tt.instructions})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want %q", err, tt.want)
			}
		})
	}
}

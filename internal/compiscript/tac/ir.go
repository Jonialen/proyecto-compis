// Package tac represents typed, source-located three-address code for Compiscript.
package tac

import (
	"fmt"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type OperandKind string

const (
	Literal   OperandKind = "literal"
	Symbol    OperandKind = "symbol"
	Temporary OperandKind = "temporary"
)

// Operand identifies a value, not a target-specific address or machine register.
type Operand struct {
	Kind  OperandKind    `json:"kind"`
	Value string         `json:"value"`
	Type  model.TypeKind `json:"type"`
}

func (o Operand) String() string { return o.Value + ":" + string(o.Type) }

type Opcode string

const (
	Copy           Opcode = "copy"
	Add            Opcode = "+"
	Subtract       Opcode = "-"
	Multiply       Opcode = "*"
	Divide         Opcode = "/"
	Negate         Opcode = "neg"
	IntegerToFloat Opcode = "i2f"
	Equal          Opcode = "=="
	NotEqual       Opcode = "!="
	Less           Opcode = "<"
	LessEqual      Opcode = "<="
	Greater        Opcode = ">"
	GreaterEqual   Opcode = ">="
	LogicalNot     Opcode = "!"
	Label          Opcode = "label"
	Jump           Opcode = "jump"
	BranchTrue     Opcode = "branchTrue"
)

type Instruction struct {
	Op     Opcode   `json:"op"`
	Target Operand  `json:"target"`
	Left   Operand  `json:"left"`
	Right  *Operand `json:"right,omitempty"`
	Span   ast.Span `json:"span"`
	Label  string   `json:"label,omitempty"`
}

func (i Instruction) String() string {
	switch i.Op {
	case Copy:
		return fmt.Sprintf("%s = %s", i.Target, i.Left)
	case Negate, IntegerToFloat, LogicalNot:
		return fmt.Sprintf("%s = %s %s", i.Target, i.Op, i.Left)
	case Label:
		return i.Label + ":"
	case Jump:
		return "goto " + i.Label
	case BranchTrue:
		return fmt.Sprintf("if %s goto %s", i.Left, i.Label)
	default:
		return fmt.Sprintf("%s = %s %s %s", i.Target, i.Left, i.Op, i.Right)
	}
}

type Program struct {
	Instructions    []Instruction `json:"instructions"`
	PeakTemporaries int           `json:"peakTemporaries"`
}

// String renders instructions in execution order with one trailing newline each.
func (p Program) String() string {
	var out strings.Builder
	for _, instruction := range p.Instructions {
		fmt.Fprintln(&out, instruction.String())
	}
	return out.String()
}

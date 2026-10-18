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
	Class string         `json:"class,omitempty"`
	Shape string         `json:"shape,omitempty"`
}

func (o Operand) String() string { return o.Value + ":" + string(o.Type) }

type Opcode string

const (
	Copy             Opcode = "copy"
	Add              Opcode = "+"
	Subtract         Opcode = "-"
	Multiply         Opcode = "*"
	Divide           Opcode = "/"
	Negate           Opcode = "neg"
	IntegerToFloat   Opcode = "i2f"
	Equal            Opcode = "=="
	NotEqual         Opcode = "!="
	Less             Opcode = "<"
	LessEqual        Opcode = "<="
	Greater          Opcode = ">"
	GreaterEqual     Opcode = ">="
	LogicalNot       Opcode = "!"
	Label            Opcode = "label"
	Jump             Opcode = "jump"
	BranchTrue       Opcode = "branchTrue"
	Call             Opcode = "call"
	Return           Opcode = "return"
	EnterEnvironment Opcode = "enterEnvironment"
	MakeClosure      Opcode = "makeClosure"
	CallIndirect     Opcode = "callIndirect"
	NewObject        Opcode = "newObject"
	LoadField        Opcode = "loadField"
	StoreField       Opcode = "storeField"
	BindMethod       Opcode = "bindMethod"
	NewList          Opcode = "newList"
	ListLength       Opcode = "listLength"
	LoadIndex        Opcode = "loadIndex"
	StoreIndex       Opcode = "storeIndex"
)

type Instruction struct {
	Op          Opcode      `json:"op"`
	Target      Operand     `json:"target"`
	Left        Operand     `json:"left"`
	Right       *Operand    `json:"right,omitempty"`
	Span        ast.Span    `json:"span"`
	Label       string      `json:"label,omitempty"`
	Callee      string      `json:"callee,omitempty"`
	Arguments   []Operand   `json:"arguments,omitempty"`
	ReturnValue *Operand    `json:"returnValue,omitempty"`
	Environment int         `json:"environment,omitempty"`
	Signature   *model.Type `json:"signature,omitempty"`
	Class       string      `json:"class,omitempty"`
	Member      string      `json:"member,omitempty"`
	Offset      *int        `json:"offset,omitempty"`
	Receiver    *Operand    `json:"receiver,omitempty"`
	TypeClasses []string    `json:"typeClasses,omitempty"`
	Index       *Operand    `json:"index,omitempty"`
}

func (i Instruction) String() string {
	switch i.Op {
	case NewList:
		values := make([]string, len(i.Arguments))
		for n, value := range i.Arguments {
			values[n] = value.String()
		}
		return fmt.Sprintf("%s = newList [%s]", i.Target, strings.Join(values, ", "))
	case ListLength:
		return fmt.Sprintf("%s = listLength %s", i.Target, i.Left)
	case LoadIndex:
		return fmt.Sprintf("%s = loadIndex %s[%s]", i.Target, i.Left, i.Index)
	case StoreIndex:
		return fmt.Sprintf("storeIndex %s[%s] = %s", i.Left, i.Index, i.Right)
	case NewObject:
		return fmt.Sprintf("%s = newObject %s", i.Target, i.Class)
	case LoadField:
		return fmt.Sprintf("%s = loadField %s %s[%d]", i.Target, i.Left, i.Class, *i.Offset)
	case StoreField:
		return fmt.Sprintf("storeField %s %s[%d] = %s", i.Left, i.Class, *i.Offset, i.Right)
	case BindMethod:
		return fmt.Sprintf("%s = bindMethod %s %s.%s", i.Target, i.Left, i.Class, i.Member)
	case EnterEnvironment:
		return fmt.Sprintf("enter environment s%d", i.Environment)
	case MakeClosure:
		return fmt.Sprintf("%s = closure %s environment s%d", i.Target, i.Callee, i.Environment)
	case Call, CallIndirect:
		args := make([]string, len(i.Arguments))
		for n, arg := range i.Arguments {
			args[n] = arg.String()
		}
		call := "call " + i.Callee + "(" + strings.Join(args, ", ") + ")"
		if i.Op == CallIndirect {
			call = "callIndirect " + i.Left.String() + "(" + strings.Join(args, ", ") + ")"
		}
		if i.Target.Kind != "" {
			return i.Target.String() + " = " + call
		}
		return call
	case Return:
		if i.ReturnValue != nil {
			return "return " + i.ReturnValue.String()
		}
		return "return"
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
	Instructions    []Instruction       `json:"instructions"`
	PeakTemporaries int                 `json:"peakTemporaries"`
	Procedures      []Procedure         `json:"procedures,omitempty"`
	Environments    []EnvironmentLayout `json:"environments,omitempty"`
	Classes         []ClassLayout       `json:"classes,omitempty"`
}

// String renders instructions in execution order with one trailing newline each.
func (p Program) String() string {
	var out strings.Builder
	for _, class := range p.Classes {
		fmt.Fprintf(&out, "class %s parent %s [fields=%d initializer=%s]:\n", class.ID, class.Parent, len(class.Fields), class.Initializer)
		for _, field := range class.Fields {
			fmt.Fprintf(&out, "  field %d %s:%s mutable=%t\n", field.Offset, field.ID, field.Type.Kind, field.Mutable)
		}
		for _, method := range class.Methods {
			fmt.Fprintf(&out, "  method %s = %s\n", method.Name, method.Callee)
		}
	}
	for _, env := range p.Environments {
		bindings := make([]string, len(env.Bindings))
		for i, o := range env.Bindings {
			bindings[i] = o.String()
		}
		fmt.Fprintf(&out, "environment s%d parent s%d (%s)\n", env.ScopeID, env.ParentID, strings.Join(bindings, ", "))
	}
	if len(p.Procedures) != 0 {
		fmt.Fprintln(&out, "main:")
	}
	for _, instruction := range p.Instructions {
		fmt.Fprintln(&out, instruction.String())
	}
	for _, procedure := range p.Procedures {
		formals := make([]string, len(procedure.Parameters))
		for i, parameter := range procedure.Parameters {
			formals[i] = parameter.String()
		}
		fmt.Fprintf(&out, "procedure %s(%s) -> %s [scope=%d frame=%d cells]:\n", procedure.Label, strings.Join(formals, ", "), procedure.ResultKind, procedure.ScopeID, procedure.Frame.Cells)
		if procedure.LexicalScope != 0 {
			fmt.Fprintf(&out, "  lexical environment s%d\n", procedure.LexicalScope)
		}
		for _, slot := range procedure.Frame.Slots {
			fmt.Fprintf(&out, "  cell %d %s", slot.Offset, slot.Role)
			if slot.Operand != nil && slot.Dynamic {
				fmt.Fprintf(&out, " %s", slot.Operand.Value)
			} else if slot.Operand != nil {
				fmt.Fprintf(&out, " %s", slot.Operand)
			}
			if slot.Dynamic {
				fmt.Fprint(&out, " dynamic")
			}
			fmt.Fprintln(&out)
		}
		for _, instruction := range procedure.Instructions {
			fmt.Fprintln(&out, instruction.String())
		}
		fmt.Fprintln(&out, "end "+procedure.Label)
	}
	return out.String()
}

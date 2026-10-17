package tac

import (
	"testing"

	"genanalex/internal/compiscript/model"
)

func TestInstructionText(t *testing.T) {
	x := Operand{Kind: Symbol, Value: "s1.x", Type: model.TypeInteger}
	one := Operand{Kind: Literal, Value: "1", Type: model.TypeInteger}
	temp := Operand{Kind: Temporary, Value: "t0", Type: model.TypeInteger}
	for _, tt := range []struct {
		op   Opcode
		want string
	}{
		{Copy, "s1.x:integer = 1:integer"},
		{Add, "s1.x:integer = 1:integer + t0:integer"},
		{Subtract, "s1.x:integer = 1:integer - t0:integer"},
		{Multiply, "s1.x:integer = 1:integer * t0:integer"},
		{Divide, "s1.x:float = 1:float / t0:float"},
		{Negate, "s1.x:integer = neg 1:integer"},
		{IntegerToFloat, "s1.x:float = i2f 1:integer"},
	} {
		t.Run(string(tt.op), func(t *testing.T) {
			instruction := Instruction{Op: tt.op, Target: x, Left: one, Right: &temp}
			if tt.op == Divide || tt.op == IntegerToFloat {
				instruction.Target.Type = model.TypeFloat
			}
			if tt.op == Divide {
				instruction.Left.Type = model.TypeFloat
				right := temp
				right.Type = model.TypeFloat
				instruction.Right = &right
			}
			if got := instruction.String(); got != tt.want {
				t.Fatalf("instruction = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProgramText(t *testing.T) {
	if got := (Program{}).String(); got != "" {
		t.Fatalf("empty program = %q", got)
	}
	x := Operand{Kind: Symbol, Value: "s1.x", Type: model.TypeInteger}
	one := Operand{Kind: Literal, Value: "1", Type: model.TypeInteger}
	program := Program{Instructions: []Instruction{{Op: Copy, Target: x, Left: one}}}
	want := "s1.x:integer = 1:integer\n"
	if first, second := program.String(), program.String(); first != want || second != first {
		t.Fatalf("program text = %q / %q, want %q", first, second, want)
	}
}

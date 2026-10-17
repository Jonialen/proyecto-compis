package tac

import (
	"encoding/json"
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

func TestConditionalInstructionText(t *testing.T) {
	condition := Operand{Kind: Temporary, Value: "t0", Type: model.TypeBoolean}
	left := Operand{Kind: Literal, Value: "1", Type: model.TypeInteger}
	right := Operand{Kind: Literal, Value: "2", Type: model.TypeInteger}
	for _, op := range []Opcode{Equal, NotEqual, Less, LessEqual, Greater, GreaterEqual} {
		instruction := Instruction{Op: op, Target: condition, Left: left, Right: &right}
		want := "t0:boolean = 1:integer " + string(op) + " 2:integer"
		if instruction.String() != want {
			t.Fatalf("%s: %q", op, instruction.String())
		}
	}
	for _, tt := range []struct {
		instruction Instruction
		want        string
	}{
		{Instruction{Op: Label, Label: "L0"}, "L0:"},
		{Instruction{Op: Jump, Label: "L0"}, "goto L0"},
		{Instruction{Op: BranchTrue, Label: "L0", Left: condition}, "if t0:boolean goto L0"},
		{Instruction{Op: LogicalNot, Target: condition, Left: condition}, "t0:boolean = ! t0:boolean"},
	} {
		if tt.instruction.String() != tt.want {
			t.Fatalf("got %q want %q", tt.instruction.String(), tt.want)
		}
	}
}

func TestNumericInstructionJSONCompatibility(t *testing.T) {
	// Pin field order and omission independently of the current report encoder.
	one := Operand{Kind: Literal, Value: "1", Type: model.TypeInteger}
	data, err := json.Marshal(Instruction{Op: Copy, Target: one, Left: one})
	want := `{"op":"copy","target":{"kind":"literal","value":"1","type":"integer"},"left":{"kind":"literal","value":"1","type":"integer"},"span":{"start":{"offset":0,"line":0,"column":0},"end":{"offset":0,"line":0,"column":0}}}`
	if err != nil || string(data) != want {
		t.Fatalf("JSON=%s err=%v", data, err)
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

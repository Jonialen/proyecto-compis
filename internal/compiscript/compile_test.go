package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

func TestCompileNumeric(t *testing.T) {
	tests := []struct {
		name, source, want string
		peak               int
	}{
		{"precedence", "let x=1+2*3;", "t0:integer = 2:integer * 3:integer\nt0:integer = 1:integer + t0:integer\ns1.x:integer = t0:integer\n", 1},
		{"left associative", "let x=8-3-1;", "t0:integer = 8:integer - 3:integer\nt0:integer = t0:integer - 1:integer\ns1.x:integer = t0:integer\n", 1},
		{"division", "let x=1/2;", "t0:float = i2f 1:integer\nt1:float = i2f 2:integer\nt0:float = t0:float / t1:float\ns1.x:float = t0:float\n", 2},
		{"mixed", "let x=1+2.5;", "t0:float = i2f 1:integer\nt0:float = t0:float + 2.5:float\ns1.x:float = t0:float\n", 1},
		{"widen stores", "let x:float=1; x=2;", "t0:float = i2f 1:integer\ns1.x:float = t0:float\nt0:float = i2f 2:integer\ns1.x:float = t0:float\n", 1},
		{"constant and grouping", "const x=-(1+2); let y=-x;", "t0:integer = 1:integer + 2:integer\nt0:integer = neg t0:integer\ns1.x:integer = t0:integer\nt0:integer = neg s1.x:integer\ns1.y:integer = t0:integer\n", 1},
		{"shadow initializer", "let x=1; { let x=x+2; x=x*3; } x=x+4;", "s1.x:integer = 1:integer\nt0:integer = s1.x:integer + 2:integer\ns2.x:integer = t0:integer\nt0:integer = s2.x:integer * 3:integer\ns2.x:integer = t0:integer\nt0:integer = s1.x:integer + 4:integer\ns1.x:integer = t0:integer\n", 1},
		{"sibling blocks", "{let x=1;} {let x=2;}", "s2.x:integer = 1:integer\ns3.x:integer = 2:integer\n", 0},
		{"live left operand", "let x=(1+2)*((3+4)*(5+6));", "t0:integer = 1:integer + 2:integer\nt1:integer = 3:integer + 4:integer\nt2:integer = 5:integer + 6:integer\nt1:integer = t1:integer * t2:integer\nt0:integer = t0:integer * t1:integer\ns1.x:integer = t0:integer\n", 3},
		{"large integer", "let x=999999999999999999999999;", "s1.x:integer = 999999999999999999999999:integer\n", 0},
		{"float unary", "let x=-2.5;", "t0:float = neg 2.5:float\ns1.x:float = t0:float\n", 1},
		{"drop expression results", "1+2; 3+4;", "t0:integer = 1:integer + 2:integer\nt0:integer = 3:integer + 4:integer\n", 1},
		{"empty", "", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			before := append([]byte{}, source...)
			report := Compile(source)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compilation failed: %+v", report)
			}
			if got := report.TAC.String(); got != tt.want {
				t.Fatalf("TAC:\n%s\nwant:\n%s", got, tt.want)
			}
			if report.TAC.PeakTemporaries != tt.peak {
				t.Fatalf("peak = %d, want %d", report.TAC.PeakTemporaries, tt.peak)
			}
			for _, instruction := range report.TAC.Instructions {
				if instruction.Span.Start.Line < 1 || instruction.Span.End.Offset <= instruction.Span.Start.Offset {
					t.Fatalf("missing span: %+v", instruction)
				}
			}
			if !bytes.Equal(source, before) {
				t.Fatal("source mutated")
			}
			analysis := Analyze(source)
			if !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatal("semantic snapshots changed")
			}
			first, _ := json.Marshal(report)
			second, _ := json.Marshal(Compile(source))
			if !bytes.Equal(first, second) {
				t.Fatalf("nondeterministic JSON: %s != %s", first, second)
			}
			if tt.source == "" && !bytes.Contains(first, []byte(`"instructions":[]`)) {
				t.Fatalf("empty instructions: %s", first)
			}
		})
	}
}

func TestCompileRepeatedExpressions(t *testing.T) {
	source := "let x=0;" + strings.Repeat("x=(1+2)*(3+4);", 100)
	report := Compile([]byte(source))
	want := "s1.x:integer = 0:integer\n" + strings.Repeat("t0:integer = 1:integer + 2:integer\nt1:integer = 3:integer + 4:integer\nt0:integer = t0:integer * t1:integer\ns1.x:integer = t0:integer\n", 100)
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		t.Fatalf("compilation failed: %+v", report.Diagnostics)
	}
	if report.TAC.String() != want || report.TAC.PeakTemporaries != 2 {
		t.Fatalf("unexpected repeated TAC: %+v", report.TAC)
	}
}

func TestCompileRejectsInvalidConstructionsWithoutPartialTAC(t *testing.T) {
	for _, tt := range []struct{ source, rejected, message string }{
		{`try{missing;}catch(e){}`, `missing`, "unresolved name missing"},
		{`function f(a) {missing;return;}`, `missing`, "unresolved name missing"},
		{`try{let y=[1,missing];}catch(e){}`, `missing`, "unresolved name missing"},
		{`try{let y=["a",missing];}catch(e){}`, `missing`, "unresolved name missing"},
		{`let y=missing;`, `missing`, "unresolved name missing"},
		{`function y(a){missing;}`, `missing`, "unresolved name missing"},
		{`let y=true?1:missing;`, `missing`, "unresolved name missing"},
		{`let y=0; try{y=missing;}catch(e){}`, `missing`, "unresolved name missing"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			input := []byte("let x=1; " + tt.source)
			analysis := Analyze(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) != 1 {
				t.Fatalf("expected rejection without TAC: %+v", report)
			}
			start := strings.Index(string(input), tt.rejected)
			want := model.Diagnostics{{Code: "SEM_UNRESOLVED", Phase: model.Phase("semantic"), Message: tt.message, Span: compileSpan(start, start+len(tt.rejected))}}
			if report.Codegen != nil || !reflect.DeepEqual(report.Diagnostics, want) || !reflect.DeepEqual(analysis.Diagnostics, want) {
				t.Fatalf("diagnostics = %+v, want %+v", report.Diagnostics, want)
			}
			if !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatal("snapshots changed on rejection")
			}
		})
	}
}

func compileSpan(start, end int) ast.Span {
	return ast.Span{Start: ast.Position{Offset: start, Line: 1, Column: start + 1}, End: ast.Position{Offset: end, Line: 1, Column: end + 1}}
}

func TestCompileInstructionMetadata(t *testing.T) {
	integer := func(value string) tac.Operand {
		return tac.Operand{Kind: tac.Literal, Value: value, Type: model.TypeInteger}
	}
	temp := func(value string) tac.Operand {
		return tac.Operand{Kind: tac.Temporary, Value: value, Type: model.TypeFloat}
	}
	right := temp("t1")
	want := &tac.Program{PeakTemporaries: 2, Instructions: []tac.Instruction{
		{Op: tac.IntegerToFloat, Target: temp("t0"), Left: integer("1"), Span: compileSpan(6, 7)},
		{Op: tac.IntegerToFloat, Target: temp("t1"), Left: integer("2"), Span: compileSpan(8, 9)},
		{Op: tac.Divide, Target: temp("t0"), Left: temp("t0"), Right: &right, Span: compileSpan(6, 9)},
		{Op: tac.Copy, Target: tac.Operand{Kind: tac.Symbol, Value: "s1.x", Type: model.TypeFloat}, Left: temp("t0"), Span: compileSpan(0, 10)},
	}}
	report := Compile([]byte("let x=1/2;"))
	if !reflect.DeepEqual(report.TAC, want) {
		t.Fatalf("typed located instructions = %+v, want %+v", report.TAC, want)
	}
	data, err := json.Marshal(report)
	if err != nil || !bytes.Contains(data, []byte(`"diagnostics":[]`)) || !bytes.Contains(data, []byte(`"tac":{`)) || !bytes.Contains(data, []byte(`"scopes":[`)) {
		t.Fatalf("report JSON: %s (%v)", data, err)
	}
}

func TestCompilePreservesAnalysisErrors(t *testing.T) {
	for _, source := range []string{`let x=;`, `let x=missing;`, `const x=1; x=2;`, `let x=true+1;`, `let x=1; @`} {
		t.Run(source, func(t *testing.T) {
			analysis := Analyze([]byte(source))
			report := Compile([]byte(source))
			if len(analysis.Diagnostics) == 0 {
				t.Fatal("fixture must fail analysis")
			}
			if report.TAC != nil || !reflect.DeepEqual(report.Diagnostics, analysis.Diagnostics) || !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatalf("analysis contract changed: %+v vs %+v", report, analysis)
			}
		})
	}
}

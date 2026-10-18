package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

func TestCompileRecursiveInvocations(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"factorial", `function fact(n:integer):integer { if(n<=1) return 1; else return n*fact(n-1); } let x=fact(5); let y=fact(3);`, map[string]any{"s1.x": float64(120), "s1.y": float64(6)}},
		{"two recursive results", `function fib(n:integer):integer { if(n<=1) return n; else return fib(n-1)+fib(n-2); } let x=fib(8);`, map[string]any{"s1.x": float64(21)}},
		{"mutual boolean", `function even(n:integer):boolean { if(n==0) return true; else return odd(n-1); } function odd(n:integer):boolean { if(n==0) return false; else return even(n-1); } let x=even(8); let y=odd(7); let z=odd(0);`, map[string]any{"s1.x": true, "s1.y": true, "s1.z": false}},
		{"null procedure", `let g=0; function count(n:integer) { if(n==0) return null; g=g+1; count(n-1); return; } count(5);`, map[string]any{"s1.g": float64(5)}},
		{"caller local isolation", `function sum(n:integer):integer { let saved=n; if(n==0) return 0; else { n=n-1; let child=sum(n); return saved+child+n; } } let x=sum(4);`, map[string]any{"s1.x": float64(16)}},
		{"recursive float widening", `function sum(n:integer):float { if(n==0) return 0; else return n+sum(n-1); } let x=sum(4);`, map[string]any{"s1.x": float64(10)}},
		{"unused self cycle", `function forever() { forever(); }`, map[string]any{}},
		{"unused mutual cycle", `function a() { b(); } function b() { a(); }`, map[string]any{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			report := Compile(input)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil || !reflect.DeepEqual(values, tt.want) {
				t.Fatalf("values=%v want=%v err=%v", values, tt.want, err)
			}
			first, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			second, err := json.Marshal(Compile(input))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(input, before) || !bytes.Equal(first, second) || !reflect.DeepEqual(report.Scopes, Analyze(input).Scopes) {
				t.Fatal("input/scopes changed or recursive generation is nondeterministic")
			}
		})
	}
}

func TestRecursiveGlobalFixedPoint(t *testing.T) {
	// The cycle must inherit an outgoing dependency, including an unused write.
	functions := `function a(n:integer):integer { if(n==0) return 0; else return b(n-1); }
function b(n:integer):integer { if(n==0) return read(); else return c(n-1); }
function c(n:integer):integer { if(false) { z=9; } if(n==0) return 0; else return a(n-1); }
function read():integer { return g; }`
	for _, early := range []bool{false, true} {
		t.Run(map[bool]string{false: "initialized", true: "early call"}[early], func(t *testing.T) {
			source := `let g=7; let z=0; ` + functions + ` let x=c(2);`
			if early {
				source = `let x=c(2); let g=7; let z=0; ` + functions
			}
			report := Compile([]byte(source))
			if early {
				if report.TAC != nil || len(report.Diagnostics) != 1 || report.Diagnostics[0].Message != "call requires initialized global binding s1.g" {
					t.Fatalf("non-atomic or wrong early-call rejection: %+v", report)
				}
				return
			}
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			want := []tac.Operand{{Kind: tac.Symbol, Value: "s1.g", Type: model.TypeInteger}, {Kind: tac.Symbol, Value: "s1.z", Type: model.TypeInteger}}
			for _, procedure := range report.TAC.Procedures[:3] {
				if !reflect.DeepEqual(procedure.Globals, want) {
					t.Errorf("%s globals=%v want=%v", procedure.Label, procedure.Globals, want)
				}
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil || values["s1.x"] != float64(7) || values["s1.z"] != float64(0) {
				t.Fatalf("values=%v err=%v", values, err)
			}
		})
	}
}

func TestRecursiveOracleStillBoundsDepth(t *testing.T) {
	program := compileConditions(t, `function forever() { forever(); } forever();`)
	_, err := runFunctionFrames(program)
	if err == nil || !strings.Contains(err.Error(), "call depth budget exceeded") {
		t.Fatalf("unbounded recursive oracle: %v", err)
	}
}

func TestRecursiveEarlyWriteIsAtomic(t *testing.T) {
	for _, source := range []string{
		`a(1); let z=0; function a(n:integer) { if(n>0) b(n-1); } function b(n:integer) { if(false) z=9; a(n); }`,
		`function a(n:integer) { b(n); } function b(n:integer) { a(n); try{missing;}catch(e){} }`,
	} {
		input := []byte(source)
		before := append([]byte(nil), input...)
		report := Compile(input)
		if report.TAC != nil || len(report.Diagnostics) == 0 || !bytes.Equal(input, before) {
			t.Fatalf("non-atomic recursive rejection: %+v", report.Diagnostics)
		}
		if strings.Contains(source, "let z") && report.Diagnostics[0].Message != "call requires initialized global binding s1.z" {
			t.Fatalf("lost write-only global dependency: %+v", report.Diagnostics)
		}
	}
}

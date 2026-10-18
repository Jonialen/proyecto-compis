package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

// Two connected recursive components must reach the same dependency closure
// regardless of function declaration order or outgoing call order.
func TestRecursiveDependencyOrders(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		terms := "b(n-1)+d(0)"
		if reverse {
			terms = "d(0)+b(n-1)"
		}
		functions := []string{
			"function a(n:integer):integer { if(n==0) return x; return " + terms + "; }",
			"function b(n:integer):integer { if(n==0) return y; return c(n-1); }",
			"function c(n:integer):integer { if(n==0) return z; return a(n-1); }",
			"function d(n:integer):integer { if(n==0) return w; return e(n-1); }",
			"function e(n:integer):integer { if(n==0) return w; return d(n-1)+e(0); }",
		}
		var visit func(int)
		visit = func(i int) {
			if i < len(functions) {
				for j := i; j < len(functions); j++ {
					functions[i], functions[j] = functions[j], functions[i]
					visit(i + 1)
					functions[i], functions[j] = functions[j], functions[i]
				}
				return
			}
			input := []byte("let z=3; let y=2; let x=1; let w=4; " + strings.Join(functions, " ") + " let result=a(3);")
			hash := sha256.Sum256(input)
			report := Compile(input)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			for _, procedure := range report.TAC.Procedures {
				want := []string{"s1.w"}
				switch procedure.Label {
				case "s1.a", "s1.b", "s1.c":
					want = []string{"s1.w", "s1.x", "s1.y", "s1.z"}
				}
				var got []string
				for _, global := range procedure.Globals {
					got = append(got, global.Value)
					if string(global.Kind) != "symbol" || string(global.Type) != "integer" {
						t.Fatalf("invalid dependency: %+v", global)
					}
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s globals=%v want=%v", procedure.Label, got, want)
				}
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil || values["s1.result"] != float64(5) {
				t.Fatalf("values=%v err=%v", values, err)
			}
			first, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			second, err := json.Marshal(Compile(input))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) || sha256.Sum256(input) != hash || !reflect.DeepEqual(report.Scopes, Analyze(input).Scopes) {
				t.Fatal("nondeterminism or input/scope mutation")
			}
		}
		visit(0)
	}
}

func TestRecursiveEarlyDependencyPaths(t *testing.T) {
	for i, body := range []string{
		"return g;",
		"g=9; return 1;",
		"if(false) return g; return 1;",
		"if(false) g=9; return 1;",
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			prefix := "function a(n:integer):integer { if(n==0) return b(); return a(n-1); } "
			suffix := "let g=7; function b():integer { " + body + " }"
			input := []byte(prefix + "let result=a(1); " + suffix)
			hash := sha256.Sum256(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) != 1 || report.Diagnostics[0].Message != "call requires initialized global binding s1.g" {
				t.Fatalf("non-atomic or wrong error: %+v", report.Diagnostics)
			}
			if sha256.Sum256(input) != hash || !reflect.DeepEqual(report.Scopes, Analyze(input).Scopes) {
				t.Fatal("rejection mutated input/scopes")
			}
			late := Compile([]byte(prefix + suffix + " let result=a(1);"))
			if late.TAC == nil || len(late.Diagnostics) != 0 {
				t.Fatalf("late call rejected: %+v", late.Diagnostics)
			}
		})
	}
}

func TestRecursiveGeneratePreservesAST(t *testing.T) {
	for _, early := range []bool{false, true} {
		source := "let g=1; function a(n:integer):integer { if(n==0) return g; return a(n-1); } let x=a(3);"
		if early {
			source = "let x=a(3); " + strings.TrimSuffix(source, " let x=a(3);")
		}
		input := []byte(source)
		hash := sha256.Sum256(input)
		program, frontendDiagnostics := frontend.Parse(input)
		scopes, semanticDiagnostics := semantic.Analyze(program)
		if len(frontendDiagnostics)+len(semanticDiagnostics) != 0 {
			t.Fatalf("frontend: %v %v", frontendDiagnostics, semanticDiagnostics)
		}
		before, err := json.Marshal([]any{program, scopes})
		if err != nil {
			t.Fatal(err)
		}
		result, diagnostics := tac.Generate(program, scopes)
		if early {
			if result != nil || len(diagnostics) != 1 || diagnostics[0].Message != "call requires initialized global binding s1.g" {
				t.Fatalf("early: %v", diagnostics)
			}
		} else if result == nil || len(diagnostics) != 0 {
			t.Fatalf("success: %v", diagnostics)
		}
		after, err := json.Marshal([]any{program, scopes})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) || sha256.Sum256(input) != hash {
			t.Fatal("generation mutated AST, scopes or bytes")
		}
	}
}

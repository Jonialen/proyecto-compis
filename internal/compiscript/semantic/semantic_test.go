package semantic_test

import (
	"os"
	"slices"
	"testing"

	compiscript "genanalex/internal/compiscript"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	source, err := os.ReadFile("testdata/" + name + ".cps")
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func codes(diagnostics model.Diagnostics) []string {
	result := make([]string, len(diagnostics))
	for i := range diagnostics {
		result[i] = diagnostics[i].Code
	}
	return result
}

func TestNamesScopesResolutionAndDuplicates(t *testing.T) {
	program, parseDiagnostics := frontend.Parse(fixture(t, "valid"))
	if len(parseDiagnostics) != 0 {
		t.Fatalf("parse diagnostics: %+v", parseDiagnostics)
	}
	scopes, diagnostics := semantic.Analyze(program)
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
	if len(scopes) != 3 || scopes[0].Kind != model.ScopeGlobal || scopes[1].Kind != model.ScopeFunction || scopes[2].Kind != model.ScopeBlock {
		t.Fatalf("unexpected ordered scopes: %+v", scopes)
	}
	if got := []string{scopes[0].Symbols[0].Name, scopes[0].Symbols[1].Name}; !slices.Equal(got, []string{"global", "sum"}) {
		t.Fatalf("global symbol order = %v", got)
	}
	if report := compiscript.Analyze(fixture(t, "valid")); report.AST.Kind != "program" || len(report.Diagnostics) != 0 {
		t.Fatalf("facade report: %+v", report)
	}

	program, _ = frontend.Parse(fixture(t, "invalid"))
	_, diagnostics = semantic.Analyze(program)
	want := []string{"SEM_DUPLICATE", "SEM_TYPE", "SEM_CONSTANT_ASSIGNMENT", "SEM_DUPLICATE", "SEM_UNRESOLVED", "SEM_TYPE", "SEM_DUPLICATE"}
	if got := codes(diagnostics); !slices.Equal(got, want) {
		t.Fatalf("diagnostic codes = %v, want %v", got, want)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Phase != model.PhaseSemantic || diagnostic.Span.Start.Line < 1 || diagnostic.Span.Start.Column < 1 {
			t.Fatalf("diagnostic is not source-located: %+v", diagnostic)
		}
	}
}

func TestTypesOperatorsAssignmentsAndErrorSuppression(t *testing.T) {
	tests := []struct {
		name, source string
		want         []string
	}{
		{"promotion division and null", `let n: float = 1 + 2.5; let q: float = 4 / 2; let s: string = null;`, nil},
		{"valid operators", `let s: string = "a" + "b"; let b: boolean = 1 < 2.0 && true;`, nil},
		{"invalid operators and assignments", `let i: integer = 1.5; let b: boolean = 1 && true; let s: string = "a" - "b";`, []string{"SEM_TYPE", "SEM_OPERATOR", "SEM_OPERATOR"}},
		{"function operand", `function f(): integer { return 1; } let x: integer = f + 1;`, []string{"SEM_OPERATOR"}},
		{"error suppression", `let x: integer = absent + "x";`, []string{"SEM_UNRESOLVED"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, _ := frontend.Parse([]byte(tt.source))
			_, diagnostics := semantic.Analyze(program)
			if got := codes(diagnostics); !slices.Equal(got, tt.want) {
				t.Fatalf("diagnostic codes = %v, want %v", got, tt.want)
			}
		})
	}
}

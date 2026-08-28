package semantic_test

import (
	"slices"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
)

func analyzeFixture(t *testing.T, name string) (model.ScopeSnapshots, model.Diagnostics) {
	t.Helper()
	program, diagnostics := frontend.Parse(fixture(t, name))
	if len(diagnostics) != 0 {
		t.Fatalf("parse diagnostics: %+v", diagnostics)
	}
	return semantic.Analyze(program)
}

func requireCounts(t *testing.T, diagnostics model.Diagnostics, want map[string]int) {
	t.Helper()
	got := map[string]int{}
	for _, diagnostic := range diagnostics {
		got[diagnostic.Code]++
		if diagnostic.Phase != model.PhaseSemantic || diagnostic.Span.Start.Line < 1 || diagnostic.Span.Start.Column < 1 {
			t.Fatalf("diagnostic is not located: %+v", diagnostic)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("diagnostic counts = %v, want %v", got, want)
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("diagnostic counts = %v, want %v", got, want)
		}
	}
}

func TestFunctionsRecursionForwardCallsAndCapture(t *testing.T) {
	scopes, diagnostics := analyzeFixture(t, "functions_valid")
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
	foundCapture := false
	for _, current := range scopes {
		for _, symbol := range current.Symbols {
			foundCapture = foundCapture || symbol.Name == "seed" && symbol.Captured
		}
	}
	if !foundCapture {
		t.Fatal("closure did not mark seed as captured")
	}
}

func TestFunctionsArgumentsReturnsAndExternalDeclaration(t *testing.T) {
	_, diagnostics := analyzeFixture(t, "functions_invalid")
	requireCounts(t, diagnostics, map[string]int{
		"SEM_ARGUMENT": 2, "SEM_ARITY": 1, "SEM_CONDITION": 1,
		"SEM_MISSING_RETURN": 4, "SEM_RETURN": 2, "SEM_TRANSFER": 1,
	})

	result := ast.TypeRef{Name: "integer"}
	program := ast.Program{Statements: ast.Statements{ast.FunctionDeclStmt{Name: "native", Result: &result}}}
	scopes, diagnostics := semantic.Analyze(program)
	if len(diagnostics) != 0 || len(scopes[0].Symbols) != 1 || scopes[0].Symbols[0].Type.Result.Kind != model.TypeInteger {
		t.Fatalf("external declaration: scopes=%+v diagnostics=%+v", scopes, diagnostics)
	}
}

func TestFlowValidConditionsLoopsSwitchAndTransfers(t *testing.T) {
	_, diagnostics := analyzeFixture(t, "flow_valid")
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
}

func TestFlowInvalidConditionsCasesTransfersAndUnreachable(t *testing.T) {
	_, diagnostics := analyzeFixture(t, "flow_invalid")
	requireCounts(t, diagnostics, map[string]int{
		"SEM_CASE_TYPE": 1, "SEM_CONDITION": 4, "SEM_DUPLICATE_CASE": 4,
		"SEM_TRANSFER": 4, "SEM_UNREACHABLE": 4,
	})
	wantOrder := slices.Clone(codes(diagnostics))
	_, again := analyzeFixture(t, "flow_invalid")
	if !slices.Equal(codes(again), wantOrder) {
		t.Fatalf("diagnostic order changed: %v then %v", wantOrder, codes(again))
	}
}

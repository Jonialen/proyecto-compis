package tac_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

func TestGeneratePreservesInputs(t *testing.T) {
	for _, source := range []string{"let x:float=1; {let x=x+2;} x=x/2;", "let x=1; print(x);", "let x;"} {
		t.Run(source, func(t *testing.T) {
			program, parseDiagnostics := frontend.Parse([]byte(source))
			scopes, diagnostics := semantic.Analyze(program)
			if len(parseDiagnostics)+len(diagnostics) != 0 {
				t.Fatalf("invalid fixture: %+v %+v", parseDiagnostics, diagnostics)
			}
			before, err := json.Marshal([]any{program, scopes})
			if err != nil {
				t.Fatal(err)
			}
			first, firstDiagnostics := tac.Generate(program, scopes)
			second, secondDiagnostics := tac.Generate(program, scopes)
			if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstDiagnostics, secondDiagnostics) {
				t.Fatal("generation is nondeterministic")
			}
			if source == "let x:float=1; {let x=x+2;} x=x/2;" {
				if first == nil || len(firstDiagnostics) != 0 {
					t.Fatalf("valid program rejected: %+v", firstDiagnostics)
				}
			} else if first != nil || len(firstDiagnostics) != 1 {
				t.Fatalf("unsupported program accepted: %+v %+v", first, firstDiagnostics)
			}
			if source == "let x;" {
				if first != nil || len(firstDiagnostics) != 1 || firstDiagnostics[0].Message != "numeric declarations require an initializer" {
					t.Fatalf("missing initializer accepted: %+v %+v", first, firstDiagnostics)
				}
			}
			after, err := json.Marshal([]any{program, scopes})
			if err != nil || string(before) != string(after) {
				t.Fatalf("AST or snapshots mutated: %s != %s (%v)", before, after, err)
			}
		})
	}
}

func TestGenerateRejectsInferredErrorAndEffectfulExpressions(t *testing.T) {
	// Defensive public-lowerer checks: these AST forms must not accidentally emit
	// numeric instructions, even when called outside the source compilation gate.
	span := ast.Span{Start: ast.Position{Line: 1, Column: 1}, End: ast.Position{Offset: 1, Line: 1, Column: 2}}
	for _, expression := range []ast.Expression{
		ast.AssignExpr{Span: span}, ast.PropertyAssignExpr{Span: span},
		ast.LiteralExpr{Span: span, Lexeme: "1e3"}, ast.LiteralExpr{Span: span, Lexeme: ".5"},
	} {
		program := ast.Program{Span: span, Statements: ast.Statements{ast.ExprStmt{Span: span, Expression: expression}}}
		scopes := model.ScopeSnapshots{{ID: 1, Kind: model.ScopeGlobal, Span: span}}
		result, diagnostics := tac.Generate(program, scopes)
		want := model.Diagnostics{{Code: "TAC_UNSUPPORTED", Phase: model.Phase("codegen"), Message: "unsupported expression in numeric TAC", Span: span}}
		if result != nil || !reflect.DeepEqual(diagnostics, want) {
			t.Fatalf("unsupported expression accepted: %+v %+v", result, diagnostics)
		}
	}
	program := ast.Program{Span: span, Statements: ast.Statements{ast.VarDeclStmt{Span: span, Name: "x", Initializer: ast.LiteralExpr{Span: span, Lexeme: "1"}}}}
	scopes := model.ScopeSnapshots{{ID: 1, Kind: model.ScopeGlobal, Span: span, Symbols: model.Symbols{{Name: "x", Span: span, Type: model.Type{Kind: model.TypeError}}}}}
	result, diagnostics := tac.Generate(program, scopes)
	want := model.Diagnostics{{Code: "TAC_UNSUPPORTED", Phase: model.Phase("codegen"), Message: "only integer, float and boolean declarations are supported", Span: span}}
	if result != nil || !reflect.DeepEqual(diagnostics, want) {
		t.Fatalf("error type accepted: %+v %+v", result, diagnostics)
	}
}

func TestGenerateConditionsPreservesASTAndScopes(t *testing.T) {
	source := []byte("let x=true; if(x&&!false){let x=!x;}else if(!x)x=true;")
	program, parseDiagnostics := frontend.Parse(source)
	scopes, diagnostics := semantic.Analyze(program)
	if len(parseDiagnostics)+len(diagnostics) != 0 {
		t.Fatalf("invalid fixture: %+v %+v", parseDiagnostics, diagnostics)
	}
	before, err := json.Marshal([]any{program, scopes})
	if err != nil {
		t.Fatal(err)
	}
	first, firstDiagnostics := tac.Generate(program, scopes)
	second, secondDiagnostics := tac.Generate(program, scopes)
	if first == nil || len(firstDiagnostics) != 0 || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstDiagnostics, secondDiagnostics) {
		t.Fatalf("nondeterministic or rejected generation: %+v %+v", firstDiagnostics, secondDiagnostics)
	}
	after, err := json.Marshal([]any{program, scopes})
	if err != nil || string(before) != string(after) {
		t.Fatalf("AST or snapshots mutated: %s != %s (%v)", before, after, err)
	}
}

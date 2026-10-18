package tac_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

func TestGenerateFunctionsPreservesASTAndScopes(t *testing.T) {
	for _, source := range []string{
		`let g=1; function f(a:integer):integer{{let a=a+g;} return a;} let x=f(2);`,
		`function f(){function nested(){return;} return;}`,
		`function f(){f();}`,
		`function zero():integer{return 0;} let saved=zero; function install(seed:integer){let count=seed; function next():integer{count=count+1;return count;} saved=next;} install(10); let first=saved();`,
	} {
		program, diags := frontend.Parse([]byte(source))
		scopes, semDiags := semantic.Analyze(program)
		if len(diags)+len(semDiags) != 0 {
			t.Fatalf("fixture: %+v %+v", diags, semDiags)
		}
		before, _ := json.Marshal([]any{program, scopes})
		a, da := tac.Generate(program, scopes)
		b, db := tac.Generate(program, scopes)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(da, db) {
			t.Fatal("nondeterministic generation")
		}
		after, _ := json.Marshal([]any{program, scopes})
		if string(before) != string(after) {
			t.Fatal("AST or scopes mutated")
		}
	}
}

func TestGenerateRejectsBodylessFunction(t *testing.T) {
	program, diags := frontend.Parse([]byte(`function f():integer{return 1;}`))
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	n := program.Statements[0].(ast.FunctionDeclStmt)
	n.Body = nil
	program.Statements[0] = n
	scopes, diags := semantic.Analyze(program)
	if len(diags) != 0 {
		t.Fatal(diags)
	}
	result, diags := tac.Generate(program, scopes)
	if result != nil || len(diags) != 1 || diags[0].Message != "bodyless functions are unsupported" || diags[0].Span != n.Span {
		t.Fatalf("result=%+v diagnostics=%+v", result, diags)
	}
}

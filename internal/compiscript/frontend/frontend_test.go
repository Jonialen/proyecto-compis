package frontend

import (
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestParseExposesLocatedProgramStatementsAndDiagnostics(t *testing.T) {
	source := []byte("let answer: integer = 42;")
	program, diagnostics := Parse(source)

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	if got, want := program.Span, (ast.Span{Start: ast.Position{Offset: 0, Line: 1, Column: 1}, End: ast.Position{Offset: len(source), Line: 1, Column: len(source) + 1}}); got != want {
		t.Fatalf("program span = %#v, want %#v", got, want)
	}
	if len(program.Statements) != 1 {
		t.Fatalf("statements = %#v, want one declaration", program.Statements)
	}
	if _, ok := program.Statements[0].(ast.VarDeclStmt); !ok {
		t.Fatalf("statement = %T, want ast.VarDeclStmt", program.Statements[0])
	}
}

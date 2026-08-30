package frontend

import (
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
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

func TestGeneratedVisitorDispatchBuildsProgram(t *testing.T) {
	source := []byte("let answer: integer = 42; print(answer);")
	parser := generated.NewCompiscriptParser(antlr.NewCommonTokenStream(
		generated.NewCompiscriptLexer(antlr.NewInputStream(string(source))), antlr.TokenDefaultChannel,
	))

	got, ok := parser.Program().(antlr.ParseTree).Accept(newStatementMapper(source)).(ast.Program)
	if !ok {
		t.Fatal("generated visitor did not return an AST program")
	}
	if len(got.Statements) != 2 {
		t.Fatalf("statements = %d, want 2", len(got.Statements))
	}
	if _, ok := got.Statements[0].(ast.VarDeclStmt); !ok {
		t.Fatalf("first statement = %T, want ast.VarDeclStmt", got.Statements[0])
	}
	if _, ok := got.Statements[1].(ast.PrintStmt); !ok {
		t.Fatalf("second statement = %T, want ast.PrintStmt", got.Statements[1])
	}
}

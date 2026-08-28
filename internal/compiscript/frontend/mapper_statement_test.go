package frontend

import (
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestMapStatementAlternatives(t *testing.T) {
	tests := []struct {
		name, source string
		want         ast.Statement
	}{
		{"variable declaration", "let value: integer = 1;", ast.VarDeclStmt{}},
		{"constant declaration", "const value: integer = 1;", ast.ConstDeclStmt{}},
		{"assignment", "value = 1;", ast.AssignStmt{}},
		{"function", "function f(value: integer, other) {}", ast.FunctionDeclStmt{}},
		{"class", "class Child: Parent { let value: integer; }", ast.ClassDeclStmt{}},
		{"expression", "call();", ast.ExprStmt{}},
		{"print", "print(1);", ast.PrintStmt{}},
		{"block", "{ print(1); }", ast.BlockStmt{}},
		{"if", "if (true) print(1); else print(2);", ast.IfStmt{}},
		{"while", "while (true) {}", ast.WhileStmt{}},
		{"do while", "do {} while (true);", ast.DoWhileStmt{}},
		{"for", "for (let i: integer = 0; i < 2; i = i + 1) {}", ast.ForStmt{}},
		{"foreach", "foreach (item in items) {}", ast.ForeachStmt{}},
		{"try catch", "try {} catch (err) {}", ast.TryCatchStmt{}},
		{"switch", "switch (value) { case 1: print(1); default: print(2); }", ast.SwitchStmt{}},
		{"break", "break;", ast.BreakStmt{}},
		{"continue", "continue;", ast.ContinueStmt{}},
		{"return", "return value;", ast.ReturnStmt{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, diagnostics := Parse([]byte(tt.source))
			if len(diagnostics) != 0 {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
			if len(program.Statements) != 1 {
				t.Fatalf("statements = %d, want 1", len(program.Statements))
			}
			if reflect.TypeOf(program.Statements[0]) != reflect.TypeOf(tt.want) {
				t.Fatalf("statement = %T, want %T", program.Statements[0], tt.want)
			}
			assertStatementContract(t, tt.source, program.Statements[0])
		})
	}

	t.Run("parameters cases and synthetic blocks", func(t *testing.T) {
		program, diagnostics := Parse([]byte("function f(value: integer, other) {} if (true) print(1); else print(2); switch (value) { case 1: print(1); default: print(2); }"))
		if len(diagnostics) != 0 || len(program.Statements) != 3 {
			t.Fatalf("program = %#v, diagnostics = %#v", program, diagnostics)
		}
		function := program.Statements[0].(ast.FunctionDeclStmt)
		if len(function.Parameters) != 2 || function.Parameters[0].Type == nil || function.Parameters[0].Type.Name != "integer" || function.Parameters[1].Type != nil {
			t.Fatalf("parameters = %#v", function.Parameters)
		}
		conditional := program.Statements[1].(ast.IfStmt)
		if conditional.Then == nil || conditional.Else == nil || len(conditional.Then.Statements) != 1 || len(conditional.Else.Statements) != 1 {
			t.Fatalf("synthetic blocks = %#v", conditional)
		}
		switchStmt := program.Statements[2].(ast.SwitchStmt)
		if len(switchStmt.Cases) != 2 || switchStmt.Cases[0].Default || !switchStmt.Cases[1].Default || len(switchStmt.Cases[0].Statements) != 1 {
			t.Fatalf("cases = %#v", switchStmt.Cases)
		}
	})

	t.Run("nested function parameters stay local", func(t *testing.T) {
		program, diagnostics := Parse([]byte("function outer(value: integer) { function inner(other: string) {} }"))
		outer := program.Statements[0].(ast.FunctionDeclStmt)
		inner := outer.Body.Statements[0].(ast.FunctionDeclStmt)
		if len(diagnostics) != 0 || len(outer.Parameters) != 1 || outer.Parameters[0].Name != "value" || len(inner.Parameters) != 1 || inner.Parameters[0].Name != "other" {
			t.Fatalf("outer=%#v inner=%#v diagnostics=%#v", outer.Parameters, inner.Parameters, diagnostics)
		}
	})
}

func assertStatementContract(t *testing.T, source string, statement ast.Statement) {
	t.Helper()
	if got, want := statement.SourceSpan(), testSpan(source, 0, len(source)); got != want {
		t.Fatalf("statement span = %#v, want %#v", got, want)
	}
	switch got := statement.(type) {
	case ast.VarDeclStmt:
		assertTypeAndLiteral(t, got.Name, got.Type, got.Initializer, "value", "integer", "1")
	case ast.ConstDeclStmt:
		assertTypeAndLiteral(t, got.Name, got.Type, got.Value, "value", "integer", "1")
	case ast.AssignStmt:
		assertIdentifierAndLiteral(t, got.Target, got.Value, "value", "1")
	case ast.FunctionDeclStmt:
		if got.Name != "f" || len(got.Parameters) != 2 || got.Parameters[0].Name != "value" || got.Parameters[0].Type == nil || got.Parameters[0].Type.Name != "integer" || got.Parameters[1].Name != "other" || got.Parameters[1].Type != nil || got.Result != nil || got.Body == nil || len(got.Body.Statements) != 0 {
			t.Fatalf("function = %#v", got)
		}
	case ast.ClassDeclStmt:
		if got.Name != "Child" || got.Parent != "Parent" || len(got.Members) != 1 {
			t.Fatalf("class = %#v", got)
		}
	case ast.ExprStmt:
		if call, ok := got.Expression.(ast.CallExpr); !ok || len(call.Arguments) != 0 {
			t.Fatalf("expression = %#v", got)
		}
	case ast.PrintStmt:
		assertLiteral(t, got.Value, "1")
	case ast.BlockStmt:
		if len(got.Statements) != 1 {
			t.Fatalf("block = %#v", got)
		}
	case ast.IfStmt:
		if got.Then == nil || got.Else == nil || len(got.Then.Statements) != 1 || len(got.Else.Statements) != 1 {
			t.Fatalf("if = %#v", got)
		}
	case ast.WhileStmt:
		if got.Body == nil || len(got.Body.Statements) != 0 {
			t.Fatalf("while = %#v", got)
		}
	case ast.DoWhileStmt:
		if got.Body == nil || len(got.Body.Statements) != 0 {
			t.Fatalf("do while = %#v", got)
		}
	case ast.ForStmt:
		if got.Init == nil || got.Condition == nil || got.Post == nil || got.Body == nil {
			t.Fatalf("for = %#v", got)
		}
	case ast.ForeachStmt:
		if got.Name != "item" || got.Iterable == nil || got.Body == nil {
			t.Fatalf("foreach = %#v", got)
		}
	case ast.TryCatchStmt:
		if got.Name != "err" || got.Try == nil || got.Catch == nil {
			t.Fatalf("try catch = %#v", got)
		}
	case ast.SwitchStmt:
		if got.Value == nil || len(got.Cases) != 2 || got.Cases[0].Default || !got.Cases[1].Default {
			t.Fatalf("switch = %#v", got)
		}
	case ast.ReturnStmt:
		assertIdentifier(t, got.Value, "value")
	}
}

func assertTypeAndLiteral(t *testing.T, name string, typ *ast.TypeRef, value ast.Expression, wantName, wantType, wantValue string) {
	t.Helper()
	if name != wantName || typ == nil || typ.Name != wantType || typ.Dimensions != 0 {
		t.Fatalf("name/type = %q/%#v", name, typ)
	}
	assertLiteral(t, value, wantValue)
}

func assertIdentifierAndLiteral(t *testing.T, target, value ast.Expression, wantTarget, wantValue string) {
	t.Helper()
	assertIdentifier(t, target, wantTarget)
	assertLiteral(t, value, wantValue)
}
func assertIdentifier(t *testing.T, value ast.Expression, want string) {
	t.Helper()
	if got, ok := value.(ast.IdentifierExpr); !ok || got.Name != want {
		t.Fatalf("identifier = %#v, want %q", value, want)
	}
}
func assertLiteral(t *testing.T, value ast.Expression, want string) {
	t.Helper()
	if got, ok := value.(ast.LiteralExpr); !ok || got.Lexeme != want {
		t.Fatalf("literal = %#v, want %q", value, want)
	}
}

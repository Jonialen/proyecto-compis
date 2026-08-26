package frontend

import (
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestLocalizedRecovery(t *testing.T) {
	t.Run("bad statement preserves valid siblings", func(t *testing.T) {
		program, diagnostics := Parse([]byte("let before: integer = 1; let broken: integer = ; let after: integer = 2;"))
		if len(diagnostics) == 0 {
			t.Fatal("diagnostics = none, want syntax diagnostic")
		}
		want := []ast.Statement{ast.VarDeclStmt{}, ast.BadStmt{}, ast.VarDeclStmt{}}
		if len(program.Statements) != len(want) {
			t.Fatalf("statements = %#v, want %d localized statements", program.Statements, len(want))
		}
		for i := range want {
			if reflect.TypeOf(program.Statements[i]) != reflect.TypeOf(want[i]) {
				t.Fatalf("statement[%d] = %T, want %T", i, program.Statements[i], want[i])
			}
		}
	})

	t.Run("bad expression stays inside its statement", func(t *testing.T) {
		program, diagnostics := Parse([]byte("print(1); print(1 + ); print(2);"))
		if len(diagnostics) == 0 || len(program.Statements) != 3 {
			t.Fatalf("program = %#v, diagnostics = %#v", program, diagnostics)
		}
		broken, ok := program.Statements[1].(ast.PrintStmt)
		if !ok {
			t.Fatalf("statement[1] = %T, want ast.PrintStmt", program.Statements[1])
		}
		binary, ok := broken.Value.(ast.BinaryExpr)
		if !ok {
			t.Fatalf("broken value = %T, want ast.BinaryExpr", broken.Value)
		}
		if _, ok := binary.Right.(ast.BadExpr); !ok {
			t.Fatalf("broken operand = %T, want ast.BadExpr", binary.Right)
		}
		for _, index := range []int{0, 2} {
			if _, ok := program.Statements[index].(ast.PrintStmt); !ok {
				t.Fatalf("statement[%d] = %T, want valid ast.PrintStmt", index, program.Statements[index])
			}
		}
	})
}

func TestOfficialExamples(t *testing.T) {
	for _, source := range []string{
		`let a: integer = 10; let b: string = "hola"; let c: boolean = true; let d = null; let lista = [1, 2, 3]; print(lista[0]);`,
		`function saludar(nombre: string): string { return "Hola " + nombre; } class Animal { let nombre: string; function constructor(nombre: string) { this.nombre = nombre; } } foreach (n in lista) { if (n < 60) continue; if (n == 100) break; print(n); }`,
	} {
		program, diagnostics := Parse([]byte(source))
		if len(diagnostics) != 0 {
			t.Fatalf("official example diagnostics = %#v", diagnostics)
		}
		if len(program.Statements) == 0 {
			t.Fatal("official example produced no statements")
		}
		for _, statement := range program.Statements {
			if _, bad := statement.(ast.BadStmt); bad {
				t.Fatalf("official example statement = %#v, want mapped statement", statement)
			}
		}
	}
}

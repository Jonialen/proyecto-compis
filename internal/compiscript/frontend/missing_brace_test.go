package frontend

import (
	"reflect"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func TestMissingClosingBrace(t *testing.T) {
	const source = "if (true) { print(1);"
	program, diagnostics := Parse([]byte(source))
	wantProgram := ast.Program{
		Span: testSpan(source, 0, len(source)),
		Statements: ast.Statements{
			ast.BadStmt{Span: testSpan(source, 0, 9)},
			ast.BadStmt{Span: testSpan(source, 10, len(source))},
		},
	}
	wantDiagnostics := model.Diagnostics{
		{
			Code: "syntax", Phase: model.PhaseSyntax,
			Message: "no viable alternative at input '{print(1);'",
			Span:    testSpan(source, len(source), len(source)),
		},
		{
			Code: "syntax", Phase: model.PhaseSyntax,
			Message: "extraneous input '<EOF>' expecting {'{', '}', 'let', 'var', 'const', 'print', '(', 'if', 'while', 'do', 'for', 'foreach', 'break', 'continue', 'return', 'try', 'switch', 'function', 'class', '-', '!', 'null', 'true', 'false', 'new', 'this', '[', Literal, Identifier}",
			Span:    testSpan(source, len(source), len(source)),
		},
	}
	if !reflect.DeepEqual(program, wantProgram) {
		t.Fatalf("program = %#v, want %#v", program, wantProgram)
	}
	if !reflect.DeepEqual(diagnostics, wantDiagnostics) {
		t.Fatalf("diagnostics = %#v, want %#v", diagnostics, wantDiagnostics)
	}
}

func TestMissingClosingBraceValidControls(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		wantElse     bool
		wantNested   bool
	}{
		{"closed counterpart", "if (true) { print(1); }", false, false},
		{"block else", "if (true) { print(1); } else { print(2); }", true, false},
		{"nested block", "if (true) { if (true) { print(1); } }", false, true},
		{"statement else", "if (true) print(1); else print(2);", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			program, diagnostics := Parse([]byte(tt.source))
			if len(diagnostics) != 0 || len(program.Statements) != 1 {
				t.Fatalf("program = %#v, diagnostics = %#v", program, diagnostics)
			}
			conditional, ok := program.Statements[0].(ast.IfStmt)
			if !ok || conditional.Span != testSpan(tt.source, 0, len(tt.source)) || conditional.Then == nil ||
				len(conditional.Then.Statements) != 1 || (conditional.Else != nil) != tt.wantElse {
				t.Fatalf("conditional = %#v", program.Statements[0])
			}
			assertLiteral(t, conditional.Condition, "true")
			body := conditional.Then.Statements[0]
			if tt.wantNested {
				nested, ok := body.(ast.IfStmt)
				if !ok || nested.Then == nil || len(nested.Then.Statements) != 1 || nested.Else != nil {
					t.Fatalf("nested conditional = %#v", body)
				}
				assertLiteral(t, nested.Condition, "true")
				body = nested.Then.Statements[0]
			}
			print, ok := body.(ast.PrintStmt)
			if !ok {
				t.Fatalf("body = %T, want ast.PrintStmt", body)
			}
			assertLiteral(t, print.Value, "1")
			if tt.wantElse {
				if len(conditional.Else.Statements) != 1 {
					t.Fatalf("else = %#v", conditional.Else)
				}
				print, ok := conditional.Else.Statements[0].(ast.PrintStmt)
				if !ok {
					t.Fatalf("else body = %T, want ast.PrintStmt", conditional.Else.Statements[0])
				}
				assertLiteral(t, print.Value, "2")
			}
		})
	}
}

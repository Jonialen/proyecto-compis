package frontend

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestForClauseRoles(t *testing.T) {
	for _, init := range []string{";", "let i=0;", "i=0;"} {
		for _, condition := range []string{"", "i<3"} {
			for _, post := range []string{"", "i=i+1"} {
				source := fmt.Sprintf("for (%s %s; %s) {}", init, condition, post)
				t.Run(source, func(t *testing.T) {
					input := []byte(source)
					before := bytes.Clone(input)
					program, diagnostics := Parse(input)
					if len(diagnostics) != 0 || len(program.Statements) != 1 {
						t.Fatalf("parse: %+v %+v", program, diagnostics)
					}
					n := program.Statements[0].(ast.ForStmt)
					if (n.Init == nil) != (init == ";") || (n.Condition == nil) != (condition == "") || (n.Post == nil) != (post == "") {
						t.Fatalf("clause roles: init=%T condition=%T post=%T", n.Init, n.Condition, n.Post)
					}
					if init != ";" {
						switch init {
						case "let i=0;":
							if _, ok := n.Init.(ast.VarDeclStmt); !ok {
								t.Fatalf("init=%T", n.Init)
							}
						case "i=0;":
							if _, ok := n.Init.(ast.AssignStmt); !ok {
								t.Fatalf("init=%T", n.Init)
							}
						}
						start := strings.Index(source, init)
						if n.Init.SourceSpan() != testSpan(source, start, start+len(init)) {
							t.Fatal("initializer span changed")
						}
					}
					if condition != "" {
						binary, ok := n.Condition.(ast.BinaryExpr)
						if !ok || binary.Operator != "<" {
							t.Fatalf("condition=%+v", n.Condition)
						}
						start := strings.Index(source, condition)
						if binary.Span != testSpan(source, start, start+len(condition)) {
							t.Fatal("condition span changed")
						}
					}
					if post != "" {
						assignment, ok := n.Post.(ast.AssignExpr)
						if !ok {
							t.Fatalf("post=%T", n.Post)
						}
						assertIdentifier(t, assignment.Target, "i")
						start := strings.Index(source, post)
						if assignment.Span != testSpan(source, start, start+len(post)) {
							t.Fatal("post span changed")
						}
					}
					if n.Span != testSpan(source, 0, len(source)) || !bytes.Equal(input, before) {
						t.Fatal("source/loop span changed")
					}
				})
			}
		}
	}
	t.Run("boolean post is not condition", func(t *testing.T) {
		program, diagnostics := Parse([]byte("for (;true;false) {}"))
		if len(diagnostics) != 0 {
			t.Fatal(diagnostics)
		}
		n := program.Statements[0].(ast.ForStmt)
		assertLiteral(t, n.Condition, "true")
		assertLiteral(t, n.Post, "false")
	})
	t.Run("omitted condition with boolean post", func(t *testing.T) {
		program, diagnostics := Parse([]byte("for (;;false) {}"))
		if len(diagnostics) != 0 {
			t.Fatal(diagnostics)
		}
		n := program.Statements[0].(ast.ForStmt)
		if n.Condition != nil {
			t.Fatalf("condition=%+v", n.Condition)
		}
		assertLiteral(t, n.Post, "false")
	})
}

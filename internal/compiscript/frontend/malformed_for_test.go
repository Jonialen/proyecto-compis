package frontend

import (
	"crypto/sha256"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestMalformedForMissingBody(t *testing.T) {
	source := []byte("for(let i=0; i<;i=i+1){")
	before := sha256.Sum256(source)
	program, diagnostics := Parse(source)
	if len(diagnostics) == 0 {
		t.Fatal("missing syntax diagnostics")
	}
	// Keep the parser's original locations rather than replacing them with a
	// synthetic mapper error or accepting the recovered header as a valid loop.
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostics = %+v, want three parser diagnostics", diagnostics)
	}
	for i, offset := range []int{15, 17, 23} {
		d := diagnostics[i]
		if d.Phase != "syntax" || d.Code != "syntax" || d.Span.Start.Offset != offset || d.Span.Start.Line != 1 || d.Span.Start.Column != offset+1 {
			t.Fatalf("diagnostic[%d] = %+v", i, d)
		}
	}
	for _, d := range diagnostics {
		if d.Span.Start.Line < 1 || d.Span.Start.Column < 1 {
			t.Fatalf("unlocated diagnostic: %+v", d)
		}
	}
	if sha256.Sum256(source) != before {
		t.Fatal("source changed")
	}
	if len(program.Statements) == 0 {
		t.Fatal("recovery lost the rejected statement")
	}
	if _, ok := program.Statements[0].(ast.BadStmt); !ok {
		t.Fatalf("recovery fabricated a valid loop: %T", program.Statements[0])
	}
}

func TestMalformedForPreservesPrecedingDeclaration(t *testing.T) {
	program, diagnostics := Parse([]byte("let before=1;\nfor(let i=0; i<;i=i+1){"))
	if len(diagnostics) == 0 || len(program.Statements) < 2 {
		t.Fatalf("missing recovery: %+v, %+v", program, diagnostics)
	}
	declaration, ok := program.Statements[0].(ast.VarDeclStmt)
	if !ok || declaration.Name != "before" {
		t.Fatalf("lost valid preceding declaration: %+v", program.Statements[0])
	}
	if _, ok := program.Statements[1].(ast.BadStmt); !ok {
		t.Fatalf("malformed loop not rejected locally: %T", program.Statements[1])
	}
}

func TestMissingLoopBodies(t *testing.T) {
	for _, source := range []string{
		"while(true)", "do", "foreach(x in [1])", "for(let i=0; i<2; i=i+1)",
	} {
		t.Run(source, func(t *testing.T) {
			program, diagnostics := Parse([]byte(source))
			if len(diagnostics) == 0 || len(program.Statements) == 0 {
				t.Fatalf("missing recovered statement/diagnostics: %+v, %+v", program, diagnostics)
			}
			for _, d := range diagnostics {
				if d.Span.Start.Line != 1 || d.Span.Start.Column < 1 {
					t.Fatalf("unlocated diagnostic: %+v", d)
				}
			}
			if _, ok := program.Statements[0].(ast.BadStmt); !ok {
				t.Fatalf("missing body became valid: %T", program.Statements[0])
			}
		})
	}
}

package frontend

import (
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func TestParseReportsLexerAndParserDiagnosticsInEncounterOrder(t *testing.T) {
	_, diagnostics := Parse([]byte("é\nlet value: integer = ;"))
	if len(diagnostics) < 2 {
		t.Fatalf("diagnostic count = %d, want at least lexical and syntax diagnostics", len(diagnostics))
	}

	if got, want := diagnostics[0].Phase, model.PhaseLexical; got != want {
		t.Fatalf("first diagnostic phase = %q, want %q", got, want)
	}
	if got, want := diagnostics[0].Span, (ast.Span{Start: ast.Position{Offset: 0, Line: 1, Column: 1}, End: ast.Position{Offset: 2, Line: 1, Column: 2}}); got != want {
		t.Fatalf("first diagnostic span = %#v, want %#v", got, want)
	}
	if got, want := diagnostics[1].Phase, model.PhaseSyntax; got != want {
		t.Fatalf("second diagnostic phase = %q, want %q", got, want)
	}
}

func TestParseOrdersMixedDiagnosticsBySourceEncounter(t *testing.T) {
	_, diagnostics := Parse([]byte("let first: integer = ; @ let second: integer = ;"))
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostic count = %d, want 3", len(diagnostics))
	}

	want := []struct {
		phase model.Phase
		start int
		end   int
	}{
		{phase: model.PhaseSyntax, start: 21, end: 22},
		{phase: model.PhaseLexical, start: 23, end: 24},
		{phase: model.PhaseSyntax, start: 47, end: 48},
	}
	for i, expected := range want {
		got := diagnostics[i]
		if got.Phase != expected.phase || got.Span.Start.Offset != expected.start || got.Span.End.Offset != expected.end {
			t.Fatalf("diagnostic[%d] = %#v, want phase %q and span [%d,%d)", i, got, expected.phase, expected.start, expected.end)
		}
	}
}

func TestOrderDiagnosticsPreservesInputOrderForEqualOffsets(t *testing.T) {
	first := model.Diagnostic{Phase: model.PhaseSyntax, Span: ast.Span{Start: ast.Position{Offset: 7}, End: ast.Position{Offset: 7}}}
	second := model.Diagnostic{Phase: model.PhaseLexical, Span: ast.Span{Start: ast.Position{Offset: 7}, End: ast.Position{Offset: 8}}}
	ordered := orderDiagnostics(model.Diagnostics{first, second})
	if len(ordered) != 2 || ordered[0] != first || ordered[1] != second {
		t.Fatalf("ordered diagnostics = %#v, want %#v followed by %#v", ordered, first, second)
	}
}

func TestParseDiagnosticSpansCoverDeletionAndUseZeroWidthForInsertionAndEOF(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   ast.Span
	}{
		{
			name:   "deletion covers offending token",
			source: "let value: integer = 1 2;",
			want:   ast.Span{Start: ast.Position{Offset: 23, Line: 1, Column: 24}, End: ast.Position{Offset: 24, Line: 1, Column: 25}},
		},
		{
			name:   "insertion is zero width at lookahead",
			source: "let value: integer = 1 let next: integer = 2;",
			want:   ast.Span{Start: ast.Position{Offset: 23, Line: 1, Column: 24}, End: ast.Position{Offset: 23, Line: 1, Column: 24}},
		},
		{
			name:   "end of input is zero width",
			source: "let value: integer = 1",
			want:   ast.Span{Start: ast.Position{Offset: 22, Line: 1, Column: 23}, End: ast.Position{Offset: 22, Line: 1, Column: 23}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, diagnostics := Parse([]byte(test.source))
			if len(diagnostics) == 0 {
				t.Fatal("diagnostic count = 0, want one syntax diagnostic")
			}
			got := diagnostics[len(diagnostics)-1]
			if got.Phase != model.PhaseSyntax {
				t.Fatalf("diagnostic phase = %q, want %q", got.Phase, model.PhaseSyntax)
			}
			if got.Span != test.want {
				t.Fatalf("diagnostic span = %#v, want %#v; message = %q", got.Span, test.want, got.Message)
			}
		})
	}
}

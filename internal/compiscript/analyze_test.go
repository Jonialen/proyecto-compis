package compiscript

import (
	"reflect"
	"slices"
	"testing"

	"genanalex/internal/compiscript/model"
)

func flatten(view model.ASTView, kinds *[]string) {
	*kinds = append(*kinds, view.Kind)
	if view.Children == nil {
		panic("AST view children must not be nil")
	}
	for _, child := range view.Children {
		flatten(child, kinds)
	}
}

func hasLabel(view model.ASTView, label string) bool {
	if view.Label == label {
		return true
	}
	for _, child := range view.Children {
		if hasLabel(child, label) {
			return true
		}
	}
	return false
}

func TestAnalyzeVisualASTDeterministic(t *testing.T) {
	source := []byte(`let data = [1, 2]; function choose(value: integer): integer { if (true) { return value; } else { return 0; } } data[0]; data.size; let changed = data.size = 2; choose(1); let selected = true ? 1 : 2; let made = new Box();`)
	first, second := Analyze(source), Analyze(source)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated reports differ")
	}
	var kinds []string
	flatten(first.AST, &kinds)
	for _, kind := range []string{"program", "variable", "type", "array", "function", "parameter", "block", "if", "return", "identifier", "index", "property", "property-assignment", "call", "ternary", "new"} {
		if !slices.Contains(kinds, kind) {
			t.Errorf("visual AST lacks %q: %v", kind, kinds)
		}
	}
	if len(first.Scopes) < 4 || first.Scopes[0].Symbols[0].Name != "data" || first.Scopes[0].Symbols[1].Name != "choose" {
		t.Fatalf("unordered scopes/symbols: %+v", first.Scopes)
	}
	if first.AST.Span.Start.Offset != 0 || first.AST.Span.End.Offset != len(source) {
		t.Fatalf("program span = %+v", first.AST.Span)
	}
}

func TestAnalyzePreservesFrontendDiagnostics(t *testing.T) {
	report := Analyze([]byte("let broken: integer = ; @"))
	lexical, syntax := false, false
	for _, diagnostic := range report.Diagnostics {
		lexical = lexical || diagnostic.Phase == model.PhaseLexical
		syntax = syntax || diagnostic.Phase == model.PhaseSyntax
	}
	if !lexical || !syntax || report.AST.Children == nil {
		t.Fatalf("frontend diagnostics/report lost: %+v", report)
	}
}

func TestAnalyzeVisualASTAllStatements(t *testing.T) {
	source := []byte(`const fixed: integer = 1; let values: integer[] = [1]; fixed = 2; while (true) { break; } do { continue; } while (false); for (let i: integer = 0; true; i = i + 1) { print(i); } foreach (item in [1]) { print(item); } try { print(1); } catch (err) { print(err); } switch (1) { case 1: print(1); default: print(0); } class Box: Parent { let field: integer = 1; }`)
	report := Analyze(source)
	var kinds []string
	flatten(report.AST, &kinds)
	for _, kind := range []string{"constant", "assignment", "while", "break", "do-while", "continue", "for", "foreach", "try-catch", "switch", "case", "class", "print"} {
		if !slices.Contains(kinds, kind) {
			t.Errorf("visual AST lacks %q: %v", kind, kinds)
		}
	}
	for _, label := range []string{"integer[]", "default", "Box : Parent"} {
		if !hasLabel(report.AST, label) {
			t.Errorf("visual AST lacks label %q", label)
		}
	}
}

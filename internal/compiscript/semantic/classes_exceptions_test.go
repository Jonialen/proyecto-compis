package semantic_test

import (
	"slices"
	"strings"
	"testing"

	"genanalex/internal/compiscript/model"
)

func TestClassesMembersInheritanceConstructorsAndThis(t *testing.T) {
	scopes, diagnostics := analyzeFixture(t, "classes_valid")
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
	var classes []model.ScopeSnapshot
	for _, current := range scopes {
		if current.Kind == model.ScopeClass {
			classes = append(classes, current)
		}
	}
	if len(classes) != 5 {
		t.Fatalf("class scopes = %+v", classes)
	}
	got := []string{}
	for _, symbol := range classes[0].Symbols {
		got = append(got, symbol.Name)
	}
	if !slices.Equal(got, []string{"name", "constructor", "speak"}) || classes[0].Symbols[0].Kind != model.SymbolField || classes[0].Symbols[1].Kind != model.SymbolMethod {
		t.Fatalf("ordered members = %+v", classes[0].Symbols)
	}
}

func TestClassesRejectInvalidHierarchyMembersAndConstruction(t *testing.T) {
	_, diagnostics := analyzeFixture(t, "classes_invalid")
	requireCounts(t, diagnostics, map[string]int{
		"SEM_ARGUMENT": 1, "SEM_ARITY": 2, "SEM_DUPLICATE": 1,
		"SEM_CONSTANT_ASSIGNMENT": 1, "SEM_TYPE": 1,
		"SEM_INHERITED_MEMBER": 1, "SEM_INHERITANCE_CYCLE": 2,
		"SEM_MEMBER": 2, "SEM_THIS": 2, "SEM_UNKNOWN_BASE": 1,
		"SEM_UNKNOWN_CLASS": 2,
	})
}

func TestCatchBindingScopeAndBothBlocks(t *testing.T) {
	scopes, diagnostics := analyzeFixture(t, "catch_valid")
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
	found := false
	wantOffset := strings.Index(string(fixture(t, "catch_valid")), "problem")
	for _, current := range scopes {
		for _, symbol := range current.Symbols {
			found = found || current.Kind == model.ScopeCatch && symbol.Name == "problem" && symbol.Kind == model.SymbolCatch && symbol.Type.Kind == model.TypeException && symbol.Span.Start.Offset == wantOffset && symbol.Span.End.Offset == wantOffset+len("problem")
		}
	}
	if !found {
		t.Fatalf("catch binding missing from scopes: %+v", scopes)
	}

	_, diagnostics = analyzeFixture(t, "catch_invalid")
	requireCounts(t, diagnostics, map[string]int{"SEM_UNRESOLVED": 3})
}

package semantic_test

import (
	"reflect"
	"slices"
	"testing"

	"genanalex/internal/compiscript/model"
)

func TestCollectionsInferContextualEmptyAndRaggedTypes(t *testing.T) {
	scopes, diagnostics := analyzeFixture(t, "collections_valid")
	if len(diagnostics) != 0 {
		t.Fatalf("semantic diagnostics: %+v", diagnostics)
	}
	symbols := map[string]model.Type{}
	for _, symbol := range scopes[0].Symbols {
		symbols[symbol.Name] = symbol.Type
	}
	tests := []struct {
		name  string
		depth int
	}{
		{"numbers", 1}, {"ragged", 2}, {"empty", 1}, {"nestedEmpty", 2}, {"trailingEmpty", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeOf := symbols[tt.name]
			for range tt.depth {
				if typeOf.Kind != model.TypeList || typeOf.Element == nil {
					t.Fatalf("%s type = %+v", tt.name, symbols[tt.name])
				}
				typeOf = *typeOf.Element
			}
			if typeOf.Kind != model.TypeInteger {
				t.Fatalf("%s element type = %+v", tt.name, typeOf)
			}
		})
	}
	again, repeatedDiagnostics := analyzeFixture(t, "collections_valid")
	if !reflect.DeepEqual(scopes, again) || !reflect.DeepEqual(diagnostics, repeatedDiagnostics) {
		t.Fatal("collection analysis changed across identical runs")
	}
}

func TestCollectionsRejectInvalidLiteralsIndicesAssignmentsAndBounds(t *testing.T) {
	_, diagnostics := analyzeFixture(t, "collections_invalid")
	wantCodes := []string{"SEM_EMPTY_COLLECTION", "SEM_COLLECTION", "SEM_COLLECTION", "SEM_BOUNDS", "SEM_BOUNDS", "SEM_BOUNDS", "SEM_INDEX", "SEM_TYPE", "SEM_INDEX", "SEM_INDEX", "SEM_TYPE", "SEM_EMPTY_COLLECTION", "SEM_BOUNDS", "SEM_BOUNDS", "SEM_EMPTY_COLLECTION", "SEM_BOUNDS", "SEM_UNRESOLVED", "SEM_BOUNDS"}
	wantLines := []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 12, 13, 14, 16, 19, 21, 22, 24, 25}
	if got := codes(diagnostics); !slices.Equal(got, wantCodes) {
		t.Fatalf("diagnostic codes = %v, want %v: %+v", got, wantCodes, diagnostics)
	}
	for i, diagnostic := range diagnostics {
		if diagnostic.Phase != model.PhaseSemantic || diagnostic.Span.Start.Line != wantLines[i] || diagnostic.Span.Start.Column < 1 {
			t.Fatalf("diagnostic %d is not deterministically located: %+v", i, diagnostic)
		}
	}
	_, repeated := analyzeFixture(t, "collections_invalid")
	if !reflect.DeepEqual(diagnostics, repeated) {
		t.Fatalf("diagnostics changed across runs: %+v then %+v", diagnostics, repeated)
	}
}

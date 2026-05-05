package yapar

import (
	"reflect"
	"testing"

	"genanalex/internal/shared"
)

func TestSetHelpers(t *testing.T) {
	t.Run("merge and sort", func(t *testing.T) {
		left := NewSet("B", "A")
		changed := left.Merge(NewSet("C", "A"))
		if !changed {
			t.Fatal("expected merge to report changes")
		}

		got := left.Sorted()
		want := []string{"A", "B", "C"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sorted set = %v, want %v", got, want)
		}
	})
}

func TestFilterIgnoredTokens(t *testing.T) {
	tokens := []shared.Token{{Type: "WS"}, {Type: "ID", Lexeme: "x", Line: 1}}
	got := FilterIgnoredTokens(tokens, map[string]bool{"WS": true})
	if len(got) != 1 || got[0].Type != "ID" {
		t.Fatalf("filtered tokens = %#v, want only ID", got)
	}
}

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

func TestComputeFirstFollowArithmeticGrammar(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS TIMES LPAREN RPAREN
%%
expr : expr PLUS term | term ;
term : term TIMES factor | factor ;
factor : ID | LPAREN expr RPAREN ;
`)

	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	assertSetEquals(t, ff.First["expr"], "ID", "LPAREN")
	assertSetEquals(t, ff.First["term"], "ID", "LPAREN")
	assertSetEquals(t, ff.First["factor"], "ID", "LPAREN")
	assertSetEquals(t, ff.First[g.Augmented], "ID", "LPAREN")
	assertSetEquals(t, ff.First["ID"], "ID")

	if ff.Nullable["expr"] || ff.Nullable["term"] || ff.Nullable["factor"] || ff.Nullable[g.Augmented] {
		t.Fatalf("unexpected nullable map = %#v", ff.Nullable)
	}

	assertSetEquals(t, ff.Follow[g.Augmented], EndMarker)
	assertSetEquals(t, ff.Follow["expr"], EndMarker, "PLUS", "RPAREN")
	assertSetEquals(t, ff.Follow["term"], EndMarker, "PLUS", "RPAREN", "TIMES")
	assertSetEquals(t, ff.Follow["factor"], EndMarker, "PLUS", "RPAREN", "TIMES")
}

func TestComputeFirstFollowWithEpsilonAndNullableSuffixes(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : a b ;
a : A | ;
b : B | ;
`)

	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	for _, nonTerminal := range []string{"s", "a", "b", g.Augmented} {
		if !ff.Nullable[nonTerminal] {
			t.Fatalf("Nullable[%q] = false, want true", nonTerminal)
		}
	}

	assertSetEquals(t, ff.First["a"], "A", Epsilon)
	assertSetEquals(t, ff.First["b"], "B", Epsilon)
	assertSetEquals(t, ff.First["s"], "A", "B", Epsilon)
	assertSetEquals(t, ff.First[g.Augmented], "A", "B", Epsilon)

	assertSetEquals(t, ff.Follow[g.Augmented], EndMarker)
	assertSetEquals(t, ff.Follow["s"], EndMarker)
	assertSetEquals(t, ff.Follow["a"], "B", EndMarker)
	assertSetEquals(t, ff.Follow["b"], EndMarker)
}

func TestComputeFirstFollowTransitiveNullable(t *testing.T) {
	g := mustBuildGrammar(t, `%token A
%%
s : a ;
a : b ;
b : ;
`)

	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	for _, nonTerminal := range []string{"s", "a", "b", g.Augmented} {
		if !ff.Nullable[nonTerminal] {
			t.Fatalf("Nullable[%q] = false, want true", nonTerminal)
		}
	}

	assertSetEquals(t, ff.First["b"], Epsilon)
	assertSetEquals(t, ff.First["a"], Epsilon)
	assertSetEquals(t, ff.First["s"], Epsilon)
	assertSetEquals(t, ff.Follow["b"], EndMarker)
}

func TestFirstOfSequenceHandlesNullablePrefixes(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : a b ;
a : A | ;
b : B | ;
`)

	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	assertSetEquals(t, FirstOfSequence([]Symbol{{Name: "a"}, {Name: "b"}}, ff), "A", "B", Epsilon)
	assertSetEquals(t, FirstOfSequence([]Symbol{{Name: "a"}, {Name: "B", Terminal: true}}, ff), "A", "B")
	assertSetEquals(t, FirstOfSequence([]Symbol{{Name: "A", Terminal: true}, {Name: "b"}}, ff), "A")
	assertSetEquals(t, FirstOfSequence(nil, ff), Epsilon)
}

func mustBuildGrammar(t *testing.T, input string) *Grammar {
	t.Helper()

	spec, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	g, err := BuildGrammar(spec)
	if err != nil {
		t.Fatalf("BuildGrammar() error = %v", err)
	}

	return g
}

func assertSetEquals(t *testing.T, got Set, wantItems ...string) {
	t.Helper()

	gotSorted := got.Sorted()
	wantSorted := NewSet(wantItems...).Sorted()
	if !reflect.DeepEqual(gotSorted, wantSorted) {
		t.Fatalf("set = %v, want %v", gotSorted, wantSorted)
	}
}

package yapar

import (
	"reflect"
	"testing"
)

func TestClosureExpandsNonTerminalItemsWithoutDuplicates(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)

	got := Closure(g, []Item{{ProductionID: 0, Dot: 0}, {ProductionID: 0, Dot: 0}})
	want := []Item{{ProductionID: 0, Dot: 0}, {ProductionID: 1, Dot: 0}, {ProductionID: 2, Dot: 0}, {ProductionID: 3, Dot: 0}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Closure() = %#v, want %#v", got, want)
	}
}

func TestGotoBuildsShiftedClosuresForTerminalAndNonTerminal(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)
	initial := Closure(g, []Item{{ProductionID: 0, Dot: 0}})

	t.Run("terminal", func(t *testing.T) {
		got := Goto(g, initial, "D")
		want := []Item{{ProductionID: 3, Dot: 1}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Goto(D) = %#v, want %#v", got, want)
		}
	})

	t.Run("non-terminal", func(t *testing.T) {
		got := Goto(g, initial, "c")
		want := []Item{{ProductionID: 1, Dot: 1}, {ProductionID: 2, Dot: 0}, {ProductionID: 3, Dot: 0}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Goto(c) = %#v, want %#v", got, want)
		}
	})

	t.Run("unknown symbol", func(t *testing.T) {
		got := Goto(g, initial, "missing")
		if len(got) != 0 {
			t.Fatalf("Goto(missing) = %#v, want empty", got)
		}
	})
}

func TestBuildCanonicalCollectionBuildsKnownLR0Automaton(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)

	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}

	if len(states) != 7 {
		t.Fatalf("len(states) = %d, want 7", len(states))
	}

	for i, state := range states {
		if state.ID != i {
			t.Fatalf("state[%d].ID = %d, want %d", i, state.ID, i)
		}
	}

	wantInitial := []Item{{ProductionID: 0, Dot: 0}, {ProductionID: 1, Dot: 0}, {ProductionID: 2, Dot: 0}, {ProductionID: 3, Dot: 0}}
	if !reflect.DeepEqual(states[0].Items, wantInitial) {
		t.Fatalf("initial state items = %#v, want %#v", states[0].Items, wantInitial)
	}

	stateOnC := states[transitions[0]["C"]].Items
	wantStateThree := []Item{{ProductionID: 2, Dot: 0}, {ProductionID: 2, Dot: 1}, {ProductionID: 3, Dot: 0}}
	if !reflect.DeepEqual(stateOnC, wantStateThree) {
		t.Fatalf("state on C = %#v, want %#v", stateOnC, wantStateThree)
	}

	acceptStateID, ok := transitions[0]["s"]
	if !ok {
		t.Fatal("missing transition 0 --s-->")
	}
	if got := states[acceptStateID].Items; !reflect.DeepEqual(got, []Item{{ProductionID: 0, Dot: 1}}) {
		t.Fatalf("state on s = %#v, want augmented accept item", got)
	}

	stateOncID, ok := transitions[0]["c"]
	if !ok {
		t.Fatal("missing transition 0 --c-->")
	}
	if got := states[stateOncID].Items; !reflect.DeepEqual(got, []Item{{ProductionID: 1, Dot: 1}, {ProductionID: 2, Dot: 0}, {ProductionID: 3, Dot: 0}}) {
		t.Fatalf("state on c = %#v, want shifted start continuation", got)
	}

	stateOnDID, ok := transitions[0]["D"]
	if !ok {
		t.Fatal("missing transition 0 --D-->")
	}
	if got := states[stateOnDID].Items; !reflect.DeepEqual(got, []Item{{ProductionID: 3, Dot: 1}}) {
		t.Fatalf("state on D = %#v, want reduced c -> D item", got)
	}

	completedS, ok := transitions[stateOncID]["c"]
	if !ok {
		t.Fatal("missing transition (0 --c--) --c-->")
	}
	if got := states[completedS].Items; !reflect.DeepEqual(got, []Item{{ProductionID: 1, Dot: 2}}) {
		t.Fatalf("completed s state = %#v, want s -> c c .", got)
	}

	if transitions[transitions[0]["C"]]["C"] != transitions[0]["C"] {
		t.Fatalf("expected self-loop on symbol C from state reached by 0 --C-->")
	}
	if got := states[transitions[transitions[0]["C"]]["c"]].Items; !reflect.DeepEqual(got, []Item{{ProductionID: 2, Dot: 2}}) {
		t.Fatalf("completed c state = %#v, want c -> C c .", got)
	}
}

func TestLR1SignaturesTrackLookaheadsAndCoreSeparately(t *testing.T) {
	items := []LR1Item{{ProductionID: 2, Dot: 1, Lookahead: "$"}, {ProductionID: 1, Dot: 0, Lookahead: "ID"}, {ProductionID: 2, Dot: 1, Lookahead: "PLUS"}}

	if got, want := lr1ItemKey(items[0]), "2:1:$"; got != want {
		t.Fatalf("lr1ItemKey() = %q, want %q", got, want)
	}
	if got, want := lr1StateSignature(items), "1:0:ID|2:1:$|2:1:PLUS"; got != want {
		t.Fatalf("lr1StateSignature() = %q, want %q", got, want)
	}
	if got, want := coreSignature(items), "1:0|2:1"; got != want {
		t.Fatalf("coreSignature() = %q, want %q", got, want)
	}
}

func TestClosureLR1PropagatesLookaheadsWithoutDuplicates(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	got := ClosureLR1(g, ff, []LR1Item{{ProductionID: 0, Dot: 0, Lookahead: "$"}, {ProductionID: 0, Dot: 0, Lookahead: "$"}})
	want := []LR1Item{
		{ProductionID: 0, Dot: 0, Lookahead: "$"},
		{ProductionID: 1, Dot: 0, Lookahead: "$"},
		{ProductionID: 2, Dot: 0, Lookahead: "C"},
		{ProductionID: 2, Dot: 0, Lookahead: "D"},
		{ProductionID: 3, Dot: 0, Lookahead: "C"},
		{ProductionID: 3, Dot: 0, Lookahead: "D"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ClosureLR1() = %#v, want %#v", got, want)
	}
}

func TestGotoLR1BuildsShiftedLookaheadAwareClosure(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	initial := ClosureLR1(g, ff, []LR1Item{{ProductionID: 0, Dot: 0, Lookahead: "$"}})

	t.Run("non-terminal", func(t *testing.T) {
		got := GotoLR1(g, ff, initial, "c")
		want := []LR1Item{
			{ProductionID: 1, Dot: 1, Lookahead: "$"},
			{ProductionID: 2, Dot: 0, Lookahead: "$"},
			{ProductionID: 3, Dot: 0, Lookahead: "$"},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GotoLR1(c) = %#v, want %#v", got, want)
		}
	})

	t.Run("terminal", func(t *testing.T) {
		got := GotoLR1(g, ff, initial, "D")
		want := []LR1Item{{ProductionID: 3, Dot: 1, Lookahead: "C"}, {ProductionID: 3, Dot: 1, Lookahead: "D"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GotoLR1(D) = %#v, want %#v", got, want)
		}
	})
}

func TestBuildLR1CollectionBuildsKnownCanonicalAutomaton(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	states, transitions, err := BuildLR1Collection(g, ff)
	if err != nil {
		t.Fatalf("BuildLR1Collection() error = %v", err)
	}

	if len(states) != 10 {
		t.Fatalf("len(states) = %d, want 10", len(states))
	}
	for i, state := range states {
		if state.ID != i {
			t.Fatalf("state[%d].ID = %d, want %d", i, state.ID, i)
		}
	}

	wantInitial := []LR1Item{
		{ProductionID: 0, Dot: 0, Lookahead: "$"},
		{ProductionID: 1, Dot: 0, Lookahead: "$"},
		{ProductionID: 2, Dot: 0, Lookahead: "C"},
		{ProductionID: 2, Dot: 0, Lookahead: "D"},
		{ProductionID: 3, Dot: 0, Lookahead: "C"},
		{ProductionID: 3, Dot: 0, Lookahead: "D"},
	}
	if !reflect.DeepEqual(states[0].Items, wantInitial) {
		t.Fatalf("initial LR(1) state = %#v, want %#v", states[0].Items, wantInitial)
	}

	acceptStateID, ok := transitions[0]["s"]
	if !ok {
		t.Fatal("missing transition 0 --s-->")
	}
	if got := states[acceptStateID].Items; !reflect.DeepEqual(got, []LR1Item{{ProductionID: 0, Dot: 1, Lookahead: "$"}}) {
		t.Fatalf("state on s = %#v, want augmented accept item", got)
	}

	stateOncID, ok := transitions[0]["c"]
	if !ok {
		t.Fatal("missing transition 0 --c-->")
	}
	if got := states[stateOncID].Items; !reflect.DeepEqual(got, []LR1Item{{ProductionID: 1, Dot: 1, Lookahead: "$"}, {ProductionID: 2, Dot: 0, Lookahead: "$"}, {ProductionID: 3, Dot: 0, Lookahead: "$"}}) {
		t.Fatalf("state on c = %#v, want shifted lookahead-aware continuation", got)
	}
}

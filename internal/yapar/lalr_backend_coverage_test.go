package yapar

import (
	"reflect"
	"testing"
)

func TestBuildLALRParserRejectsNilFirstFollow(t *testing.T) {
	g := mustBuildGrammar(t, `%token A
%%
s : A ;
`)

	parser, err := buildLALRParser(g, nil)
	if err == nil {
		t.Fatal("buildLALRParser(g, nil) error = nil, want error")
	}
	if parser != nil {
		t.Fatalf("buildLALRParser(g, nil) parser = %#v, want nil", parser)
	}
	if got, want := err.Error(), "yapar: first/follow data is required to build LR1 collection"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestLALRTableViewDefensiveBranches(t *testing.T) {
	t.Run("nil and missing tables return falsey lookups", func(t *testing.T) {
		var nilView *lalrTableView
		if kind, target, ok := nilView.ActionAt(0, "ID"); ok || kind != ActionError || target != 0 {
			t.Fatalf("nilView.ActionAt() = (%v, %d, %v), want (ActionError, 0, false)", kind, target, ok)
		}
		if target, ok := nilView.GotoAt(0, "expr"); ok || target != 0 {
			t.Fatalf("nilView.GotoAt() = (%d, %v), want (0, false)", target, ok)
		}
		if states := nilView.States(); states != nil {
			t.Fatalf("nilView.States() = %v, want nil", states)
		}
		if terminals := nilView.Terminals(); terminals != nil {
			t.Fatalf("nilView.Terminals() = %v, want nil", terminals)
		}
		if nonTerminals := nilView.NonTerminals(); nonTerminals != nil {
			t.Fatalf("nilView.NonTerminals() = %v, want nil", nonTerminals)
		}
	})

	t.Run("missing rows and symbols return false without panicking", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
		view := &lalrTableView{
			grammar: g,
			table: &ParsingTable{
				Action: map[int]map[string]Action{0: {"ID": {Kind: ActionShift, TargetState: 1}}},
				Goto:   map[int]map[string]int{0: {"expr": 2}},
			},
		}

		if kind, target, ok := view.ActionAt(0, "PLUS"); ok || kind != ActionError || target != 0 {
			t.Fatalf("ActionAt(0, PLUS) = (%v, %d, %v), want (ActionError, 0, false)", kind, target, ok)
		}
		if kind, target, ok := view.ActionAt(9, "ID"); ok || kind != ActionError || target != 0 {
			t.Fatalf("ActionAt(9, ID) = (%v, %d, %v), want (ActionError, 0, false)", kind, target, ok)
		}
		if target, ok := view.GotoAt(0, "term"); ok || target != 0 {
			t.Fatalf("GotoAt(0, term) = (%d, %v), want (0, false)", target, ok)
		}
		if target, ok := view.GotoAt(9, "expr"); ok || target != 0 {
			t.Fatalf("GotoAt(9, expr) = (%d, %v), want (0, false)", target, ok)
		}
		if got, want := view.Terminals(), []string{"$", "ID", "PLUS"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Terminals() = %v, want %v", got, want)
		}
		if got, want := view.NonTerminals(), []string{"expr"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("NonTerminals() = %v, want %v", got, want)
		}
	})
}

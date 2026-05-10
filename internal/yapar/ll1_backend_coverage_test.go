package yapar

import (
	"errors"
	"reflect"
	"testing"

	"genanalex/internal/shared"
)

func TestParserErrorsDefensiveBranches(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil spec error", err: (*SpecError)(nil), want: "<nil>"},
		{name: "spec error without line", err: &SpecError{Message: "boom"}, want: "yapar spec: boom"},
		{name: "nil grammar conflict", err: (*GrammarConflictError)(nil), want: "<nil>"},
		{name: "nil syntax error", err: (*SyntaxError)(nil), want: "<nil>"},
		{name: "syntax error without expected", err: &SyntaxError{Line: 9, GotType: "ID", Lexeme: "x"}, want: "syntax error at line 9: got \"ID\" (\"x\")"},
		{name: "nil ll1 conflict", err: (*LL1ConflictError)(nil), want: "<nil>"},
		{name: "nil left recursion", err: (*LeftRecursionError)(nil), want: "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildLL1TableDefensiveBranches(t *testing.T) {
	t.Run("nil grammar returns empty table", func(t *testing.T) {
		table, err := BuildLL1Table(nil, &FirstFollow{})
		if err != nil {
			t.Fatalf("BuildLL1Table(nil, ff) error = %v", err)
		}
		if table == nil {
			t.Fatal("BuildLL1Table(nil, ff) = nil, want empty table")
		}
		if len(table.M) != 0 {
			t.Fatalf("table.M = %v, want empty map", table.M)
		}
	})

	t.Run("nil first follow is rejected", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token A
%%
s : A ;
`)

		table, err := BuildLL1Table(g, nil)
		if err == nil {
			t.Fatal("BuildLL1Table(g, nil) error = nil, want error")
		}
		if table != nil {
			t.Fatalf("BuildLL1Table(g, nil) table = %#v, want nil", table)
		}
		if got, want := err.Error(), "yapar: first/follow data is required to build LL1 table"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("duplicate entry with same production is ignored", func(t *testing.T) {
		table := &LL1Table{M: map[string]map[string]int{"s": {"A": 7}}}
		if err := table.set("s", "A", 7); err != nil {
			t.Fatalf("set() error = %v, want nil", err)
		}
		if got, want := table.M["s"]["A"], 7; got != want {
			t.Fatalf("table.M[s][A] = %d, want %d", got, want)
		}
	})
}

func TestCheckLeftRecursionAllowsNilAndEmptyBodies(t *testing.T) {
	if err := checkLeftRecursion(nil); err != nil {
		t.Fatalf("checkLeftRecursion(nil) error = %v", err)
	}

	g := &Grammar{
		NonTerminals: map[string]bool{"s": true},
		Productions:  []Production{{ID: 1, Head: "s"}},
	}
	if err := checkLeftRecursion(g); err != nil {
		t.Fatalf("checkLeftRecursion(empty body) error = %v", err)
	}
}

func TestLL1ParserDefensiveBranches(t *testing.T) {
	t.Run("nil parser or grammar is rejected", func(t *testing.T) {
		result, err := (*ll1Parser)(nil).Parse(nil)
		if err == nil {
			t.Fatal("Parse() error = nil, want grammar error")
		}
		if result == nil || result.Accepted {
			t.Fatalf("Parse() result = %#v, want rejected result", result)
		}
		if got, want := err.Error(), "yapar: grammar is required"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("missing table is rejected", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token A
%%
s : A ;
`)

		result, err := (&ll1Parser{grammar: g}).Parse(nil)
		if err == nil {
			t.Fatal("Parse() error = nil, want table error")
		}
		if result == nil || result.Accepted {
			t.Fatalf("Parse() result = %#v, want rejected result", result)
		}
		if got, want := err.Error(), "yapar: ll1 table is required"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("unexpected trailing token expects end marker", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token A
%%
s : ;
`)
		ff, err := ComputeFirstFollow(g)
		if err != nil {
			t.Fatalf("ComputeFirstFollow() error = %v", err)
		}
		parser, err := buildLL1Parser(g, ff)
		if err != nil {
			t.Fatalf("buildLL1Parser() error = %v", err)
		}

		result, err := parser.Parse([]shared.Token{{Type: "A", Lexeme: "a", Line: 4}})
		if err == nil {
			t.Fatal("Parse() error = nil, want syntax error")
		}
		if result == nil || result.Accepted {
			t.Fatalf("Parse() result = %#v, want rejected result", result)
		}

		var syntaxErr *SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("error type = %T, want *SyntaxError", err)
		}
		if got, want := syntaxErr.Expected, []string{EndMarker}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Expected = %v, want %v", got, want)
		}
	})

	t.Run("terminal mismatch reports exact expected token", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token A B
%%
s : A B ;
`)
		ff, err := ComputeFirstFollow(g)
		if err != nil {
			t.Fatalf("ComputeFirstFollow() error = %v", err)
		}
		parser, err := buildLL1Parser(g, ff)
		if err != nil {
			t.Fatalf("buildLL1Parser() error = %v", err)
		}

		result, err := parser.Parse([]shared.Token{{Type: "A", Lexeme: "a", Line: 1}, {Type: "A", Lexeme: "a", Line: 1}})
		if err == nil {
			t.Fatal("Parse() error = nil, want syntax error")
		}
		if result == nil || result.Accepted {
			t.Fatalf("Parse() result = %#v, want rejected result", result)
		}

		var syntaxErr *SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("error type = %T, want *SyntaxError", err)
		}
		if got, want := syntaxErr.Expected, []string{"B"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Expected = %v, want %v", got, want)
		}
	})

	t.Run("unknown production ids are rejected", func(t *testing.T) {
		g := mustBuildGrammar(t, `%token A
%%
s : A ;
`)
		parser := &ll1Parser{
			grammar: g,
			table:   &LL1Table{M: map[string]map[string]int{"s": {"A": 99}}},
		}

		result, err := parser.Parse([]shared.Token{{Type: "A", Lexeme: "a", Line: 2}})
		if err == nil {
			t.Fatal("Parse() error = nil, want unknown production error")
		}
		if result == nil || result.Accepted {
			t.Fatalf("Parse() result = %#v, want rejected result", result)
		}
		if got, want := err.Error(), "yapar: unknown production id 99"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})
}

func TestLL1ParserHelpersAndTableViewDefensiveBranches(t *testing.T) {
	t.Run("lookupProduction handles missing rows", func(t *testing.T) {
		if got, ok := (&ll1Parser{}).lookupProduction("s", "A"); ok || got != 0 {
			t.Fatalf("lookupProduction() = (%d, %v), want (0, false)", got, ok)
		}

		parser := &ll1Parser{table: &LL1Table{M: map[string]map[string]int{"s": {"A": 3}}}}
		if got, ok := parser.lookupProduction("missing", "A"); ok || got != 0 {
			t.Fatalf("lookupProduction(missing) = (%d, %v), want (0, false)", got, ok)
		}
	})

	t.Run("expectedLookaheads are sorted", func(t *testing.T) {
		parser := &ll1Parser{table: &LL1Table{M: map[string]map[string]int{"s": {"B": 2, "A": 1}}}}
		if got, want := parser.expectedLookaheads("s"), []string{"A", "B"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("expectedLookaheads() = %v, want %v", got, want)
		}
		if got := (*ll1Parser)(nil).expectedLookaheads("s"); got != nil {
			t.Fatalf("expectedLookaheads(nil) = %v, want nil", got)
		}
	})

	t.Run("table view exposes nil and missing state branches", func(t *testing.T) {
		if got := (&ll1TableView{}).States(); got != nil {
			t.Fatalf("States() = %v, want nil", got)
		}
		if got := (&ll1TableView{}).Terminals(); got != nil {
			t.Fatalf("Terminals() = %v, want nil", got)
		}
		if got := (&ll1TableView{}).NonTerminals(); got != nil {
			t.Fatalf("NonTerminals() = %v, want nil", got)
		}

		view := &ll1TableView{table: &LL1Table{M: map[string]map[string]int{"s": {"A": 1}}, ntIndex: map[string]int{"s": 0}, terminals: []string{"A"}, nonTerminals: []string{"s"}}}
		if kind, prodID, ok := view.ActionAt(8, "A"); ok || kind != ActionError || prodID != 0 {
			t.Fatalf("ActionAt(invalid state) = (%v, %d, %v), want (ActionError, 0, false)", kind, prodID, ok)
		}
		if kind, prodID, ok := view.ActionAt(0, "B"); ok || kind != ActionError || prodID != 0 {
			t.Fatalf("ActionAt(missing symbol) = (%v, %d, %v), want (ActionError, 0, false)", kind, prodID, ok)
		}
		if got, ok := view.nonTerminalForState(8); ok || got != "" {
			t.Fatalf("nonTerminalForState(8) = (%q, %v), want (\"\", false)", got, ok)
		}
	})
}

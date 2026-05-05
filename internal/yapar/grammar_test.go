package yapar

import (
	"strings"
	"testing"
)

func TestBuildGrammarBuildsAugmentedFormalGrammar(t *testing.T) {
	spec, err := Parse(`%token ID PLUS TIMES LPAREN RPAREN WS
IGNORE WS
%%
expr : expr PLUS term | term ;
term : term TIMES factor | factor ;
factor : ID | LPAREN expr RPAREN ;
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	g, err := BuildGrammar(spec)
	if err != nil {
		t.Fatalf("BuildGrammar() error = %v", err)
	}

	if g.Start != "expr" {
		t.Fatalf("Start = %q, want expr", g.Start)
	}
	if g.Augmented != "expr'" {
		t.Fatalf("Augmented = %q, want expr'", g.Augmented)
	}
	if !g.IsTerminal("ID") || !g.IsTerminal(EndMarker) {
		t.Fatalf("expected terminals to include ID and %q", EndMarker)
	}
	if !g.IsNonTerminal("expr") || !g.IsNonTerminal(g.Augmented) {
		t.Fatalf("expected non-terminals to include expr and augmented start")
	}
	if !g.IgnoreSet["WS"] {
		t.Fatalf("IgnoreSet = %v, want WS ignored", g.IgnoreSet)
	}

	if len(g.Productions) != 7 {
		t.Fatalf("len(Productions) = %d, want 7", len(g.Productions))
	}

	augmented := g.Productions[0]
	if augmented.ID != 0 {
		t.Fatalf("augmented production ID = %d, want 0", augmented.ID)
	}
	if augmented.Head != g.Augmented {
		t.Fatalf("augmented head = %q, want %q", augmented.Head, g.Augmented)
	}
	if len(augmented.Body) != 1 || augmented.Body[0].Name != "expr" || augmented.Body[0].Terminal {
		t.Fatalf("augmented body = %#v, want expr as non-terminal", augmented.Body)
	}

	firstExpr := g.Productions[1]
	if firstExpr.Head != "expr" {
		t.Fatalf("production[1].Head = %q, want expr", firstExpr.Head)
	}
	if len(firstExpr.Body) != 3 {
		t.Fatalf("expr body length = %d, want 3", len(firstExpr.Body))
	}
	if firstExpr.Body[0].Name != "expr" || firstExpr.Body[0].Terminal {
		t.Fatalf("first expr symbol = %#v, want expr non-terminal", firstExpr.Body[0])
	}
	if firstExpr.Body[1].Name != "PLUS" || !firstExpr.Body[1].Terminal {
		t.Fatalf("second expr symbol = %#v, want PLUS terminal", firstExpr.Body[1])
	}
	if firstExpr.Body[2].Name != "term" || firstExpr.Body[2].Terminal {
		t.Fatalf("third expr symbol = %#v, want term non-terminal", firstExpr.Body[2])
	}
}

func TestBuildGrammarRepresentsEmptyAlternativesAsEpsilonProductions(t *testing.T) {
	spec, err := Parse(`%token ID
%%
optional : ID | ;
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	g, err := BuildGrammar(spec)
	if err != nil {
		t.Fatalf("BuildGrammar() error = %v", err)
	}

	if len(g.Productions) != 3 {
		t.Fatalf("len(Productions) = %d, want 3", len(g.Productions))
	}

	epsilonProd := g.Productions[2]
	if !epsilonProd.IsEpsilon() {
		t.Fatalf("production %#v should be epsilon", epsilonProd)
	}
	if len(epsilonProd.Body) != 0 {
		t.Fatalf("epsilon body = %#v, want empty body", epsilonProd.Body)
	}
	if got := FirstOfSequence(epsilonProd.Body, nil); !got.Has(Epsilon) {
		t.Fatalf("FirstOfSequence(empty) = %v, want %q", got.Sorted(), Epsilon)
	}
}

func TestBuildGrammarValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMessage string
	}{
		{
			name: "undeclared symbol in body",
			input: `%token ID
%%
start : expr ;
`,
			wantMessage: `symbol "expr" is neither token nor production head`,
		},
		{
			name: "ignored token must be declared",
			input: `IGNORE WS
%%
start : ;
`,
			wantMessage: `ignored token "WS" must be declared with %token`,
		},
		{
			name: "ignored token cannot appear in production",
			input: `%token ID WS
IGNORE WS
%%
start : ID WS ;
`,
			wantMessage: `ignored token "WS" cannot appear in grammar productions`,
		},
		{
			name: "token and non-terminal name collision",
			input: `%token start
%%
start : ;
`,
			wantMessage: `symbol "start" cannot be both token and non-terminal`,
		},
		{
			name: "explicit epsilon symbol is rejected",
			input: `%token ID
%%
start : ε ;
`,
			wantMessage: `explicit epsilon symbol is not allowed; use an empty alternative instead`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			_, err = BuildGrammar(spec)
			if err == nil {
				t.Fatal("BuildGrammar() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("error = %q, want message containing %q", err.Error(), tt.wantMessage)
			}
		})
	}
}

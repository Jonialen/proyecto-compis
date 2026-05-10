package yapar

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/shared"
)

func TestLL1ErrorsExposeContext(t *testing.T) {
	conflict := &LL1ConflictError{NonTerminal: "opt", Terminal: "B", ExistingProd: 2, ConflictProd: 3}
	if got := conflict.Error(); !strings.Contains(got, "opt") || !strings.Contains(got, "B") || !strings.Contains(got, "2") || !strings.Contains(got, "3") {
		t.Fatalf("LL1ConflictError.Error() = %q, want context-rich message", got)
	}

	leftRecursion := &LeftRecursionError{NonTerminal: "expr", ProductionID: 1}
	if got := leftRecursion.Error(); !strings.Contains(got, "expr") || !strings.Contains(got, "1") {
		t.Fatalf("LeftRecursionError.Error() = %q, want context-rich message", got)
	}
}

func TestBuildLL1TableBuildsPredictiveEntries(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : A opt ;
opt : B | ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	table, err := BuildLL1Table(g, ff)
	if err != nil {
		t.Fatalf("BuildLL1Table() error = %v", err)
	}

	if got, want := table.M["s"]["A"], 1; got != want {
		t.Fatalf("M[s,A] = %d, want %d", got, want)
	}
	if got, want := table.M["opt"]["B"], 2; got != want {
		t.Fatalf("M[opt,B] = %d, want %d", got, want)
	}
	if got, want := table.M["opt"][EndMarker], 3; got != want {
		t.Fatalf("M[opt,$] = %d, want %d", got, want)
	}
	if got, want := table.nonTerminals, []string{"opt", "s"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("nonTerminals = %v, want %v", got, want)
	}
	if got, want := table.terminals, []string{"$", "A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("terminals = %v, want %v", got, want)
	}
}

func TestBuildLL1TableRejectsDirectLeftRecursion(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
expr : expr A | B ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	_, err = BuildLL1Table(g, ff)
	if err == nil {
		t.Fatal("BuildLL1Table() error = nil, want left recursion error")
	}

	var leftRecursion *LeftRecursionError
	if !errors.As(err, &leftRecursion) {
		t.Fatalf("error type = %T, want *LeftRecursionError", err)
	}
	if leftRecursion.NonTerminal != "expr" || leftRecursion.ProductionID != 1 {
		t.Fatalf("left recursion = %#v, want expr production 1", leftRecursion)
	}
}

func TestBuildLL1TableRejectsPredictPredictConflict(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : A | A B ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	_, err = BuildLL1Table(g, ff)
	if err == nil {
		t.Fatal("BuildLL1Table() error = nil, want conflict")
	}

	var conflict *LL1ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error type = %T, want *LL1ConflictError", err)
	}
	if conflict.NonTerminal != "s" || conflict.Terminal != "A" {
		t.Fatalf("conflict = %#v, want non-terminal s and terminal A", conflict)
	}
}

func TestLL1ParserAcceptsAndRejectsInput(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B WS
IGNORE WS
%%
s : A opt ;
opt : B | ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	parser, err := buildLL1Parser(g, ff)
	if err != nil {
		t.Fatalf("buildLL1Parser() error = %v", err)
	}

	accepted, err := parser.Parse([]shared.Token{{Type: "WS", Lexeme: " ", Line: 1}, {Type: "A", Lexeme: "a", Line: 1}, {Type: "B", Lexeme: "b", Line: 1}})
	if err != nil {
		t.Fatalf("Parse(valid) error = %v", err)
	}
	if accepted == nil || !accepted.Accepted {
		t.Fatalf("Parse(valid) = %#v, want Accepted=true", accepted)
	}

	rejected, err := parser.Parse([]shared.Token{{Type: "B", Lexeme: "b", Line: 7}})
	if err == nil {
		t.Fatal("Parse(invalid) error = nil, want syntax error")
	}
	if rejected == nil || rejected.Accepted {
		t.Fatalf("Parse(invalid) = %#v, want Accepted=false", rejected)
	}

	var syntaxErr *SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("error type = %T, want *SyntaxError", err)
	}
	if got, want := syntaxErr.Expected, []string{"A"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("syntaxErr.Expected = %v, want %v", got, want)
	}
}

func TestLL1TableViewExposesPredictiveTable(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : A opt ;
opt : B | ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	table, err := BuildLL1Table(g, ff)
	if err != nil {
		t.Fatalf("BuildLL1Table() error = %v", err)
	}

	view := (&ll1Parser{grammar: g, table: table}).Table()
	if got, want := view.States(), []int{0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("States() = %v, want %v", got, want)
	}
	if got, want := view.Terminals(), []string{"$", "A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Terminals() = %v, want %v", got, want)
	}
	if got, want := view.NonTerminals(), []string{"opt", "s"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NonTerminals() = %v, want %v", got, want)
	}
	if kind, prodID, ok := view.ActionAt(1, "A"); !ok || kind != ActionReduce || prodID != 1 {
		t.Fatalf("ActionAt(1, A) = (%v, %d, %v), want reduce 1", kind, prodID, ok)
	}
	if kind, prodID, ok := view.ActionAt(0, EndMarker); !ok || kind != ActionReduce || prodID != 3 {
		t.Fatalf("ActionAt(0, $) = (%v, %d, %v), want reduce 3", kind, prodID, ok)
	}
	if got, ok := view.GotoAt(1, "opt"); ok || got != 0 {
		t.Fatalf("GotoAt(1, opt) = (%d, %v), want no-op false", got, ok)
	}
}

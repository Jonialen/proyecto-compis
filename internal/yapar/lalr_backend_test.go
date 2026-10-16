package yapar

import (
	"bytes"
	"errors"
	"log"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/shared"
)

func TestMergeLR1StatesMergesByCoreSignatureAndUnionsLookaheads(t *testing.T) {
	states := []LR1State{
		{ID: 0, Items: []LR1Item{{ProductionID: 1, Dot: 0, Lookahead: "$"}, {ProductionID: 2, Dot: 1, Lookahead: "A"}}},
		{ID: 1, Items: []LR1Item{{ProductionID: 1, Dot: 0, Lookahead: "A"}, {ProductionID: 2, Dot: 1, Lookahead: "B"}}},
		{ID: 2, Items: []LR1Item{{ProductionID: 1, Dot: 1, Lookahead: "$"}}},
	}
	transitions := map[int]map[string]int{
		0: {"ID": 2},
		1: {"ID": 2},
	}

	merged, remapped := MergeLR1States(states, transitions)
	if len(merged) != 2 {
		t.Fatalf("len(merged) = %d, want 2", len(merged))
	}

	var mergedCore LALRState
	var targetCore LALRState
	for _, state := range merged {
		switch coreSignature(state.Items) {
		case "1:0|2:1":
			mergedCore = state
		case "1:1":
			targetCore = state
		}
	}

	if got, want := mergedCore.Items, []LR1Item{{ProductionID: 1, Dot: 0, Lookahead: "$"}, {ProductionID: 1, Dot: 0, Lookahead: "A"}, {ProductionID: 2, Dot: 1, Lookahead: "A"}, {ProductionID: 2, Dot: 1, Lookahead: "B"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("merged lookaheads = %#v, want %#v", got, want)
	}
	if got, ok := remapped[mergedCore.ID]["ID"]; !ok || got != targetCore.ID {
		t.Fatalf("remapped transition = (%d, %v), want (%d, true)", got, ok, targetCore.ID)
	}
}

func TestBuildLALRTableResolvesReduceReduceConflictsWithWarning(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : x | y ;
x : A ;
y : B ;
`)

	states := []LALRState{{
		ID: 0,
		Items: []LR1Item{
			{ProductionID: 2, Dot: 1, Lookahead: "$"},
			{ProductionID: 1, Dot: 1, Lookahead: "$"},
		},
	}}

	var logs bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	defer log.SetOutput(originalWriter)
	defer log.SetFlags(originalFlags)

	table, err := BuildLALRTable(g, states, nil)
	if err != nil {
		t.Fatalf("BuildLALRTable() error = %v", err)
	}

	action, ok := table.Action[0]["$"]
	if !ok {
		t.Fatal("missing reduce action after conflict resolution")
	}
	if action.Kind != ActionReduce || action.ProductionID != 1 {
		t.Fatalf("resolved action = %#v, want reduce production 1", action)
	}
	if got := logs.String(); !strings.Contains(got, "reduce/reduce") || !strings.Contains(got, "production 1") {
		t.Fatalf("warning log = %q, want reduce/reduce warning mentioning chosen production", got)
	}
}

func TestBuildParserLALRAcceptsGrammarThatSLRRejects(t *testing.T) {
	g := mustBuildGrammar(t, `%token STAR ID EQUAL
%%
s : L EQUAL R | R ;
L : STAR R | ID ;
R : L ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatal(err)
	}
	slr, err := BuildParser(g, ff, MethodSLR)
	var conflict *GrammarConflictError
	if slr != nil || !errors.As(err, &conflict) {
		t.Fatalf("BuildParser(SLR) = (%v, %v), want nil parser and conflict", slr, err)
	}
	if conflict.Kind != "shift/reduce" || conflict.Symbol != "EQUAL" {
		t.Fatalf("SLR conflict = %#v, want shift/reduce on EQUAL", conflict)
	}
	lalr, err := BuildParser(g, ff, MethodLALR)
	if err != nil {
		t.Fatalf("BuildParser(LALR) error = %v", err)
	}
	for _, tt := range []struct {
		name     string
		input    string
		accepted bool
	}{
		{name: "reference", input: "ID", accepted: true},
		{name: "assignment", input: "ID EQUAL ID", accepted: true},
		{name: "dereferenced assignment", input: "STAR ID EQUAL STAR ID", accepted: true},
		{name: "nested dereference", input: "STAR STAR ID", accepted: true},
		{name: "empty"},
		{name: "missing target", input: "EQUAL ID"},
		{name: "missing value", input: "ID EQUAL"},
		{name: "chained assignment", input: "ID EQUAL ID EQUAL ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var tokens []shared.Token
			for _, kind := range strings.Fields(tt.input) {
				tokens = append(tokens, shared.Token{Type: kind, Lexeme: kind, Line: 1})
			}
			result, err := lalr.Parse(tokens)
			if result == nil || result.Accepted != tt.accepted {
				t.Fatalf("Parse(%q) = (%#v, %v), want Accepted=%v", tt.input, result, err, tt.accepted)
			}
			if tt.accepted && err != nil {
				t.Fatalf("Parse(valid) error = %v", err)
			}
			if !tt.accepted {
				var syntaxErr *SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Fatalf("Parse(invalid) error = %v, want SyntaxError", err)
				}
			}
		})
	}
}

func TestBuildLALRParserParsesAndExposesTableView(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	parser, err := buildLALRParser(g, ff)
	if err != nil {
		t.Fatalf("buildLALRParser() error = %v", err)
	}

	view := parser.Table()
	if view == nil {
		t.Fatal("parser.Table() = nil, want table view")
	}
	if len(view.States()) == 0 {
		t.Fatal("view.States() = empty, want parser states")
	}
	if kind, _, ok := view.ActionAt(0, "ID"); !ok || kind != ActionShift {
		t.Fatalf("ActionAt(0, ID) = (%v, _, %v), want shift action", kind, ok)
	}

	accepted, err := parser.Parse([]shared.Token{{Type: "WS", Lexeme: " ", Line: 1}, {Type: "ID", Lexeme: "x", Line: 1}, {Type: "PLUS", Lexeme: "+", Line: 1}, {Type: "ID", Lexeme: "y", Line: 1}})
	if err != nil {
		t.Fatalf("Parse(valid) error = %v", err)
	}
	if accepted == nil || !accepted.Accepted {
		t.Fatalf("Parse(valid) = %#v, want accepted result", accepted)
	}

	rejected, err := parser.Parse([]shared.Token{{Type: "ID", Lexeme: "x", Line: 1}, {Type: "PLUS", Lexeme: "+", Line: 1}})
	if err == nil {
		t.Fatal("Parse(invalid) error = nil, want syntax error")
	}
	if rejected == nil || rejected.Accepted {
		t.Fatalf("Parse(invalid) = %#v, want rejected result", rejected)
	}
}

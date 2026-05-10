package yapar

import (
	"bytes"
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

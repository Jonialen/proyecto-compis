package yapar

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBuildSLRTableBuildsKnownTable(t *testing.T) {
	g := mustBuildGrammar(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}

	table, err := BuildSLRTable(g, ff, states, transitions)
	if err != nil {
		t.Fatalf("BuildSLRTable() error = %v", err)
	}

	if got, want := table.Action[0]["C"], (Action{Kind: ActionShift, TargetState: transitions[0]["C"]}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[0,C] = %#v, want %#v", got, want)
	}
	if got, want := table.Action[0]["D"], (Action{Kind: ActionShift, TargetState: transitions[0]["D"]}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[0,D] = %#v, want %#v", got, want)
	}
	if got, want := table.Goto[0]["c"], transitions[0]["c"]; got != want {
		t.Fatalf("GOTO[0,c] = %d, want %d", got, want)
	}
	if got, want := table.Goto[0]["s"], transitions[0]["s"]; got != want {
		t.Fatalf("GOTO[0,s] = %d, want %d", got, want)
	}

	reduceState := transitions[0]["D"]
	if got, want := table.Action[reduceState]["C"], (Action{Kind: ActionReduce, ProductionID: 3}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[%d,C] = %#v, want %#v", reduceState, got, want)
	}
	if got, want := table.Action[reduceState]["D"], (Action{Kind: ActionReduce, ProductionID: 3}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[%d,D] = %#v, want %#v", reduceState, got, want)
	}
	if got, want := table.Action[reduceState][EndMarker], (Action{Kind: ActionReduce, ProductionID: 3}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[%d,$] = %#v, want %#v", reduceState, got, want)
	}

	acceptState := transitions[0]["s"]
	if got, want := table.Action[acceptState][EndMarker], (Action{Kind: ActionAccept}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ACTION[%d,$] = %#v, want %#v", acceptState, got, want)
	}

	if got, want := table.ExpectedTokens(0), []string{"C", "D"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpectedTokens(0) = %v, want %v", got, want)
	}
	if got, want := table.ExpectedTokens(reduceState), []string{"$", "C", "D"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpectedTokens(%d) = %v, want %v", reduceState, got, want)
	}
	if _, exists := table.Goto[0]["C"]; exists {
		t.Fatal("GOTO should not contain terminal transitions")
	}
	if _, exists := table.Action[0]["c"]; exists {
		t.Fatal("ACTION should not contain non-terminal transitions")
	}
}

func TestBuildSLRTableDetectsShiftReduceConflict(t *testing.T) {
	g := mustBuildGrammar(t, `%token STAR ID EQUAL
%%
s : L EQUAL R | R ;
L : STAR R | ID ;
R : L ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}

	_, err = BuildSLRTable(g, ff, states, transitions)
	if err == nil {
		t.Fatal("BuildSLRTable() error = nil, want shift/reduce conflict")
	}

	var conflict *GrammarConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error type = %T, want *GrammarConflictError", err)
	}
	if conflict.Kind != "shift/reduce" {
		t.Fatalf("conflict.Kind = %q, want shift/reduce", conflict.Kind)
	}
	if conflict.Symbol != "EQUAL" {
		t.Fatalf("conflict.Symbol = %q, want EQUAL", conflict.Symbol)
	}
	if !strings.Contains(conflict.Error(), "existing=shift") || !strings.Contains(conflict.Error(), "new=reduce") {
		t.Fatalf("conflict error = %q, want descriptive actions", conflict.Error())
	}
	if describeProduction(g, conflict.New.ProductionID) != "R -> L" {
		t.Fatalf("conflict reduce production = %q, want R -> L", describeProduction(g, conflict.New.ProductionID))
	}
	if !stateHasItem(states[conflict.State], g, "s", []string{"L", "EQUAL", "R"}, 1) {
		t.Fatalf("state %d does not contain expected shift item", conflict.State)
	}
	if !stateHasItem(states[conflict.State], g, "R", []string{"L"}, 1) {
		t.Fatalf("state %d does not contain expected reduce item", conflict.State)
	}
	if transitions[conflict.State]["EQUAL"] != conflict.Current.TargetState {
		t.Fatalf("shift target = %d, want %d from transitions", conflict.Current.TargetState, transitions[conflict.State]["EQUAL"])
	}
}

func TestBuildSLRTableDetectsReduceReduceConflict(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID
%%
s : a | b ;
a : ID ;
b : ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}

	_, err = BuildSLRTable(g, ff, states, transitions)
	if err == nil {
		t.Fatal("BuildSLRTable() error = nil, want reduce/reduce conflict")
	}

	var conflict *GrammarConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error type = %T, want *GrammarConflictError", err)
	}
	if conflict.Kind != "reduce/reduce" {
		t.Fatalf("conflict.Kind = %q, want reduce/reduce", conflict.Kind)
	}
	if conflict.Symbol != EndMarker {
		t.Fatalf("conflict.Symbol = %q, want %q", conflict.Symbol, EndMarker)
	}
	if conflict.Current.Kind != ActionReduce || conflict.New.Kind != ActionReduce {
		t.Fatalf("conflict actions = %#v / %#v, want both reduce", conflict.Current, conflict.New)
	}
	if describeProduction(g, conflict.Current.ProductionID) != "a -> ID" {
		t.Fatalf("current production = %q, want a -> ID", describeProduction(g, conflict.Current.ProductionID))
	}
	if describeProduction(g, conflict.New.ProductionID) != "b -> ID" {
		t.Fatalf("new production = %q, want b -> ID", describeProduction(g, conflict.New.ProductionID))
	}
	if !strings.Contains(conflict.Error(), "reduce") {
		t.Fatalf("conflict error = %q, want reduce details", conflict.Error())
	}
}

func stateHasItem(state State, g *Grammar, head string, body []string, dot int) bool {
	for _, item := range state.Items {
		production, ok := productionByID(g, item.ProductionID)
		if !ok || production.Head != head || item.Dot != dot || len(production.Body) != len(body) {
			continue
		}
		match := true
		for i, symbol := range production.Body {
			if symbol.Name != body[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

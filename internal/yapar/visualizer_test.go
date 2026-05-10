package yapar

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type visualizerStubTable struct {
	actions      map[int]map[string]Action
	gotos        map[int]map[string]int
	states       []int
	terminals    []string
	nonTerminals []string
}

func (s visualizerStubTable) ActionAt(state int, symbol string) (ActionKind, int, bool) {
	if row := s.actions[state]; row != nil {
		if action, ok := row[symbol]; ok {
			return action.Kind, actionValue(action), true
		}
	}
	return ActionError, 0, false
}

func (s visualizerStubTable) GotoAt(state int, symbol string) (int, bool) {
	if row := s.gotos[state]; row != nil {
		target, ok := row[symbol]
		return target, ok
	}
	return 0, false
}

func (s visualizerStubTable) States() []int          { return append([]int(nil), s.states...) }
func (s visualizerStubTable) Terminals() []string    { return append([]string(nil), s.terminals...) }
func (s visualizerStubTable) NonTerminals() []string { return append([]string(nil), s.nonTerminals...) }

func TestProductionLabel(t *testing.T) {
	g := mustBuildGrammar(t, `%token A
%%
s : A | ;
`)

	if got, want := ProductionLabel(g, 1), "s → A"; got != want {
		t.Fatalf("ProductionLabel(1) = %q, want %q", got, want)
	}
	if got, want := ProductionLabel(g, 2), "s → ε"; got != want {
		t.Fatalf("ProductionLabel(2) = %q, want %q", got, want)
	}
	if got, want := ProductionLabel(g, 99), "production #99"; got != want {
		t.Fatalf("ProductionLabel(99) = %q, want %q", got, want)
	}
}

func TestBuildVisReport(t *testing.T) {
	t.Run("rejects nil grammar", func(t *testing.T) {
		report, err := BuildVisReport(nil, nil, MethodSLR)
		if err == nil {
			t.Fatal("BuildVisReport() error = nil, want error")
		}
		if report != nil {
			t.Fatalf("BuildVisReport() report = %#v, want nil", report)
		}
	})

	t.Run("routes supported methods", func(t *testing.T) {
		ll1Grammar := mustBuildGrammar(t, `%token A B
%%
s : A opt ;
opt : B | ;
`)
		ll1FF, err := ComputeFirstFollow(ll1Grammar)
		if err != nil {
			t.Fatalf("ComputeFirstFollow() error = %v", err)
		}
		ll1Report, err := BuildVisReport(ll1Grammar, ll1FF, MethodLL1)
		if err != nil {
			t.Fatalf("BuildVisReport(ll1) error = %v", err)
		}
		if ll1Report.Method != MethodLL1 {
			t.Fatalf("ll1 report method = %q, want %q", ll1Report.Method, MethodLL1)
		}
		if ll1Report.Table == nil {
			t.Fatal("ll1 report table = nil, want populated table")
		}
		if ll1Report.LR0States != nil {
			t.Fatalf("ll1 report LR0States = %#v, want nil", ll1Report.LR0States)
		}
		if ll1Report.LR0Trans != nil {
			t.Fatalf("ll1 report LR0Trans = %#v, want nil", ll1Report.LR0Trans)
		}

		lrGrammar := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
		lrFF, err := ComputeFirstFollow(lrGrammar)
		if err != nil {
			t.Fatalf("ComputeFirstFollow() error = %v", err)
		}

		for _, method := range []Method{MethodSLR, MethodLALR} {
			report, err := BuildVisReport(lrGrammar, lrFF, method)
			if err != nil {
				t.Fatalf("BuildVisReport(%s) error = %v", method, err)
			}
			if report.Method != method {
				t.Fatalf("report method = %q, want %q", report.Method, method)
			}
			if report.Table == nil {
				t.Fatalf("report table for %s = nil, want populated table", method)
			}
			if len(report.Table.States()) == 0 {
				t.Fatalf("report table states for %s = 0, want populated table", method)
			}
			if len(report.LR0States) == 0 {
				t.Fatalf("report LR0States for %s = 0, want populated automaton", method)
			}
			if len(report.LR0Trans) == 0 {
				t.Fatalf("report LR0Trans for %s = 0, want populated transitions", method)
			}
		}
	})
}

func TestRenderTableText(t *testing.T) {
	report := &VisualizationReport{
		Method: MethodSLR,
		Table: visualizerStubTable{
			actions: map[int]map[string]Action{
				0: {
					"A": {Kind: ActionShift, TargetState: 2},
					"B": {Kind: ActionReduce, ProductionID: 4},
					"$": {Kind: ActionAccept},
				},
			},
			gotos:        map[int]map[string]int{0: {"expr": 7}},
			states:       []int{0},
			terminals:    []string{"A", "B", "$"},
			nonTerminals: []string{"expr"},
		},
	}

	if got, want := RenderTableText(report), "State\tACTION\tGOTO\n0\tA=s2, B=r4, $=acc\texpr=7\n"; got != want {
		t.Fatalf("RenderTableText() = %q, want %q", got, want)
	}
}

func TestRenderTableTextNilSafety(t *testing.T) {
	tests := []struct {
		name   string
		report *VisualizationReport
		want   string
	}{
		{
			name:   "nil report returns empty table marker",
			report: nil,
			want:   "<empty table>\n",
		},
		{
			name:   "nil table returns empty table marker",
			report: &VisualizationReport{Method: MethodSLR},
			want:   "<empty table>\n",
		},
		{
			name: "empty action and goto rows render placeholders",
			report: &VisualizationReport{Table: visualizerStubTable{
				states:       []int{3},
				terminals:    []string{"ID"},
				nonTerminals: []string{"expr"},
			}},
			want: "State\tACTION\tGOTO\n3\t-\t-\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderTableText(tt.report); got != tt.want {
				t.Fatalf("RenderTableText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTableJSON(t *testing.T) {
	report := &VisualizationReport{
		Method: MethodSLR,
		Table: visualizerStubTable{
			actions: map[int]map[string]Action{
				0: {
					"A": {Kind: ActionShift, TargetState: 2},
					"$": {Kind: ActionAccept},
				},
			},
			gotos:        map[int]map[string]int{0: {"expr": 7}},
			states:       []int{0},
			terminals:    []string{"A", "B", "$"},
			nonTerminals: []string{"expr"},
		},
	}

	raw, err := RenderTableJSON(report)
	if err != nil {
		t.Fatalf("RenderTableJSON() error = %v", err)
	}

	var payload struct {
		Method       string   `json:"method"`
		Terminals    []string `json:"terminals"`
		NonTerminals []string `json:"non_terminals"`
		States       []struct {
			ID      int               `json:"id"`
			Actions map[string]string `json:"actions"`
			Gotos   map[string]int    `json:"gotos"`
		} `json:"states"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\nraw=%s", err, string(raw))
	}
	if payload.Method != string(MethodSLR) {
		t.Fatalf("method = %q, want %q", payload.Method, MethodSLR)
	}
	if len(payload.States) != 1 {
		t.Fatalf("states len = %d, want 1", len(payload.States))
	}
	if got, want := payload.States[0].Actions["A"], "s2"; got != want {
		t.Fatalf("actions[A] = %q, want %q", got, want)
	}
	if got, want := payload.States[0].Actions["B"], ""; got != want {
		t.Fatalf("actions[B] = %q, want empty string", got)
	}
	if got, want := payload.States[0].Actions["$"], "acc"; got != want {
		t.Fatalf("actions[$] = %q, want %q", got, want)
	}
	if got, want := payload.States[0].Gotos["expr"], 7; got != want {
		t.Fatalf("gotos[expr] = %d, want %d", got, want)
	}
}

func TestRenderTableJSONNilSafety(t *testing.T) {
	tests := []struct {
		name   string
		report *VisualizationReport
	}{
		{name: "nil report", report: nil},
		{name: "nil table", report: &VisualizationReport{Method: MethodLALR}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := RenderTableJSON(tt.report)
			if err != nil {
				t.Fatalf("RenderTableJSON() error = %v", err)
			}

			var payload struct {
				Method       string            `json:"method"`
				Terminals    []string          `json:"terminals"`
				NonTerminals []string          `json:"non_terminals"`
				States       []json.RawMessage `json:"states"`
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v\nraw=%s", err, string(raw))
			}
			if payload.Method != "" {
				t.Fatalf("method = %q, want empty method for nil-safe payload", payload.Method)
			}
			if len(payload.Terminals) != 0 {
				t.Fatalf("terminals len = %d, want 0", len(payload.Terminals))
			}
			if len(payload.NonTerminals) != 0 {
				t.Fatalf("non_terminals len = %d, want 0", len(payload.NonTerminals))
			}
			if len(payload.States) != 0 {
				t.Fatalf("states len = %d, want 0", len(payload.States))
			}
		})
	}
}

func TestRenderAutomatonDOT(t *testing.T) {
	report := &VisualizationReport{
		Method: MethodSLR,
		Grammar: mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`),
		LR0States: []State{{
			ID:    0,
			Items: []Item{{ProductionID: 0, Dot: 0}},
		}},
		LR0Trans: map[int]map[string]int{0: {"expr": 1}},
	}

	dot, err := RenderAutomatonDOT(report)
	if err != nil {
		t.Fatalf("RenderAutomatonDOT() error = %v", err)
	}
	if !strings.Contains(dot, "digraph LR0") {
		t.Fatalf("DOT = %q, want digraph header", dot)
	}
	if !strings.Contains(dot, "I0") {
		t.Fatalf("DOT = %q, want state label", dot)
	}
	if !strings.Contains(dot, "expr' → • expr") {
		t.Fatalf("DOT = %q, want production label", dot)
	}
	if !strings.Contains(dot, "I0 -> I1 [label=\"expr\"]") {
		t.Fatalf("DOT = %q, want transition", dot)
	}

	ll1Report := &VisualizationReport{Method: MethodLL1}
	_, err = RenderAutomatonDOT(ll1Report)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RenderAutomatonDOT(ll1) error = %v, want ErrUnsupported", err)
	}
}

func TestRenderAutomatonDOTNilSafety(t *testing.T) {
	t.Run("nil report is unsupported", func(t *testing.T) {
		_, err := RenderAutomatonDOT(nil)
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("RenderAutomatonDOT(nil) error = %v, want ErrUnsupported", err)
		}
	})

	t.Run("nil grammar with lr0 states is rejected", func(t *testing.T) {
		_, err := RenderAutomatonDOT(&VisualizationReport{
			Method:    MethodSLR,
			LR0States: []State{{ID: 0}},
		})
		if err == nil {
			t.Fatal("RenderAutomatonDOT() error = nil, want grammar error")
		}
		if got, want := err.Error(), "yapar: grammar is required to render LR(0) automaton"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("nil transition rows still render node-only graph", func(t *testing.T) {
		report := &VisualizationReport{
			Method: MethodSLR,
			Grammar: mustBuildGrammar(t, `%token ID
%%
expr : ID ;
`),
			LR0States: []State{{
				ID:    4,
				Items: []Item{{ProductionID: 0, Dot: 0}},
			}},
		}

		dot, err := RenderAutomatonDOT(report)
		if err != nil {
			t.Fatalf("RenderAutomatonDOT() error = %v", err)
		}
		if !strings.Contains(dot, "I4") {
			t.Fatalf("DOT = %q, want rendered node", dot)
		}
		if strings.Contains(dot, "->") {
			t.Fatalf("DOT = %q, want no transitions for nil LR0Trans", dot)
		}
	})
}

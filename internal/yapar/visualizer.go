package yapar

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var ErrUnsupported = errors.New("yapar: operation not supported for this parser method")

type VisualizationReport struct {
	Method    Method
	Grammar   *Grammar
	Table     TableView
	LR0States []State
	LR0Trans  map[int]map[string]int
}

func ProductionLabel(g *Grammar, id int) string {
	production, ok := productionByID(g, id)
	if !ok {
		return "production #" + strconv.Itoa(id)
	}
	if len(production.Body) == 0 {
		return production.Head + " → " + Epsilon
	}
	parts := make([]string, len(production.Body))
	for i, symbol := range production.Body {
		parts[i] = symbol.Name
	}
	return production.Head + " → " + strings.Join(parts, " ")
}

func BuildVisReport(g *Grammar, ff *FirstFollow, method Method) (*VisualizationReport, error) {
	if g == nil {
		return nil, fmt.Errorf("yapar: grammar is required to build visualization report")
	}

	report := &VisualizationReport{Method: method, Grammar: g}

	switch method {
	case MethodLL1:
		table, err := BuildLL1Table(g, ff)
		if err != nil {
			return nil, err
		}
		report.Table = &ll1TableView{grammar: g, table: table}
		return report, nil
	case MethodSLR:
		states, transitions, err := BuildCanonicalCollection(g)
		if err != nil {
			return nil, err
		}
		table, err := BuildSLRTable(g, ff, states, transitions)
		if err != nil {
			return nil, err
		}
		report.Table = &slrTableView{grammar: g, table: table}
		report.LR0States = states
		report.LR0Trans = cloneTransitions(transitions)
		return report, nil
	case MethodLALR:
		lr1States, lr1Transitions, err := BuildLR1Collection(g, ff)
		if err != nil {
			return nil, err
		}
		mergedStates, mergedTransitions := MergeLR1States(lr1States, lr1Transitions)
		table, err := BuildLALRTable(g, mergedStates, mergedTransitions)
		if err != nil {
			return nil, err
		}
		lr0States, lr0Transitions, err := BuildCanonicalCollection(g)
		if err != nil {
			return nil, err
		}
		report.Table = &lalrTableView{grammar: g, table: table}
		report.LR0States = lr0States
		report.LR0Trans = cloneTransitions(lr0Transitions)
		return report, nil
	default:
		return nil, fmt.Errorf("parser method %q not implemented yet: valid options are %s", method, strings.Join(methodNames(), ", "))
	}
}

func RenderTableText(report *VisualizationReport) string {
	if report == nil || report.Table == nil {
		return "<empty table>\n"
	}

	terms := report.Table.Terminals()
	nonTerms := report.Table.NonTerminals()
	states := report.Table.States()

	var builder strings.Builder
	builder.WriteString("State\tACTION\tGOTO\n")
	for _, state := range states {
		builder.WriteString(strconv.Itoa(state))
		builder.WriteByte('\t')
		builder.WriteString(renderActionRow(report.Table, state, terms))
		builder.WriteByte('\t')
		builder.WriteString(renderGotoRow(report.Table, state, nonTerms))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func RenderTableJSON(report *VisualizationReport) ([]byte, error) {
	if report == nil || report.Table == nil {
		return json.Marshal(struct {
			Method       string        `json:"method"`
			Terminals    []string      `json:"terminals"`
			NonTerminals []string      `json:"non_terminals"`
			States       []interface{} `json:"states"`
		}{States: []interface{}{}})
	}

	type jsonState struct {
		ID      int               `json:"id"`
		Actions map[string]string `json:"actions"`
		Gotos   map[string]int    `json:"gotos"`
	}
	type payload struct {
		Method       string      `json:"method"`
		Terminals    []string    `json:"terminals"`
		NonTerminals []string    `json:"non_terminals"`
		States       []jsonState `json:"states"`
	}

	terms := report.Table.Terminals()
	nonTerms := report.Table.NonTerminals()
	states := report.Table.States()
	out := payload{
		Method:       string(report.Method),
		Terminals:    append([]string(nil), terms...),
		NonTerminals: append([]string(nil), nonTerms...),
		States:       make([]jsonState, 0, len(states)),
	}
	for _, state := range states {
		row := jsonState{
			ID:      state,
			Actions: make(map[string]string, len(terms)),
			Gotos:   make(map[string]int),
		}
		for _, terminal := range terms {
			row.Actions[terminal] = renderActionCell(report.Table, state, terminal)
		}
		for _, nonTerminal := range nonTerms {
			if target, ok := report.Table.GotoAt(state, nonTerminal); ok {
				row.Gotos[nonTerminal] = target
			}
		}
		out.States = append(out.States, row)
	}

	return json.Marshal(out)
}

func RenderAutomatonDOT(report *VisualizationReport) (string, error) {
	if report == nil || report.LR0States == nil {
		return "", ErrUnsupported
	}
	if report.Grammar == nil {
		return "", fmt.Errorf("yapar: grammar is required to render LR(0) automaton")
	}

	var builder strings.Builder
	builder.WriteString("digraph LR0 {\n")
	builder.WriteString("  rankdir=LR;\n")
	builder.WriteString("  node [shape=box];\n")
	for _, state := range report.LR0States {
		builder.WriteString("  I")
		builder.WriteString(strconv.Itoa(state.ID))
		builder.WriteString(" [label=")
		builder.WriteString(strconv.Quote(dotStateLabel(report.Grammar, state)))
		builder.WriteString("];\n")
	}
	for _, from := range sortedTransitionStates(report.LR0Trans) {
		for _, symbol := range sortedTransitionSymbols(report.LR0Trans[from]) {
			builder.WriteString("  I")
			builder.WriteString(strconv.Itoa(from))
			builder.WriteString(" -> I")
			builder.WriteString(strconv.Itoa(report.LR0Trans[from][symbol]))
			builder.WriteString(" [label=")
			builder.WriteString(strconv.Quote(symbol))
			builder.WriteString("];\n")
		}
	}
	builder.WriteString("}\n")
	return builder.String(), nil
}

func renderActionRow(table TableView, state int, terminals []string) string {
	parts := make([]string, 0, len(terminals))
	for _, symbol := range terminals {
		action := renderActionCell(table, state, symbol)
		if action == "" {
			continue
		}
		parts = append(parts, symbol+"="+action)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func renderGotoRow(table TableView, state int, nonTerminals []string) string {
	parts := make([]string, 0, len(nonTerminals))
	for _, symbol := range nonTerminals {
		if target, ok := table.GotoAt(state, symbol); ok {
			parts = append(parts, fmt.Sprintf("%s=%d", symbol, target))
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func renderActionCell(table TableView, state int, symbol string) string {
	kind, value, ok := table.ActionAt(state, symbol)
	if !ok {
		return ""
	}
	switch kind {
	case ActionShift:
		return fmt.Sprintf("s%d", value)
	case ActionReduce:
		return fmt.Sprintf("r%d", value)
	case ActionAccept:
		return "acc"
	default:
		return ""
	}
}

func dotStateLabel(g *Grammar, state State) string {
	lines := []string{fmt.Sprintf("I%d", state.ID)}
	for _, item := range state.Items {
		lines = append(lines, dotItemLabel(g, item))
	}
	return strings.Join(lines, "\n")
}

func dotItemLabel(g *Grammar, item Item) string {
	production, ok := productionByID(g, item.ProductionID)
	if !ok {
		return "production #" + strconv.Itoa(item.ProductionID)
	}
	parts := make([]string, 0, len(production.Body)+1)
	for index := 0; index <= len(production.Body); index++ {
		if index == item.Dot {
			parts = append(parts, "•")
		}
		if index < len(production.Body) {
			parts = append(parts, production.Body[index].Name)
		}
	}
	if len(production.Body) == 0 && item.Dot == 0 {
		parts = append(parts, Epsilon)
	}
	return production.Head + " → " + strings.Join(parts, " ")
}

func sortedTransitionStates(transitions map[int]map[string]int) []int {
	states := make([]int, 0, len(transitions))
	for state := range transitions {
		states = append(states, state)
	}
	sort.Ints(states)
	return states
}

func sortedTransitionSymbols(row map[string]int) []string {
	symbols := make([]string, 0, len(row))
	for symbol := range row {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	return symbols
}

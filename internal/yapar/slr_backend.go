package yapar

import "genanalex/internal/shared"

type slrParser struct {
	grammar *Grammar
	table   *ParsingTable
}

func buildSLRParser(g *Grammar, ff *FirstFollow) (ExecutableParser, error) {
	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		return nil, err
	}
	table, err := BuildSLRTable(g, ff, states, transitions)
	if err != nil {
		return nil, err
	}
	return &slrParser{grammar: g, table: table}, nil
}

func (p *slrParser) Parse(tokens []shared.Token) (*ParseResult, error) {
	return ParseTokens(p.grammar, p.table, tokens)
}

func (p *slrParser) Table() TableView {
	return &slrTableView{grammar: p.grammar, table: p.table}
}

type slrTableView struct {
	grammar *Grammar
	table   *ParsingTable
}

func (v *slrTableView) ActionAt(state int, symbol string) (ActionKind, int, bool) {
	if v == nil || v.table == nil || v.table.Action == nil {
		return ActionError, 0, false
	}
	row := v.table.Action[state]
	if row == nil {
		return ActionError, 0, false
	}
	action, ok := row[symbol]
	if !ok {
		return ActionError, 0, false
	}
	return action.Kind, actionValue(action), true
}

func (v *slrTableView) GotoAt(state int, symbol string) (int, bool) {
	if v == nil || v.table == nil || v.table.Goto == nil {
		return 0, false
	}
	row := v.table.Goto[state]
	if row == nil {
		return 0, false
	}
	target, ok := row[symbol]
	return target, ok
}

func (v *slrTableView) States() []int {
	if v == nil || v.table == nil {
		return nil
	}
	return sortedStateIDsFromMaps(v.table.Action, v.table.Goto)
}

func (v *slrTableView) Terminals() []string {
	return sortedGrammarTerminals(v.grammar)
}

func (v *slrTableView) NonTerminals() []string {
	return sortedGrammarNonTerminals(v.grammar)
}

func actionValue(action Action) int {
	switch action.Kind {
	case ActionShift:
		return action.TargetState
	case ActionReduce:
		return action.ProductionID
	default:
		return 0
	}
}

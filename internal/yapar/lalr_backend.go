package yapar

import (
	"log"

	"genanalex/internal/shared"
)

type LALRState struct {
	ID    int
	Items []LR1Item
}

type lalrParser struct {
	grammar *Grammar
	table   *ParsingTable
}

func buildLALRParser(g *Grammar, ff *FirstFollow) (ExecutableParser, error) {
	states, transitions, err := BuildLR1Collection(g, ff)
	if err != nil {
		return nil, err
	}
	mergedStates, mergedTransitions := MergeLR1States(states, transitions)
	table, err := BuildLALRTable(g, mergedStates, mergedTransitions)
	if err != nil {
		return nil, err
	}
	return &lalrParser{grammar: g, table: table}, nil
}

func (p *lalrParser) Parse(tokens []shared.Token) (*ParseResult, error) {
	return ParseTokens(p.grammar, p.table, tokens)
}

func (p *lalrParser) Table() TableView {
	return &lalrTableView{grammar: p.grammar, table: p.table}
}

type lalrTableView struct {
	grammar *Grammar
	table   *ParsingTable
}

func (v *lalrTableView) ActionAt(state int, symbol string) (ActionKind, int, bool) {
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

func (v *lalrTableView) GotoAt(state int, symbol string) (int, bool) {
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

func (v *lalrTableView) States() []int {
	if v == nil || v.table == nil {
		return nil
	}
	return sortedStateIDsFromMaps(v.table.Action, v.table.Goto)
}

func (v *lalrTableView) Terminals() []string {
	return sortedGrammarTerminals(v.grammar)
}

func (v *lalrTableView) NonTerminals() []string {
	return sortedGrammarNonTerminals(v.grammar)
}

func MergeLR1States(states []LR1State, transitions map[int]map[string]int) ([]LALRState, map[int]map[string]int) {
	if len(states) == 0 {
		return nil, map[int]map[string]int{}
	}

	groups := make(map[string]int, len(states))
	merged := make([]LALRState, 0, len(states))
	originalToMerged := make(map[int]int, len(states))

	for _, state := range states {
		signature := coreSignature(state.Items)
		mergedID, exists := groups[signature]
		if !exists {
			mergedID = len(merged)
			groups[signature] = mergedID
			merged = append(merged, LALRState{ID: mergedID})
		}
		originalToMerged[state.ID] = mergedID
		merged[mergedID].Items = mergeLR1ItemSets(merged[mergedID].Items, state.Items)
	}

	remapped := make(map[int]map[string]int)
	for fromState, row := range transitions {
		mergedFrom := originalToMerged[fromState]
		if remapped[mergedFrom] == nil {
			remapped[mergedFrom] = make(map[string]int)
		}
		for symbol, toState := range row {
			remapped[mergedFrom][symbol] = originalToMerged[toState]
		}
	}

	return merged, remapped
}

func BuildLALRTable(g *Grammar, states []LALRState, transitions map[int]map[string]int) (*ParsingTable, error) {
	table := &ParsingTable{
		Action: make(map[int]map[string]Action),
		Goto:   make(map[int]map[string]int),
	}
	if g == nil || len(states) == 0 {
		return table, nil
	}

	for stateID, row := range transitions {
		for symbol, target := range row {
			switch {
			case g.IsTerminal(symbol):
				if err := setLALRAction(table, g, stateID, symbol, Action{Kind: ActionShift, TargetState: target}); err != nil {
					return nil, err
				}
			case g.IsNonTerminal(symbol):
				if table.Goto[stateID] == nil {
					table.Goto[stateID] = make(map[string]int)
				}
				table.Goto[stateID][symbol] = target
			}
		}
	}

	for _, state := range states {
		for _, item := range state.Items {
			production, ok := productionByID(g, item.ProductionID)
			if !ok || item.Dot != len(production.Body) {
				continue
			}

			if production.ID == 0 && item.Lookahead == EndMarker {
				if err := setLALRAction(table, g, state.ID, EndMarker, Action{Kind: ActionAccept}); err != nil {
					return nil, err
				}
				continue
			}

			if err := setLALRAction(table, g, state.ID, item.Lookahead, Action{Kind: ActionReduce, ProductionID: production.ID}); err != nil {
				return nil, err
			}
		}
	}

	return table, nil
}

func mergeLR1ItemSets(left, right []LR1Item) []LR1Item {
	merged := append([]LR1Item(nil), left...)
	seen := make(map[string]bool, len(left)+len(right))
	for _, item := range merged {
		seen[lr1ItemKey(item)] = true
	}
	for _, item := range right {
		key := lr1ItemKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, item)
	}
	sortLR1Items(merged)
	return merged
}

func setLALRAction(table *ParsingTable, g *Grammar, state int, symbol string, next Action) error {
	err := table.setAction(state, symbol, next)
	if err == nil {
		return nil
	}

	conflict, ok := err.(*GrammarConflictError)
	if !ok || conflict.Kind != "reduce/reduce" || conflict.Current.Kind != ActionReduce || conflict.New.Kind != ActionReduce {
		return err
	}

	chosen := conflict.Current
	rejected := conflict.New
	if conflict.New.ProductionID < conflict.Current.ProductionID {
		chosen = conflict.New
		rejected = conflict.Current
	}
	table.Action[state][symbol] = chosen
	log.Printf(
		"warning: resolved reduce/reduce conflict at state %d, symbol %q; keeping production %d (%s) over production %d (%s)",
		state,
		symbol,
		chosen.ProductionID,
		describeProduction(g, chosen.ProductionID),
		rejected.ProductionID,
		describeProduction(g, rejected.ProductionID),
	)
	return nil
}

var _ TableView = (*lalrTableView)(nil)
var _ ExecutableParser = (*lalrParser)(nil)

// Package yapar contiene la representación base de la tabla ACTION/GOTO.
package yapar

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ActionKind enumera las acciones posibles de una tabla LR.
type ActionKind int

const (
	ActionError ActionKind = iota
	ActionShift
	ActionReduce
	ActionAccept
)

// Action representa una celda de la tabla ACTION.
type Action struct {
	Kind         ActionKind
	TargetState  int
	ProductionID int
}

// ParsingTable agrupa ACTION y GOTO para el simulador LR.
type ParsingTable struct {
	Action map[int]map[string]Action
	Goto   map[int]map[string]int
}

// BuildSLRTable construye la tabla SLR(1) a partir del autómata LR(0) y FOLLOW.
func BuildSLRTable(g *Grammar, ff *FirstFollow, states []State, transitions map[int]map[string]int) (*ParsingTable, error) {
	table := &ParsingTable{
		Action: make(map[int]map[string]Action),
		Goto:   make(map[int]map[string]int),
	}
	if g == nil || len(states) == 0 {
		return table, nil
	}
	if ff == nil {
		return nil, fmt.Errorf("yapar: first/follow data is required to build SLR table")
	}

	for stateID, row := range transitions {
		for symbol, target := range row {
			switch {
			case g.IsTerminal(symbol):
				if err := table.setAction(stateID, symbol, Action{Kind: ActionShift, TargetState: target}); err != nil {
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

			if production.ID == 0 {
				if err := table.setAction(state.ID, EndMarker, Action{Kind: ActionAccept}); err != nil {
					return nil, err
				}
				continue
			}

			for _, lookahead := range ff.Follow[production.Head].Sorted() {
				if err := table.setAction(state.ID, lookahead, Action{Kind: ActionReduce, ProductionID: production.ID}); err != nil {
					return nil, err
				}
			}
		}
	}

	return table, nil
}

// ExpectedTokens lista terminales definidos para una fila ACTION concreta.
func (t *ParsingTable) ExpectedTokens(state int) []string {
	if t == nil || t.Action == nil {
		return nil
	}
	row := t.Action[state]
	tokens := make([]string, 0, len(row))
	for symbol, action := range row {
		if action.Kind != ActionError {
			tokens = append(tokens, symbol)
		}
	}
	sort.Strings(tokens)
	return tokens
}

func cloneTransitions(src map[int]map[string]int) map[int]map[string]int {
	if len(src) == 0 {
		return map[int]map[string]int{}
	}
	dst := make(map[int]map[string]int, len(src))
	for state, row := range src {
		rowCopy := make(map[string]int, len(row))
		for symbol, target := range row {
			rowCopy[symbol] = target
		}
		dst[state] = rowCopy
	}
	return dst
}

func (t *ParsingTable) setAction(state int, symbol string, next Action) error {
	if t.Action[state] == nil {
		t.Action[state] = make(map[string]Action)
	}
	current, exists := t.Action[state][symbol]
	if !exists {
		t.Action[state][symbol] = next
		return nil
	}
	if actionsEqual(current, next) {
		return nil
	}
	return &GrammarConflictError{
		State:   state,
		Symbol:  symbol,
		Kind:    conflictKind(current, next),
		Current: current,
		New:     next,
	}
}

func actionsEqual(left, right Action) bool {
	return left.Kind == right.Kind && left.TargetState == right.TargetState && left.ProductionID == right.ProductionID
}

func conflictKind(current, next Action) string {
	if current.Kind == ActionReduce && next.Kind == ActionReduce {
		return "reduce/reduce"
	}
	if (current.Kind == ActionShift && next.Kind == ActionReduce) || (current.Kind == ActionReduce && next.Kind == ActionShift) {
		return "shift/reduce"
	}
	if (current.Kind == ActionAccept && next.Kind == ActionReduce) || (current.Kind == ActionReduce && next.Kind == ActionAccept) {
		return "accept/reduce"
	}
	if (current.Kind == ActionAccept && next.Kind == ActionShift) || (current.Kind == ActionShift && next.Kind == ActionAccept) {
		return "accept/shift"
	}
	return "action/action"
}

func formatAction(action Action) string {
	switch action.Kind {
	case ActionShift:
		return "shift " + strconv.Itoa(action.TargetState)
	case ActionReduce:
		return "reduce " + strconv.Itoa(action.ProductionID)
	case ActionAccept:
		return "accept"
	default:
		return "error"
	}
}

func describeProduction(g *Grammar, id int) string {
	production, ok := productionByID(g, id)
	if !ok {
		return "production #" + strconv.Itoa(id)
	}
	if len(production.Body) == 0 {
		return production.Head + " -> " + Epsilon
	}
	parts := make([]string, len(production.Body))
	for i, symbol := range production.Body {
		parts[i] = symbol.Name
	}
	return production.Head + " -> " + strings.Join(parts, " ")
}

// Package yapar contiene la representación base de la tabla ACTION/GOTO.
package yapar

import "sort"

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

// BuildSLRTable reserva la construcción de la tabla SLR(1).
func BuildSLRTable(g *Grammar, ff *FirstFollow, states []State, transitions map[int]map[string]int) (*ParsingTable, error) {
	_ = g
	_ = ff
	_ = states
	return &ParsingTable{
		Action: make(map[int]map[string]Action),
		Goto:   cloneTransitions(transitions),
	}, ErrNotImplemented
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

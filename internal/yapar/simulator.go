// Package yapar contiene el runtime mínimo que consumirá la tabla LR.
package yapar

import (
	"fmt"

	"genanalex/internal/shared"
)

// ParseResult reporta el resultado mínimo del simulador sintáctico.
type ParseResult struct {
	Accepted bool
}

// ParseTokens ejecuta el runtime SLR/LR sobre un stream de tokens.
func ParseTokens(g *Grammar, table *ParsingTable, tokens []shared.Token) (*ParseResult, error) {
	if g == nil {
		return &ParseResult{Accepted: false}, fmt.Errorf("yapar: grammar is required")
	}
	if table == nil {
		return &ParseResult{Accepted: false}, fmt.Errorf("yapar: parsing table is required")
	}

	filtered := FilterIgnoredTokens(tokens, g.IgnoreSet)
	stream := append([]shared.Token{}, filtered...)
	stream = append(stream, shared.Token{Type: EndMarker, Line: endOfInputLine(tokens)})

	stateStack := []int{0}
	index := 0

	for index < len(stream) {
		state := stateStack[len(stateStack)-1]
		lookahead := stream[index]
		action := lookupAction(table, state, lookahead.Type)

		switch action.Kind {
		case ActionShift:
			stateStack = append(stateStack, action.TargetState)
			index++
		case ActionReduce:
			production, ok := productionByID(g, action.ProductionID)
			if !ok {
				return &ParseResult{Accepted: false}, fmt.Errorf("yapar: unknown production id %d", action.ProductionID)
			}
			if len(stateStack)-1 < len(production.Body) {
				return &ParseResult{Accepted: false}, fmt.Errorf("yapar: invalid reduce stack for production %s", describeProduction(g, production.ID))
			}
			stateStack = stateStack[:len(stateStack)-len(production.Body)]
			gotoState, ok := lookupGoto(table, stateStack[len(stateStack)-1], production.Head)
			if !ok {
				return &ParseResult{Accepted: false}, fmt.Errorf("yapar: missing goto from state %d on %q", stateStack[len(stateStack)-1], production.Head)
			}
			stateStack = append(stateStack, gotoState)
		case ActionAccept:
			return &ParseResult{Accepted: true}, nil
		default:
			return &ParseResult{Accepted: false}, &SyntaxError{
				Line:     syntaxErrorLine(lookahead, tokens),
				GotType:  lookahead.Type,
				Lexeme:   lookahead.Lexeme,
				Expected: table.ExpectedTokens(state),
			}
		}
	}

	return &ParseResult{Accepted: false}, &SyntaxError{
		Line:     endOfInputLine(tokens),
		GotType:  EndMarker,
		Expected: table.ExpectedTokens(stateStack[len(stateStack)-1]),
	}
}

// FilterIgnoredTokens elimina tokens marcados para omisión por la gramática.
func FilterIgnoredTokens(tokens []shared.Token, ignoreSet map[string]bool) []shared.Token {
	if len(tokens) == 0 {
		return nil
	}
	filtered := make([]shared.Token, 0, len(tokens))
	for _, token := range tokens {
		if ignoreSet[token.Type] {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

func lookupAction(table *ParsingTable, state int, symbol string) Action {
	if table == nil || table.Action == nil {
		return Action{Kind: ActionError}
	}
	if row := table.Action[state]; row != nil {
		if action, ok := row[symbol]; ok {
			return action
		}
	}
	return Action{Kind: ActionError}
}

func lookupGoto(table *ParsingTable, state int, symbol string) (int, bool) {
	if table == nil || table.Goto == nil {
		return 0, false
	}
	if row := table.Goto[state]; row != nil {
		gotoState, ok := row[symbol]
		return gotoState, ok
	}
	return 0, false
}

func endOfInputLine(tokens []shared.Token) int {
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Line > 0 {
			return tokens[i].Line
		}
	}
	return 1
}

func syntaxErrorLine(lookahead shared.Token, original []shared.Token) int {
	if lookahead.Line > 0 {
		return lookahead.Line
	}
	return endOfInputLine(original)
}

// Package yapar contiene el runtime mínimo que consumirá la tabla LR.
package yapar

import "genanalex/internal/shared"

// ParseResult reporta el resultado mínimo del simulador sintáctico.
type ParseResult struct {
	Accepted bool
}

// ParseTokens ejecutará el algoritmo LR cuando la tabla esté implementada.
func ParseTokens(g *Grammar, table *ParsingTable, tokens []shared.Token) (*ParseResult, error) {
	_ = g
	_ = table
	_ = tokens
	return &ParseResult{Accepted: false}, ErrNotImplemented
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

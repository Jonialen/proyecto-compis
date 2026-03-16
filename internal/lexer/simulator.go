// Package lexer provides the simulation engine used to identify tokens
// in a source file using one or more Deterministic Finite Automata (DFAs).
package lexer

import (
	"fmt"

	"genanalex/internal/dfa"
)

// Token represents a single lexical unit identified by the lexer.
type Token struct {
	Type   string // The type of token (e.g., "ID", "NUMBER", "OPERATOR").
	Lexeme string // The actual string from the source that matched the pattern.
	Line   int    // The 1-indexed line number where the token was found.
}

// DFAEntry bundles a DFA with metadata required for the simulation process.
type DFAEntry struct {
	DFA       *dfa.DFA // The minimized DFA used for recognition.
	TokenName string   // The label to assign to the token if this DFA matches.
	Priority  int      // Lower numbers indicate higher priority for disambiguation.
}

// Tokenize processes a source file and converts its content into a stream of tokens.
// It follows the "Maximal Munch" principle: it always tries to find the longest
// possible match from the current position. If multiple DFAs match the same
// longest lexeme, the one with the highest priority (defined by its order in
// the .yal file) is selected.
//
// Returns:
//   - A slice of identified Tokens.
//   - A slice of error messages for any unrecognized characters.
func Tokenize(dfas []DFAEntry, src *SourceFile) ([]Token, []string) {
	var tokens []Token
	var errors []string

	runes := []rune(src.Content)
	i := 0    // Pointer to the current character in the input runes.
	line := 1 // Current line counter.

	for i < len(runes) {
		// Temporary state tracking for all DFAs running in parallel.
		type dfaStatus struct {
			entry   DFAEntry
			current int
			active  bool
		}

		// Initialize all DFAs at their respective start states.
		states := make([]dfaStatus, len(dfas))
		for k, entry := range dfas {
			states[k] = dfaStatus{
				entry:   entry,
				current: entry.DFA.Start,
				active:  true,
			}
		}

		lastOKPos := -1 // End position of the longest match found so far.
		var lastOKMatches []DFAEntry

		// Lookahead loop (Maximal Munch).
		j := i
		for j < len(runes) {
			c := runes[j]
			anyActive := false

			// Attempt to transition all currently active DFAs with symbol 'c'.
			for k := range states {
				if !states[k].active {
					continue
				}
				s := &states[k]
				transitions := s.entry.DFA.Transitions[s.current]
				if nextState, ok := transitions[c]; ok {
					s.current = nextState
					anyActive = true
				} else {
					s.active = false // This DFA cannot match any further.
				}
			}

			// Stop when no DFA can consume the current character.
			if !anyActive {
				break
			}

			j++

			// Check if any of the DFAs that just transitioned are in an accepting state.
			var currentAccepting []DFAEntry
			for k := range states {
				if states[k].active && states[k].entry.DFA.Accepting[states[k].current] {
					currentAccepting = append(currentAccepting, states[k].entry)
				}
			}

			// If we have acceptance, record it as the new longest potential match.
			if len(currentAccepting) > 0 {
				lastOKPos = j
				lastOKMatches = currentAccepting
			}
		}

		// If no match was found for even a single character, it's a lexical error.
		if lastOKPos == -1 {
			errors = append(errors, fmt.Sprintf("line %d: unrecognized character %q", line, runes[i]))
			if runes[i] == '\n' {
				line++
			}
			i++
			continue
		}

		// Disambiguation: Pick the match from the rule with the highest priority.
		lexeme := string(runes[i:lastOKPos])
		bestMatch := lastOKMatches[0]
		for _, m := range lastOKMatches[1:] {
			if m.Priority < bestMatch.Priority {
				bestMatch = m
			}
		}

		// "skip" is a special action that discards the matched lexeme (e.g., for whitespace).
		if bestMatch.TokenName != "skip" {
			tokens = append(tokens, Token{
				Type:   bestMatch.TokenName,
				Lexeme: lexeme,
				Line:   line,
			})
		}

		// Track line numbers within the matched lexeme (important for multi-line tokens).
		for _, r := range lexeme {
			if r == '\n' {
				line++
			}
		}

		// Advance the main pointer to the end of the recognized lexeme.
		i = lastOKPos
	}

	return tokens, errors
}

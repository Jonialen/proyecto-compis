package lexer

import (
	"fmt"

	"genanalex/internal/dfa"
)

// Token represents a lexical token with its type, lexeme, and source line.
type Token struct {
	Type   string
	Lexeme string
	Line   int
}

// DFAEntry associates a DFA with the token it recognizes and its priority.
type DFAEntry struct {
	DFA       *dfa.DFA
	TokenName string
	Priority  int
}

// Tokenize runs the maximal munch algorithm with priority-based disambiguation.
// Returns (tokens, errors).
// Tokens with action "skip" are discarded silently.
func Tokenize(dfas []DFAEntry, src *SourceFile) ([]Token, []string) {
	var tokens []Token
	var errors []string

	runes := []rune(src.Content)
	i := 0
	line := 1

	for i < len(runes) {
		// Run all DFAs in parallel from their initial states
		type dfaState struct {
			entry   DFAEntry
			current int  // current state
			active  bool // still alive
		}

		states := make([]dfaState, len(dfas))
		for k, entry := range dfas {
			states[k] = dfaState{
				entry:   entry,
				current: entry.DFA.Start,
				active:  true,
			}
		}

		lastOKPos := -1
		lastOKMatches := []DFAEntry{} // all DFAs that accepted at lastOKPos

		j := i
		for j < len(runes) {
			c := runes[j]

			// Advance all active DFAs
			anyActive := false
			for k := range states {
				if !states[k].active {
					continue
				}
				entry := states[k]
				transitions := entry.entry.DFA.Transitions[entry.current]
				nextState, ok := transitions[c]
				if !ok {
					states[k].active = false
				} else {
					states[k].current = nextState
					states[k].active = true
					anyActive = true
				}
			}

			if !anyActive {
				break
			}

			j++

			// Check if any DFA accepts now (after consuming runes[i..j-1])
			var currentAccepting []DFAEntry
			for k := range states {
				if states[k].active && states[k].entry.DFA.Accepting[states[k].current] {
					currentAccepting = append(currentAccepting, states[k].entry)
				}
			}
			if len(currentAccepting) > 0 {
				lastOKPos = j
				lastOKMatches = currentAccepting
			}
		}

		if lastOKPos == -1 {
			// Lexical error
			errors = append(errors, fmt.Sprintf("line %d: unrecognized character %q", line, runes[i]))
			// Count newlines in skipped char
			if runes[i] == '\n' {
				line++
			}
			i++
			continue
		}

		// Count newlines in the matched lexeme to track line number
		lexemeForCount := string(runes[i:lastOKPos])

		// Pick the best match: lowest priority index (earliest rule wins)
		best := lastOKMatches[0]
		for _, m := range lastOKMatches[1:] {
			if m.Priority < best.Priority {
				best = m
			}
		}

		// Count newlines in lexeme for line tracking (done after selecting lexeme)

		// Emit token if not skip
		if best.TokenName != "skip" {
			tokens = append(tokens, Token{
				Type:   best.TokenName,
				Lexeme: lexemeForCount,
				Line:   line,
			})
		}

		// Update line counter
		for _, r := range lexemeForCount {
			if r == '\n' {
				line++
			}
		}

		i = lastOKPos
	}

	return tokens, errors
}

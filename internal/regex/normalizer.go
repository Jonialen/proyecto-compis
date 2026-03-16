// Package regex provides tools for normalizing, tokenizing, and converting
// regular expressions into postfix notation suitable for DFA construction.
package regex

import (
	"fmt"
	"strings"
	"unicode"
)

// ConcatOp is a sentinel rune representing the explicit concatenation operator.
// It is inserted during normalization to facilitate syntax tree building.
const ConcatOp = '\x01'

// EndMarker is a special augmented symbol used to mark the end of a regular expression.
// It is essential for identifying accepting states during direct DFA construction.
const EndMarker = '\x00'

// TokKind defines the type of a regular expression token.
type TokKind int

const (
	TokAtom  TokKind = iota // A literal character or a symbol from the alphabet.
	TokOp                   // A regex operator (|, *, +, ?, etc.).
	TokOpen                 // An opening parenthesis (.
	TokClose                // A closing parenthesis ).
)

// RegexToken represents a single element in a tokenized or postfix regular expression.
type RegexToken struct {
	Kind TokKind
	Atom rune // The literal character (only valid if Kind == TokAtom).
	Op   rune // The operator symbol (only valid if Kind == TokOp, TokOpen, or TokClose).
}

// Helper functions for creating RegexToken instances.
func atomTok(r rune) RegexToken { return RegexToken{Kind: TokAtom, Atom: r} }
func opTok(op rune) RegexToken  { return RegexToken{Kind: TokOp, Op: op} }
func openTok() RegexToken       { return RegexToken{Kind: TokOpen, Op: '('} }
func closeTok() RegexToken      { return RegexToken{Kind: TokClose, Op: ')'} }

// Normalize transforms a raw YALex pattern into a sequence of RegexTokens.
// It expands character classes, handles wildcards, resolves escape sequences,
// and inserts explicit concatenation operators.
func Normalize(pattern string) ([]RegexToken, error) {
	tokens, err := tokenize(pattern)
	if err != nil {
		return nil, fmt.Errorf("tokenizing %q: %w", pattern, err)
	}
	// Post-process the tokens to insert the concatenation operator (·)
	// where concatenation is implicit in the regex syntax.
	return insertConcat(tokens), nil
}

// tokenize converts the input pattern string into a flat slice of RegexTokens.
func tokenize(pattern string) ([]RegexToken, error) {
	var tokens []RegexToken
	runes := []rune(pattern)
	i := 0

	for i < len(runes) {
		c := runes[i]

		switch c {
		case '(':
			tokens = append(tokens, openTok())
			i++
		case ')':
			tokens = append(tokens, closeTok())
			i++
		case '|':
			tokens = append(tokens, opTok('|'))
			i++
		case '*':
			tokens = append(tokens, opTok('*'))
			i++
		case '+':
			tokens = append(tokens, opTok('+'))
			i++
		case '?':
			tokens = append(tokens, opTok('?'))
			i++

		case '\'':
			// Handle single-quoted character literals: 'a', '\n', '\'', etc.
			i++ // Skip opening quote
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, err
			}
			i = newI
			if i >= len(runes) || runes[i] != '\'' {
				return nil, fmt.Errorf("expected closing ' at position %d", i)
			}
			i++ // Skip closing quote
			tokens = append(tokens, atomTok(r))

		case '"':
			// Handle double-quoted string literals: "abc" -> (a·b·c).
			i++ // Skip opening quote
			var chars []rune
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					r, err := parseEscape(runes[i+1])
					if err != nil {
						return nil, err
					}
					chars = append(chars, r)
					i += 2
				} else {
					chars = append(chars, runes[i])
					i++
				}
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unclosed double quote")
			}
			i++ // Skip closing quote

			if len(chars) == 0 {
				// Empty string (epsilon) - currently skipped as we don't handle empty transitions.
			} else if len(chars) == 1 {
				tokens = append(tokens, atomTok(chars[0]))
			} else {
				// Strings are converted to a grouped sequence of concatenated atoms.
				tokens = append(tokens, openTok())
				for j, r := range chars {
					tokens = append(tokens, atomTok(r))
					if j < len(chars)-1 {
						tokens = append(tokens, opTok(ConcatOp))
					}
				}
				tokens = append(tokens, closeTok())
			}

		case '[':
			// Handle character classes like [a-z], [0-9], [^0-9], etc.
			i++ // Skip [
			classTokens, newI, err := expandCharClass(runes, i)
			if err != nil {
				return nil, fmt.Errorf("expanding char class: %w", err)
			}
			i = newI
			tokens = append(tokens, classTokens...)

		case '.':
			// Wildcard: matches any character in the alphabet (excluding \n).
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		case '_':
			// Alternative wildcard syntax (common in YALex): matches any character.
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		default:
			if unicode.IsSpace(c) {
				i++
				continue
			}
			// Treat any other character as a literal atom.
			tokens = append(tokens, atomTok(c))
			i++
		}
	}

	return tokens, nil
}

// parseSingleQuoted extracts a literal character from a single-quoted sequence.
func parseSingleQuoted(runes []rune, i int) (rune, int, error) {
	if i >= len(runes) {
		return 0, i, fmt.Errorf("unexpected end inside single quote")
	}
	if runes[i] == '\\' {
		if i+1 >= len(runes) {
			return 0, i, fmt.Errorf("unexpected end after backslash")
		}
		r, err := parseEscape(runes[i+1])
		if err != nil {
			return 0, i, err
		}
		return r, i + 2, nil
	}
	return runes[i], i + 1, nil
}

// parseEscape maps an escaped character to its corresponding rune value.
func parseEscape(c rune) (rune, error) {
	switch c {
	case 'n':
		return '\n', nil
	case 't':
		return '\t', nil
	case 'r':
		return '\r', nil
	case '\\':
		return '\\', nil
	case '\'':
		return '\'', nil
	case '"':
		return '"', nil
	case '0':
		return '\x00', nil
	default:
		// Unknown escape sequences are treated as the literal character.
		return c, nil
	}
}

// expandCharClass parses a [...] block and converts it into a grouped alternation (a|b|c...).
func expandCharClass(runes []rune, i int) ([]RegexToken, int, error) {
	complement := false
	if i < len(runes) && runes[i] == '^' {
		complement = true
		i++
	}

	var chars []rune

	for i < len(runes) && runes[i] != ']' {
		c := runes[i]

		if c == '\'' {
			// Handle quoted characters within the class, e.g., [' ' '\t'].
			i++
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, i, err
			}
			i = newI
			if i < len(runes) && runes[i] == '\'' {
				i++
			}
			chars = append(chars, r)
		} else if c == '\\' {
			// Handle escape sequences within the class.
			if i+1 >= len(runes) {
				return nil, i, fmt.Errorf("unexpected end after backslash in class")
			}
			r, err := parseEscape(runes[i+1])
			if err != nil {
				return nil, i, err
			}
			i += 2
			// Check for character ranges starting with an escape, e.g., [\x00-\xFF].
			if i+1 < len(runes) && runes[i] == '-' && runes[i+1] != ']' {
				i++
				end := runes[i]
				i++
				for r2 := r; r2 <= end; r2++ {
					chars = append(chars, r2)
				}
			} else {
				chars = append(chars, r)
			}
		} else if c == ' ' || c == '\t' {
			// Whitespace acts as a separator in YALex character classes.
			i++
		} else {
			// Handle literal character ranges, e.g., [a-z].
			if i+2 < len(runes) && runes[i+1] == '-' && runes[i+2] != ']' {
				start := c
				end := runes[i+2]
				i += 3
				for r := start; r <= end; r++ {
					chars = append(chars, r)
				}
			} else {
				chars = append(chars, c)
				i++
			}
		}
	}

	if i < len(runes) && runes[i] == ']' {
		i++
	}

	if complement {
		// For [^...], include all alphabet symbols NOT present in the specified set.
		alphabet := buildAlphabet()
		exclude := make(map[rune]bool)
		for _, c := range chars {
			exclude[c] = true
		}
		var remaining []rune
		for _, r := range alphabet {
			if !exclude[r] {
				remaining = append(remaining, r)
			}
		}
		chars = remaining
	}

	if len(chars) == 0 {
		return nil, i, fmt.Errorf("empty character class")
	}

	return buildAlternationGroup(chars), i, nil
}

// buildAlternationGroup creates a grouped (a|b|c...) token sequence from a list of symbols.
func buildAlternationGroup(chars []rune) []RegexToken {
	if len(chars) == 1 {
		return []RegexToken{atomTok(chars[0])}
	}
	tokens := []RegexToken{openTok()}
	for i, r := range chars {
		tokens = append(tokens, atomTok(r))
		if i < len(chars)-1 {
			tokens = append(tokens, opTok('|'))
		}
	}
	tokens = append(tokens, closeTok())
	return tokens
}

// buildWildcardTokens creates the token group representing any character in the alphabet.
func buildWildcardTokens() []RegexToken {
	return buildAlternationGroup(buildAlphabet())
}

// buildAlphabet returns the set of printable ASCII characters plus common control chars.
func buildAlphabet() []rune {
	var alphabet []rune
	alphabet = append(alphabet, '\t', '\r')
	for r := rune(32); r <= 126; r++ {
		alphabet = append(alphabet, r)
	}
	return alphabet
}

// insertConcat adds explicit concatenation operators (·) between adjacent tokens.
// This simplifies the conversion to postfix notation and syntax tree building.
func insertConcat(tokens []RegexToken) []RegexToken {
	var result []RegexToken

	for i, tok := range tokens {
		result = append(result, tok)
		if i+1 < len(tokens) {
			if needsConcat(tokens[i], tokens[i+1]) {
				result = append(result, opTok(ConcatOp))
			}
		}
	}

	return result
}

// needsConcat determines if a concatenation operator is required between two adjacent tokens.
func needsConcat(left, right RegexToken) bool {
	// An operator is needed if the left token "ends" an expression (atom, close paren, or quantifier)
	// and the right token "starts" an expression (atom or open paren).
	leftOutput := left.Kind == TokAtom ||
		left.Kind == TokClose ||
		(left.Kind == TokOp && (left.Op == '*' || left.Op == '+' || left.Op == '?'))

	rightInput := right.Kind == TokAtom || right.Kind == TokOpen

	return leftOutput && rightInput
}

// TokensToString converts a slice of RegexTokens into a human-readable string for debugging.
func TokensToString(tokens []RegexToken) string {
	var sb strings.Builder
	for _, tok := range tokens {
		switch tok.Kind {
		case TokAtom:
			sb.WriteString(fmt.Sprintf("A(%q)", tok.Atom))
		case TokOp:
			if tok.Op == ConcatOp {
				sb.WriteString("·")
			} else {
				sb.WriteRune(tok.Op)
			}
		case TokOpen:
			sb.WriteRune('(')
		case TokClose:
			sb.WriteRune(')')
		}
	}
	return sb.String()
}

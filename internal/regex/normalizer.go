package regex

import (
	"fmt"
	"strings"
	"unicode"
)

// ConcatOp is a sentinel rune for the concatenation operator.
// Used ONLY in the Token.Op field, never as an atom.
const ConcatOp = '\x01'

// EndMarker is the augmented end-marker symbol.
// Used ONLY as an atom symbol, never in real input.
const EndMarker = '\x00'

// TokKind distinguishes operator tokens from atom tokens.
type TokKind int

const (
	TokAtom  TokKind = iota // a literal character
	TokOp                   // an operator: |, ConcatOp, *, +, ?, (, )
	TokOpen                 // (
	TokClose                // )
)

// RegexToken is a single element in a tokenized/postfix regex.
type RegexToken struct {
	Kind TokKind
	Atom rune // valid when Kind == TokAtom
	Op   rune // valid when Kind == TokOp (or Open/Close)
}

func atomTok(r rune) RegexToken { return RegexToken{Kind: TokAtom, Atom: r} }
func opTok(op rune) RegexToken  { return RegexToken{Kind: TokOp, Op: op} }
func openTok() RegexToken       { return RegexToken{Kind: TokOpen, Op: '('} }
func closeTok() RegexToken      { return RegexToken{Kind: TokClose, Op: ')'} }

// Normalize converts a .yal pattern string into a token slice with
// explicit concatenation operators inserted.
func Normalize(pattern string) ([]RegexToken, error) {
	tokens, err := tokenize(pattern)
	if err != nil {
		return nil, fmt.Errorf("tokenizing %q: %w", pattern, err)
	}
	return insertConcat(tokens), nil
}

// tokenize parses the pattern string into a flat slice of RegexTokens.
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
			// Single-quoted literal: 'c', '\n', etc.
			i++ // skip opening '
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, err
			}
			i = newI
			if i >= len(runes) || runes[i] != '\'' {
				return nil, fmt.Errorf("expected closing ' at position %d", i)
			}
			i++ // skip closing '
			tokens = append(tokens, atomTok(r))

		case '"':
			// Double-quoted string: "abc" → atoms for a, b, c
			i++ // skip opening "
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
			i++ // skip closing "
			if len(chars) == 0 {
				// empty string - epsilon, skip
			} else if len(chars) == 1 {
				tokens = append(tokens, atomTok(chars[0]))
			} else {
				// Wrap in a group, insert explicit concats
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
			// Character class [...]
			i++ // skip [
			classTokens, newI, err := expandCharClass(runes, i)
			if err != nil {
				return nil, fmt.Errorf("expanding char class: %w", err)
			}
			i = newI
			tokens = append(tokens, classTokens...)

		case '.':
			// Wildcard
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		case '_':
			// Underscore wildcard: matches any character (same as '.')
			classTokens := buildWildcardTokens()
			tokens = append(tokens, classTokens...)
			i++

		default:
			if unicode.IsSpace(c) {
				i++
				continue
			}
			// Regular character literal
			tokens = append(tokens, atomTok(c))
			i++
		}
	}

	return tokens, nil
}

// parseSingleQuoted parses the content of a single-quoted literal.
// i points to the first char after the opening '.
// Returns (rune, newIndex) where newIndex points past the last content char
// (the closing ' will be consumed by caller).
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

// parseEscape converts an escape character (char after \) to its rune value.
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
		return c, nil
	}
}

// expandCharClass expands a [...] character class into a group of alternation tokens.
// i points past the '['. Returns the token group and new index (past ']').
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
			// single-quoted char inside class like [' ' '\t']
			i++ // skip '
			r, newI, err := parseSingleQuoted(runes, i)
			if err != nil {
				return nil, i, err
			}
			i = newI
			if i < len(runes) && runes[i] == '\'' {
				i++ // skip closing '
			}
			chars = append(chars, r)
		} else if c == '\\' {
			if i+1 >= len(runes) {
				return nil, i, fmt.Errorf("unexpected end after backslash in class")
			}
			r, err := parseEscape(runes[i+1])
			if err != nil {
				return nil, i, err
			}
			i += 2
			// Check for range after escaped char
			if i+1 < len(runes) && runes[i] == '-' && runes[i+1] != ']' {
				i++ // skip -
				end := runes[i]
				i++
				for r2 := r; r2 <= end; r2++ {
					chars = append(chars, r2)
				}
			} else {
				chars = append(chars, r)
			}
		} else if c == ' ' || c == '\t' {
			// Unquoted whitespace inside [...] is a separator (not a literal char)
			// e.g., [^ '\n'] has a space between ^ and '\n' that's just formatting
			i++
		} else {
			// Check for range like a-z
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
		i++ // skip ]
	}

	if complement {
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

// buildAlternationGroup creates a group (a|b|c...) from a list of runes.
func buildAlternationGroup(chars []rune) []RegexToken {
	if len(chars) == 1 {
		// No need for grouping
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

// buildWildcardTokens builds the token group for the '.' wildcard.
func buildWildcardTokens() []RegexToken {
	return buildAlternationGroup(buildAlphabet())
}

// buildAlphabet returns all characters for complement classes and wildcards.
// Includes ASCII 32-126 plus \t and \r, but NOT \n (conventional . behavior).
func buildAlphabet() []rune {
	var alphabet []rune
	alphabet = append(alphabet, '\t', '\r')
	for r := rune(32); r <= 126; r++ {
		alphabet = append(alphabet, r)
	}
	return alphabet
}

// insertConcat inserts explicit ConcatOp tokens between elements that should be concatenated.
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

// needsConcat returns true if a ConcatOp should be inserted between left and right.
func needsConcat(left, right RegexToken) bool {
	leftOutput := left.Kind == TokAtom ||
		left.Kind == TokClose ||
		(left.Kind == TokOp && (left.Op == '*' || left.Op == '+' || left.Op == '?'))

	rightInput := right.Kind == TokAtom || right.Kind == TokOpen

	return leftOutput && rightInput
}

// TokensToString converts a token slice to a debug string for display.
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

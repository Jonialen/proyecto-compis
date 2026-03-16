// Package yalex provides a parser for the YALex (Yet Another Lex) specification language.
// It translates .yal files into in-memory structures that define macros and token rules.
package yalex

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TokenRule represents a single lexical rule extracted from the .yal file.
// It consists of a regular expression pattern and an associated action string.
type TokenRule struct {
	Pattern  string // The regular expression pattern to match.
	Action   string // The action string (usually a token name) associated with the pattern.
	Priority int    // The 0-indexed order of appearance in the .yal file, used for disambiguation.
}

// ParseResult encapsulates the components of a parsed YALex specification.
type ParseResult struct {
	Macros map[string]string // Reusable regular expression definitions (let IDENT = regexp).
	Rules  []TokenRule       // The sequence of token rules (rule entrypoint = | pattern { action }).
}

// ParseFile reads a YALex specification from the given file path and returns its parsed representation.
func ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading yal file: %w", err)
	}
	return Parse(string(data))
}

// Parse processes the raw content of a .yal file. It performs comment removal,
// header/trailer extraction, macro collection, and rule parsing.
func Parse(content string) (*ParseResult, error) {
	// Step 1: Remove YALex comments delimited by (* and *).
	content = removeComments(content)

	// Step 2: Remove top-level header and trailer blocks defined within braces.
	content = removeHeaderTrailer(content)

	result := &ParseResult{
		Macros: make(map[string]string),
	}

	// Step 3: Extract macro definitions: let IDENT = regexp.
	letRe := regexp.MustCompile(`(?m)^[ \t]*let[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(.+)$`)
	for _, m := range letRe.FindAllStringSubmatch(content, -1) {
		name := strings.TrimSpace(m[1])
		val := strings.TrimSpace(m[2])
		result.Macros[name] = val
	}

	// Step 4: Identify the rule section (rule entrypoint = ...).
	ruleRe := regexp.MustCompile(`(?s)rule\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*(.+)`)
	ruleMatch := ruleRe.FindStringSubmatch(content)
	if ruleMatch == nil {
		return result, nil
	}
	ruleBody := ruleMatch[1]

	// Step 5: Parse the individual rules: | pattern { ACTION }.
	rules, err := parseRules(ruleBody)
	if err != nil {
		return nil, err
	}
	result.Rules = rules
	return result, nil
}

// removeComments strips multiline (* ... *) comments from the input string.
func removeComments(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '(' && s[i+1] == '*' {
			// Find the corresponding closing *) sequence.
			end := strings.Index(s[i+2:], "*)")
			if end == -1 {
				// If the comment is unclosed, we stop processing.
				break
			}
			i = i + 2 + end + 2
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

// removeHeaderTrailer isolates and removes { header } and { trailer } blocks
// that appear outside the rule definition area.
func removeHeaderTrailer(s string) string {
	ruleIdx := strings.Index(s, "rule ")
	if ruleIdx == -1 {
		ruleIdx = len(s)
	}

	prefix := s[:ruleIdx]
	suffix := s[ruleIdx:]

	prefix = removeTopLevelBraces(prefix)
	suffix = removeTrailingBraceBlock(suffix)

	return prefix + suffix
}

// removeTrailingBraceBlock identifies and removes the optional trailer block
// at the end of the specification.
func removeTrailingBraceBlock(s string) string {
	lastClose := strings.LastIndex(s, "}")
	if lastClose == -1 {
		return s
	}

	runes := []rune(s)
	depth := 0
	trailerStart := -1
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '}' {
			depth++
		} else if runes[i] == '{' {
			depth--
			if depth == 0 {
				trailerStart = i
				break
			}
		}
	}
	if trailerStart == -1 {
		return s
	}

	before := strings.TrimSpace(string(runes[:trailerStart]))
	if !strings.HasSuffix(before, "}") {
		return s
	}

	return before
}

// removeTopLevelBraces removes balanced { ... } blocks from the provided string.
func removeTopLevelBraces(s string) string {
	var buf strings.Builder
	depth := 0
	for _, c := range s {
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		} else if depth == 0 {
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// parseRules breaks down the rule body into individual pattern-action pairs.
// It handles grouped rules where multiple patterns share a single action.
func parseRules(body string) ([]TokenRule, error) {
	var rules []TokenRule
	priority := 0

	segments := splitByPipe(body)

	var pending []string // Patterns accumulated before an action block is encountered.
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		_, _, err := extractPatternAction(seg)
		hasAction := (err == nil)

		if hasAction {
			pending = append(pending, seg)
			_, action, _ := extractPatternAction(pending[len(pending)-1])
			action = strings.TrimSpace(action)

			for _, p := range pending {
				pattern, _, _ := extractPatternAction(p)
				pattern = strings.TrimSpace(pattern)
				if pattern == "" {
					continue
				}
				rules = append(rules, TokenRule{
					Pattern:  pattern,
					Action:   action,
					Priority: priority,
				})
				priority++
			}
			pending = pending[:0]
		} else {
			pending = append(pending, seg)
		}
	}

	for _, p := range pending {
		if strings.TrimSpace(p) != "" {
			return nil, fmt.Errorf("parsing rule %q: no action block found", p)
		}
	}

	return rules, nil
}

// splitByPipe divides the rule body by top-level pipe characters (|),
// correctly ignoring pipes contained within quotes, brackets, or braces.
func splitByPipe(s string) []string {
	var parts []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	inBracket := 0
	inBrace := 0
	inParen := 0

	i := 0
	runes := []rune(s)
	for i < len(runes) {
		c := runes[i]
		switch {
		case c == '\'' && !inDouble && inBracket == 0:
			inSingle = !inSingle
			cur.WriteRune(c)
		case inSingle:
			if c == '\\' && i+1 < len(runes) {
				cur.WriteRune(c)
				i++
				cur.WriteRune(runes[i])
			} else {
				cur.WriteRune(c)
			}
		case c == '"' && !inSingle && inBracket == 0:
			inDouble = !inDouble
			cur.WriteRune(c)
		case c == '[' && !inSingle && !inDouble:
			inBracket++
			cur.WriteRune(c)
		case c == ']' && !inSingle && !inDouble:
			if inBracket > 0 {
				inBracket--
			}
			cur.WriteRune(c)
		case c == '(' && !inSingle && !inDouble && inBracket == 0:
			inParen++
			cur.WriteRune(c)
		case c == ')' && !inSingle && !inDouble && inBracket == 0:
			if inParen > 0 {
				inParen--
			}
			cur.WriteRune(c)
		case c == '{' && !inSingle && !inDouble:
			inBrace++
			cur.WriteRune(c)
		case c == '}' && !inSingle && !inDouble:
			inBrace--
			cur.WriteRune(c)
		case c == '|' && !inSingle && !inDouble && inBracket == 0 && inBrace == 0 && inParen == 0:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(c)
		}
		i++
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// extractPatternAction separates a rule segment into its pattern and action components.
func extractPatternAction(line string) (pattern, action string, err error) {
	runes := []rune(line)
	inSingle := false
	inDouble := false
	inBracket := 0
	actionStart := -1

	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '\'' && !inDouble && inBracket == 0:
			inSingle = !inSingle
		case inSingle:
			if c == '\\' && i+1 < len(runes) {
				i++
			}
		case c == '"' && !inSingle && inBracket == 0:
			inDouble = !inDouble
		case c == '[' && !inSingle && !inDouble:
			inBracket++
		case c == ']' && !inSingle && !inDouble:
			if inBracket > 0 {
				inBracket--
			}
		case c == '{' && !inSingle && !inDouble && inBracket == 0:
			actionStart = i
			goto foundAction
		}
	}

foundAction:
	if actionStart == -1 {
		return line, "", fmt.Errorf("no action block found")
	}

	depth := 0
	actionEnd := -1
	for i := actionStart; i < len(runes); i++ {
		if runes[i] == '{' {
			depth++
		} else if runes[i] == '}' {
			depth--
			if depth == 0 {
				actionEnd = i
				break
			}
		}
	}
	if actionEnd == -1 {
		return "", "", fmt.Errorf("unclosed action block")
	}

	pattern = string(runes[:actionStart])
	action = string(runes[actionStart+1 : actionEnd])
	return pattern, action, nil
}

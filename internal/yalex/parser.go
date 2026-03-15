package yalex

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TokenRule represents a single lexer rule parsed from the .yal file.
type TokenRule struct {
	Pattern  string
	Action   string
	Priority int // 0-indexed order of appearance
}

// ParseResult holds the parsed macros and rules from a .yal file.
type ParseResult struct {
	Macros map[string]string
	Rules  []TokenRule
}

// ParseFile reads and parses a .yal file.
func ParseFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading yal file: %w", err)
	}
	return Parse(string(data))
}

// Parse parses the content of a .yal file.
func Parse(content string) (*ParseResult, error) {
	// Step 1: remove (* ... *) comments (possibly multiline)
	content = removeComments(content)

	// Step 2: remove top-level { header } and { trailer } blocks
	content = removeHeaderTrailer(content)

	result := &ParseResult{
		Macros: make(map[string]string),
	}

	// Step 3: extract macros (let IDENT = regexp)
	letRe := regexp.MustCompile(`(?m)^[ \t]*let[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(.+)$`)
	for _, m := range letRe.FindAllStringSubmatch(content, -1) {
		name := strings.TrimSpace(m[1])
		val := strings.TrimSpace(m[2])
		result.Macros[name] = val
	}

	// Step 4: find the rule section
	ruleRe := regexp.MustCompile(`(?s)rule\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*(.+)`)
	ruleMatch := ruleRe.FindStringSubmatch(content)
	if ruleMatch == nil {
		return result, nil
	}
	ruleBody := ruleMatch[1]

	// Step 5: extract each rule line: | pattern { ACTION }
	rules, err := parseRules(ruleBody)
	if err != nil {
		return nil, err
	}
	result.Rules = rules
	return result, nil
}

// removeComments removes (* ... *) comments including multiline ones.
func removeComments(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '(' && s[i+1] == '*' {
			// find closing *)
			end := strings.Index(s[i+2:], "*)")
			if end == -1 {
				// unclosed comment - skip to end
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

// removeHeaderTrailer removes { ... } blocks that appear before "rule" (header)
// and after the rule section (trailer), per YALex spec.
func removeHeaderTrailer(s string) string {
	// Find "rule" position; remove any balanced { } blocks before it (header).
	ruleIdx := strings.Index(s, "rule ")
	if ruleIdx == -1 {
		ruleIdx = len(s)
	}

	prefix := s[:ruleIdx]
	suffix := s[ruleIdx:]

	// Remove { ... } from prefix (header block)
	prefix = removeTopLevelBraces(prefix)

	// Remove trailing { ... } after the last rule action block (trailer block).
	// Strategy: find the last '}' that closes a rule action, then strip any
	// top-level brace blocks that appear after the entire rule section.
	// Simple approach: find the last occurrence of the pattern "} \n { ... }"
	// that appears after the rules, and strip it.
	// Robust approach: strip top-level brace blocks from the suffix ONLY after
	// the rule keyword+name line, i.e., only process braces that are not
	// part of rule action blocks.
	//
	// Since rule actions are also { } blocks, we can't naively strip all braces
	// from suffix. Instead we only strip a trailing brace block that appears
	// AFTER all pipe-rules have ended (i.e., after the last | ... { } rule).
	// Heuristic: find the last '}' in suffix, then check if there's more content
	// after it that forms a { trailer } block.
	suffix = removeTrailingBraceBlock(suffix)

	return prefix + suffix
}

// removeTrailingBraceBlock removes a trailing top-level { ... } block from the
// end of a string (the YALex trailer block), if present.
func removeTrailingBraceBlock(s string) string {
	// Find the last '}' in the string
	lastClose := strings.LastIndex(s, "}")
	if lastClose == -1 {
		return s
	}

	// Check if there's a '{' that opens a top-level block after all rule actions.
	// We scan from the end: find the matching '{' for the last '}'.
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

	// Check that everything before trailerStart (trimmed) ends with a rule action '}'
	// i.e., the trailer block is genuinely after all rules, not the last rule action.
	// Heuristic: the last rule action '}' appears on a line with a pipe '|' somewhere
	// before it. If the content between the previous '}' and trailerStart is only
	// whitespace/newlines, it's a trailer block.
	before := strings.TrimSpace(string(runes[:trailerStart]))
	if !strings.HasSuffix(before, "}") {
		// The found brace block is the ONLY/last rule action — don't strip it.
		return s
	}

	// It's a trailer block — remove it
	return before
}

// removeTopLevelBraces removes balanced { ... } blocks from a string.
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

// parseRules parses the rule body into TokenRule entries.
// Each rule is: | pattern { ACTION }
// A single rule may also span multiple pipe-alternatives sharing one action:
//
//	| 'T' | 'F'  { BOOLEAN }
//
// In that case we emit one TokenRule per alternative, all with the same action.
func parseRules(body string) ([]TokenRule, error) {
	var rules []TokenRule
	priority := 0

	// splitByPipe gives us the raw segments between top-level '|' characters.
	// We need to re-group them: a group ends when a segment contains a { action }.
	segments := splitByPipe(body)

	var pending []string // segments accumulated without an action yet
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// Does this segment contain an action block { ... }?
		_, _, err := extractPatternAction(seg)
		hasAction := (err == nil)

		if hasAction {
			// This segment closes the current group.
			pending = append(pending, seg)
			// The last element of pending has the action; prepend it to earlier ones.
			_, action, _ := extractPatternAction(pending[len(pending)-1])
			action = strings.TrimSpace(action)

			for _, p := range pending {
				// Strip the action block from the last segment to get its pattern.
				pattern, _, _ := extractPatternAction(p)
				pattern = strings.TrimSpace(pattern)
				// For earlier segments that had no action block, pattern == p trimmed.
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
			// No action yet — accumulate as part of the current group.
			pending = append(pending, seg)
		}
	}

	// Any leftover pending segments have no action — report error.
	for _, p := range pending {
		p = strings.TrimSpace(p)
		if p != "" {
			return nil, fmt.Errorf("parsing rule %q: no action block found", p)
		}
	}

	return rules, nil
}

// splitByPipe splits the rule body by '|' characters that are not inside
// single quotes, double quotes, square brackets, parentheses, or braces.
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
			if !inSingle {
				// Opening single quote
				inSingle = true
				cur.WriteRune(c)
			} else {
				// Closing single quote
				inSingle = false
				cur.WriteRune(c)
			}
		case inSingle:
			// Inside single quote: handle escape sequences like '\''
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

// extractPatternAction splits a rule line into pattern and action.
// The action is the last { ... } block.
func extractPatternAction(line string) (pattern, action string, err error) {
	// Find the last '{' that opens the action block
	// We need to find { ACTION } at the end, properly handling nested braces
	// in the pattern (e.g. character classes don't use braces, so the first
	// { should be the action start)

	// Find the opening brace of the action - it's the first unescaped {
	// that is not inside quotes or square brackets
	runes := []rune(line)
	inSingle := false
	inDouble := false
	inBracket := 0
	actionStart := -1

	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '\'' && !inDouble && inBracket == 0:
			if !inSingle {
				inSingle = true
			} else {
				inSingle = false
			}
		case inSingle:
			// Inside single quote: handle escape sequences like '\''
			if c == '\\' && i+1 < len(runes) {
				i++ // skip the escaped char
			}
			// Otherwise just consume the character
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

	// Find matching closing brace
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

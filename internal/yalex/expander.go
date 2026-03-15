package yalex

import (
	"fmt"
	"strings"
)

// Expand substitutes macro references in rule patterns.
// Macros are expanded recursively in topological order.
func Expand(macros map[string]string, rules []TokenRule) ([]TokenRule, error) {
	// First, expand macros themselves (they can reference other macros)
	expanded, err := expandMacros(macros)
	if err != nil {
		return nil, err
	}

	// Then expand each rule pattern
	result := make([]TokenRule, len(rules))
	for i, rule := range rules {
		expandedPattern, err := expandPattern(rule.Pattern, expanded)
		if err != nil {
			return nil, fmt.Errorf("expanding rule %d (%q): %w", i, rule.Pattern, err)
		}
		result[i] = TokenRule{
			Pattern:  expandedPattern,
			Action:   rule.Action,
			Priority: rule.Priority,
		}
	}
	return result, nil
}

// expandMacros fully expands all macros in topological order.
func expandMacros(macros map[string]string) (map[string]string, error) {
	expanded := make(map[string]string)
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var expandOne func(name string) (string, error)
	expandOne = func(name string) (string, error) {
		if v, ok := expanded[name]; ok {
			return v, nil
		}
		if inStack[name] {
			return "", fmt.Errorf("cyclic macro reference: %s", name)
		}
		val, ok := macros[name]
		if !ok {
			return "", fmt.Errorf("undefined macro: %s", name)
		}
		inStack[name] = true
		visited[name] = true

		// Always use the recursive resolver with cycle detection.
		// Skipping the optimistic expandPattern(val, expanded) pass avoids
		// silently treating unresolved macros as literal identifiers.
		result, err := expandPatternWithResolver(val, expanded, expandOne)
		if err != nil {
			return "", fmt.Errorf("macro %s: %w", name, err)
		}

		inStack[name] = false
		expanded[name] = result
		return result, nil
	}

	for name := range macros {
		if _, err := expandOne(name); err != nil {
			return nil, err
		}
	}
	return expanded, nil
}

// expandPattern replaces macro identifiers in a pattern with their expanded values.
// Only replaces identifiers that are not inside quotes or square brackets.
func expandPattern(pattern string, macros map[string]string) (string, error) {
	return expandPatternWithResolver(pattern, macros, func(name string) (string, error) {
		if v, ok := macros[name]; ok {
			return v, nil
		}
		return "", fmt.Errorf("undefined macro: %s", name)
	})
}

// expandPatternWithResolver is the core expansion function with a custom resolver.
func expandPatternWithResolver(pattern string, macros map[string]string, resolver func(string) (string, error)) (string, error) {
	runes := []rune(pattern)
	var result strings.Builder
	i := 0

	for i < len(runes) {
		c := runes[i]

		// Handle single-quoted literals 'c'
		if c == '\'' {
			result.WriteRune(c)
			i++
			for i < len(runes) && runes[i] != '\'' {
				if runes[i] == '\\' && i+1 < len(runes) {
					result.WriteRune(runes[i])
					i++
					result.WriteRune(runes[i])
					i++
				} else {
					result.WriteRune(runes[i])
					i++
				}
			}
			if i < len(runes) {
				result.WriteRune(runes[i]) // closing '
				i++
			}
			continue
		}

		// Handle double-quoted strings "abc"
		if c == '"' {
			result.WriteRune(c)
			i++
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					result.WriteRune(runes[i])
					i++
					result.WriteRune(runes[i])
					i++
				} else {
					result.WriteRune(runes[i])
					i++
				}
			}
			if i < len(runes) {
				result.WriteRune(runes[i]) // closing "
				i++
			}
			continue
		}

		// Handle square bracket character classes [...]
		if c == '[' {
			result.WriteRune(c)
			i++
			// copy until matching ]
			depth := 1
			for i < len(runes) && depth > 0 {
				if runes[i] == '[' {
					depth++
				} else if runes[i] == ']' {
					depth--
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Handle parentheses - pass through
		if c == '(' || c == ')' {
			result.WriteRune(c)
			i++
			continue
		}

		// Handle operators - pass through
		if c == '|' || c == '*' || c == '+' || c == '?' || c == '.' || c == '_' {
			result.WriteRune(c)
			i++
			continue
		}

		// Handle identifiers (potential macro references)
		if isIdentStart(c) {
			// Collect the full identifier
			start := i
			for i < len(runes) && isIdentContinue(runes[i]) {
				i++
			}
			ident := string(runes[start:i])

			// Try to resolve via the resolver (which handles recursive
			// expansion and cycle detection). If the resolver returns an
			// "undefined macro" error, the identifier is not a macro — write
			// it as a literal. Any other error is propagated.
			resolved, err := resolver(ident)
			if err == nil {
				result.WriteRune('(')
				result.WriteString(resolved)
				result.WriteRune(')')
			} else if strings.Contains(err.Error(), "undefined macro") {
				// Not a macro - write as-is (could be a literal identifier)
				result.WriteString(ident)
			} else {
				return "", err
			}
			continue
		}

		// Everything else - pass through
		result.WriteRune(c)
		i++
	}

	return strings.TrimSpace(result.String()), nil
}

func isIdentStart(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func isIdentContinue(c rune) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

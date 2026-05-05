// Package lexbuild compila especificaciones YALex en DFAs reutilizables.
package lexbuild

import (
	"fmt"

	"genanalex/internal/dfa"
	"genanalex/internal/lexer"
	"genanalex/internal/regex"
	"genanalex/internal/shared"
	"genanalex/internal/yalex"
)

// Result agrupa los artefactos léxicos reutilizables por distintas CLIs.
type Result struct {
	DFAEntries  []lexer.DFAEntry
	DOTContents []string
	Rules       []yalex.TokenRule
	Macros      map[string]string
}

// CompileYALFile parsea un archivo .yal y construye los DFAs correspondientes.
func CompileYALFile(path string, captureDOT bool) (*Result, error) {
	parseResult, err := yalex.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse yal file: %w", err)
	}
	return CompileParseResult(parseResult, captureDOT)
}

// CompileParseResult expande reglas y construye DFAs listos para tokenización.
func CompileParseResult(parseResult *yalex.ParseResult, captureDOT bool) (*Result, error) {
	if parseResult == nil {
		return nil, fmt.Errorf("compile yal: nil parse result")
	}

	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		return nil, fmt.Errorf("expand macros: %w", err)
	}

	result := &Result{
		DFAEntries: make([]lexer.DFAEntry, 0, len(expandedRules)),
		Rules:      append([]yalex.TokenRule(nil), expandedRules...),
		Macros:     cloneMacros(parseResult.Macros),
	}

	for _, rule := range expandedRules {
		normalized, err := regex.Normalize(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("normalize %q: %w", rule.Pattern, err)
		}

		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			return nil, fmt.Errorf("build postfix for %q: %w", rule.Pattern, err)
		}

		root, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			return nil, fmt.Errorf("build syntax tree for %q: %w", rule.Pattern, err)
		}

		if captureDOT {
			result.DOTContents = append(result.DOTContents, fmt.Sprintf("// Rule %d: %s\n%s", rule.Priority, rule.Action, dfa.ToDOT(root)))
		}

		builtDFA := dfa.BuildDFA(root, posToSymbol, rule.Action)
		minimizedDFA := dfa.Minimize(builtDFA)

		result.DFAEntries = append(result.DFAEntries, lexer.DFAEntry{
			DFA:       minimizedDFA,
			TokenName: rule.Action,
			Priority:  rule.Priority,
		})
	}

	return result, nil
}

// TokenizeFile lee un archivo fuente y lo tokeniza con los DFAs construidos.
func TokenizeFile(entries []lexer.DFAEntry, path string) ([]shared.Token, []string, error) {
	src, err := lexer.ReadSource(path)
	if err != nil {
		return nil, nil, err
	}
	tokens, tokenErrors := lexer.Tokenize(entries, src)
	return tokens, tokenErrors, nil
}

func cloneMacros(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

package yapar

import (
	"fmt"
	"sort"
	"strings"

	"genanalex/internal/shared"
)

type Method string

const (
	MethodLR0  Method = "lr0"
	MethodLR1  Method = "lr1"
	MethodSLR  Method = "slr"
	MethodLL1  Method = "ll1"
	MethodLALR Method = "lalr"
)

var validMethods = []Method{MethodLL1, MethodLR0, MethodSLR, MethodLR1, MethodLALR}

func ValidMethods() []Method {
	return append([]Method(nil), validMethods...)
}

type ExecutableParser interface {
	Parse(tokens []shared.Token) (*ParseResult, error)
	Table() TableView
}

type TableView interface {
	ActionAt(state int, symbol string) (ActionKind, int, bool)
	GotoAt(state int, symbol string) (int, bool)
	States() []int
	Terminals() []string
	NonTerminals() []string
}

func ParseMethod(raw string) (Method, error) {
	method := Method(strings.ToLower(strings.TrimSpace(raw)))
	for _, candidate := range validMethods {
		if method == candidate {
			return method, nil
		}
	}
	return "", fmt.Errorf("invalid parser method %q: valid options are %s", raw, strings.Join(methodNames(), ", "))
}

func BuildParser(g *Grammar, ff *FirstFollow, method Method) (ExecutableParser, error) {
	switch method {
	case MethodSLR:
		return buildSLRParser(g, ff)
	case MethodLL1:
		return buildLL1Parser(g, ff)
	case MethodLR0, MethodLR1:
		return nil, fmt.Errorf("parser method %q not implemented yet: valid options are %s", method, strings.Join(methodNames(), ", "))
	case MethodLALR:
		return buildLALRParser(g, ff)
	default:
		return nil, fmt.Errorf("parser method %q not implemented yet: valid options are %s", method, strings.Join(methodNames(), ", "))
	}
}

func methodNames() []string {
	result := make([]string, len(validMethods))
	for i, method := range validMethods {
		result[i] = string(method)
	}
	return result
}

func sortedStateIDsFromMaps(actionTable map[int]map[string]Action, gotoTable map[int]map[string]int) []int {
	stateSet := make(map[int]bool)
	for state := range actionTable {
		stateSet[state] = true
	}
	for state := range gotoTable {
		stateSet[state] = true
	}
	result := make([]int, 0, len(stateSet))
	for state := range stateSet {
		result = append(result, state)
	}
	sort.Ints(result)
	return result
}

func sortedGrammarTerminals(g *Grammar) []string {
	if g == nil {
		return nil
	}
	result := make([]string, 0, len(g.Terminals))
	for symbol := range g.Terminals {
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

func sortedGrammarNonTerminals(g *Grammar) []string {
	if g == nil {
		return nil
	}
	result := make([]string, 0, len(g.NonTerminals))
	for symbol := range g.NonTerminals {
		if symbol == g.Augmented {
			continue
		}
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

package yapar

import (
	"fmt"
	"sort"

	"genanalex/internal/shared"
)

type LL1Table struct {
	M            map[string]map[string]int
	ntIndex      map[string]int
	terminals    []string
	nonTerminals []string
}

func BuildLL1Table(g *Grammar, ff *FirstFollow) (*LL1Table, error) {
	table := &LL1Table{M: make(map[string]map[string]int)}
	if g == nil {
		return table, nil
	}
	if ff == nil {
		return nil, fmt.Errorf("yapar: first/follow data is required to build LL1 table")
	}
	if err := checkLeftRecursion(g); err != nil {
		return nil, err
	}

	table.nonTerminals = sortedGrammarNonTerminals(g)
	table.terminals = sortedGrammarTerminals(g)
	table.ntIndex = make(map[string]int, len(table.nonTerminals))
	for i, nonTerminal := range table.nonTerminals {
		table.ntIndex[nonTerminal] = i
		table.M[nonTerminal] = make(map[string]int)
	}

	for _, production := range g.Productions {
		if production.ID == 0 || production.Head == g.Augmented {
			continue
		}
		first := FirstOfSequence(production.Body, ff)
		for _, terminal := range withoutEpsilon(first).Sorted() {
			if err := table.set(production.Head, terminal, production.ID); err != nil {
				return nil, err
			}
		}
		if first.Has(Epsilon) {
			for _, terminal := range ff.Follow[production.Head].Sorted() {
				if err := table.set(production.Head, terminal, production.ID); err != nil {
					return nil, err
				}
			}
		}
	}

	return table, nil
}

func (t *LL1Table) set(nonTerminal, terminal string, productionID int) error {
	if t.M[nonTerminal] == nil {
		t.M[nonTerminal] = make(map[string]int)
	}
	if existing, exists := t.M[nonTerminal][terminal]; exists {
		if existing == productionID {
			return nil
		}
		return &LL1ConflictError{
			NonTerminal:  nonTerminal,
			Terminal:     terminal,
			ExistingProd: existing,
			ConflictProd: productionID,
		}
	}
	t.M[nonTerminal][terminal] = productionID
	return nil
}

func checkLeftRecursion(g *Grammar) error {
	if g == nil {
		return nil
	}
	for _, production := range g.Productions {
		if len(production.Body) == 0 {
			continue
		}
		first := production.Body[0]
		if !first.Terminal && first.Name == production.Head {
			return &LeftRecursionError{NonTerminal: production.Head, ProductionID: production.ID}
		}
	}
	return nil
}

type ll1Parser struct {
	grammar *Grammar
	table   *LL1Table
}

func buildLL1Parser(g *Grammar, ff *FirstFollow) (ExecutableParser, error) {
	table, err := BuildLL1Table(g, ff)
	if err != nil {
		return nil, err
	}
	return &ll1Parser{grammar: g, table: table}, nil
}

func (p *ll1Parser) Parse(tokens []shared.Token) (*ParseResult, error) {
	if p == nil || p.grammar == nil {
		return &ParseResult{Accepted: false}, fmt.Errorf("yapar: grammar is required")
	}
	if p.table == nil {
		return &ParseResult{Accepted: false}, fmt.Errorf("yapar: ll1 table is required")
	}

	filtered := FilterIgnoredTokens(tokens, p.grammar.IgnoreSet)
	stream := append([]shared.Token{}, filtered...)
	stream = append(stream, shared.Token{Type: EndMarker, Line: endOfInputLine(tokens)})

	stack := []string{EndMarker, p.grammar.Start}
	index := 0

	for len(stack) > 0 && index < len(stream) {
		top := stack[len(stack)-1]
		lookahead := stream[index]

		if top == EndMarker {
			if lookahead.Type == EndMarker {
				return &ParseResult{Accepted: true}, nil
			}
			return &ParseResult{Accepted: false}, &SyntaxError{
				Line:     syntaxErrorLine(lookahead, tokens),
				GotType:  lookahead.Type,
				Lexeme:   lookahead.Lexeme,
				Expected: []string{EndMarker},
			}
		}

		if p.grammar.IsTerminal(top) {
			if top == lookahead.Type {
				stack = stack[:len(stack)-1]
				index++
				continue
			}
			return &ParseResult{Accepted: false}, &SyntaxError{
				Line:     syntaxErrorLine(lookahead, tokens),
				GotType:  lookahead.Type,
				Lexeme:   lookahead.Lexeme,
				Expected: []string{top},
			}
		}

		prodID, ok := p.lookupProduction(top, lookahead.Type)
		if !ok {
			return &ParseResult{Accepted: false}, &SyntaxError{
				Line:     syntaxErrorLine(lookahead, tokens),
				GotType:  lookahead.Type,
				Lexeme:   lookahead.Lexeme,
				Expected: p.expectedLookaheads(top),
			}
		}

		production, ok := productionByID(p.grammar, prodID)
		if !ok {
			return &ParseResult{Accepted: false}, fmt.Errorf("yapar: unknown production id %d", prodID)
		}

		stack = stack[:len(stack)-1]
		for i := len(production.Body) - 1; i >= 0; i-- {
			stack = append(stack, production.Body[i].Name)
		}
	}

	return &ParseResult{Accepted: false}, &SyntaxError{
		Line:     endOfInputLine(tokens),
		GotType:  EndMarker,
		Expected: []string{EndMarker},
	}
}

func (p *ll1Parser) lookupProduction(nonTerminal, terminal string) (int, bool) {
	if p == nil || p.table == nil || p.table.M == nil {
		return 0, false
	}
	row := p.table.M[nonTerminal]
	if row == nil {
		return 0, false
	}
	prodID, ok := row[terminal]
	return prodID, ok
}

func (p *ll1Parser) expectedLookaheads(nonTerminal string) []string {
	if p == nil || p.table == nil {
		return nil
	}
	row := p.table.M[nonTerminal]
	result := make([]string, 0, len(row))
	for terminal := range row {
		result = append(result, terminal)
	}
	sort.Strings(result)
	return result
}

func (p *ll1Parser) Table() TableView {
	return &ll1TableView{grammar: p.grammar, table: p.table}
}

type ll1TableView struct {
	grammar *Grammar
	table   *LL1Table
}

func (v *ll1TableView) ActionAt(state int, symbol string) (ActionKind, int, bool) {
	nonTerminal, ok := v.nonTerminalForState(state)
	if !ok {
		return ActionError, 0, false
	}
	prodID, ok := v.table.M[nonTerminal][symbol]
	if !ok {
		return ActionError, 0, false
	}
	return ActionReduce, prodID, true
}

func (v *ll1TableView) GotoAt(state int, symbol string) (int, bool) {
	return 0, false
}

func (v *ll1TableView) States() []int {
	if v == nil || v.table == nil {
		return nil
	}
	states := make([]int, 0, len(v.table.ntIndex))
	for _, state := range v.table.ntIndex {
		states = append(states, state)
	}
	sort.Ints(states)
	return states
}

func (v *ll1TableView) Terminals() []string {
	if v == nil || v.table == nil {
		return nil
	}
	return append([]string(nil), v.table.terminals...)
}

func (v *ll1TableView) NonTerminals() []string {
	if v == nil || v.table == nil {
		return nil
	}
	return append([]string(nil), v.table.nonTerminals...)
}

func (v *ll1TableView) nonTerminalForState(state int) (string, bool) {
	if v == nil || v.table == nil {
		return "", false
	}
	for nonTerminal, index := range v.table.ntIndex {
		if index == state {
			return nonTerminal, true
		}
	}
	return "", false
}

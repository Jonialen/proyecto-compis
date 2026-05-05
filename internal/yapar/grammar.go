// Package yapar define el modelo formal mínimo de gramáticas para YAPar.
package yapar

import "fmt"

const (
	// EndMarker representa el fin de entrada reservado para el parser.
	EndMarker = "$"
	// Epsilon representa la cadena vacía a nivel semántico.
	Epsilon = "ε"
)

// Symbol representa un símbolo terminal o no terminal de la gramática.
type Symbol struct {
	Name     string
	Terminal bool
}

// Production representa una producción formal con identificador estable.
type Production struct {
	ID   int
	Head string
	Body []Symbol
}

// IsEpsilon reporta si la producción deriva la cadena vacía.
func (p Production) IsEpsilon() bool {
	return len(p.Body) == 0
}

// Grammar representa la gramática formal y aumentada consumida por el pipeline LR.
type Grammar struct {
	Start        string
	Augmented    string
	Terminals    map[string]bool
	NonTerminals map[string]bool
	Productions  []Production
	IgnoreSet    map[string]bool
}

// BuildGrammar convierte una especificación cruda en una gramática consistente.
func BuildGrammar(spec *YaparSpec) (*Grammar, error) {
	if spec == nil {
		return nil, &SpecError{Message: "nil spec"}
	}
	if len(spec.Productions) == 0 {
		return nil, &SpecError{Message: "no productions declared"}
	}
	if spec.StartSymbol == "" {
		return nil, &SpecError{Message: "missing start symbol"}
	}

	nonTerminals, err := collectNonTerminals(spec)
	if err != nil {
		return nil, err
	}
	if !nonTerminals[spec.StartSymbol] {
		return nil, &SpecError{Message: fmt.Sprintf("start symbol %q does not have a production", spec.StartSymbol)}
	}

	terminals, err := collectTerminals(spec, nonTerminals)
	if err != nil {
		return nil, err
	}
	ignoreSet, err := validateIgnoreTokens(spec, terminals)
	if err != nil {
		return nil, err
	}
	reserved := collectReservedNames(terminals, nonTerminals)

	g := &Grammar{
		Start:        spec.StartSymbol,
		Augmented:    augmentedName(spec.StartSymbol, reserved),
		Terminals:    cloneBoolMap(terminals),
		NonTerminals: cloneBoolMap(nonTerminals),
		IgnoreSet:    ignoreSet,
	}
	g.Terminals[EndMarker] = true

	g.Productions = append(g.Productions, Production{
		ID:   0,
		Head: g.Augmented,
		Body: []Symbol{{Name: g.Start, Terminal: false}},
	})

	nextID := 1
	for _, raw := range spec.Productions {
		if !g.NonTerminals[raw.Head] {
			return nil, &SpecError{Message: fmt.Sprintf("production head %q is not a declared non-terminal", raw.Head)}
		}
		if len(raw.Bodies) == 0 {
			return nil, &SpecError{Message: fmt.Sprintf("production %q must declare at least one alternative", raw.Head)}
		}
		for _, body := range raw.Bodies {
			production := Production{ID: nextID, Head: raw.Head}
			if len(body) == 0 {
				g.Productions = append(g.Productions, production)
				nextID++
				continue
			}
			for _, name := range body {
				terminal, err := resolveSymbol(name, g.Terminals, g.NonTerminals, g.IgnoreSet)
				if err != nil {
					return nil, err
				}
				production.Body = append(production.Body, Symbol{Name: name, Terminal: terminal})
			}
			g.Productions = append(g.Productions, production)
			nextID++
		}
	}

	return g, nil
}

// ProductionsFor devuelve todas las producciones asociadas a una cabeza dada.
func (g *Grammar) ProductionsFor(head string) []Production {
	if g == nil {
		return nil
	}
	result := make([]Production, 0)
	for _, production := range g.Productions {
		if production.Head == head {
			result = append(result, production)
		}
	}
	return result
}

// IsTerminal indica si el nombre corresponde a un terminal conocido.
func (g *Grammar) IsTerminal(name string) bool {
	if g == nil {
		return false
	}
	return g.Terminals[name]
}

// IsNonTerminal indica si el nombre corresponde a un no terminal conocido.
func (g *Grammar) IsNonTerminal(name string) bool {
	if g == nil {
		return false
	}
	return g.NonTerminals[name] || name == g.Augmented
}

func collectNonTerminals(spec *YaparSpec) (map[string]bool, error) {
	headSet := make(map[string]bool)
	if spec == nil {
		return headSet, nil
	}
	for _, production := range spec.Productions {
		if production.Head == "" {
			return nil, &SpecError{Message: "production head cannot be empty"}
		}
		if isReservedGrammarName(production.Head) {
			return nil, &SpecError{Message: fmt.Sprintf("symbol %q is reserved by the grammar model", production.Head)}
		}
		headSet[production.Head] = true
	}
	return headSet, nil
}

func augmentedName(start string, reserved map[string]bool) string {
	name := start + "'"
	for reserved[name] {
		name += "'"
	}
	return name
}

func resolveSymbol(name string, terminals, nonTerminals, ignoreSet map[string]bool) (bool, error) {
	switch {
	case name == "":
		return false, &SpecError{Message: "empty symbol in production body"}
	case name == Epsilon:
		return false, &SpecError{Message: "explicit epsilon symbol is not allowed; use an empty alternative instead"}
	case name == EndMarker:
		return false, &SpecError{Message: fmt.Sprintf("symbol %q is reserved for end of input", EndMarker)}
	case terminals[name]:
		if ignoreSet[name] {
			return false, &SpecError{Message: fmt.Sprintf("ignored token %q cannot appear in grammar productions", name)}
		}
		return true, nil
	case nonTerminals[name]:
		return false, nil
	default:
		return false, &SpecError{Message: fmt.Sprintf("symbol %q is neither token nor production head", name)}
	}
}

func collectTerminals(spec *YaparSpec, nonTerminals map[string]bool) (map[string]bool, error) {
	terminals := make(map[string]bool, len(spec.Tokens))
	for _, tok := range spec.Tokens {
		if tok == "" {
			return nil, &SpecError{Message: "token name cannot be empty"}
		}
		if isReservedGrammarName(tok) {
			return nil, &SpecError{Message: fmt.Sprintf("symbol %q is reserved by the grammar model", tok)}
		}
		if nonTerminals[tok] {
			return nil, &SpecError{Message: fmt.Sprintf("symbol %q cannot be both token and non-terminal", tok)}
		}
		if terminals[tok] {
			return nil, &SpecError{Message: fmt.Sprintf("token %q declared more than once", tok)}
		}
		terminals[tok] = true
	}
	return terminals, nil
}

func validateIgnoreTokens(spec *YaparSpec, terminals map[string]bool) (map[string]bool, error) {
	ignoreSet := cloneBoolMap(spec.IgnoreTokens)
	for tok := range ignoreSet {
		if !terminals[tok] {
			return nil, &SpecError{Message: fmt.Sprintf("ignored token %q must be declared with %%token", tok)}
		}
	}
	return ignoreSet, nil
}

func collectReservedNames(terminals, nonTerminals map[string]bool) map[string]bool {
	reserved := make(map[string]bool, len(terminals)+len(nonTerminals)+2)
	for name := range terminals {
		reserved[name] = true
	}
	for name := range nonTerminals {
		reserved[name] = true
	}
	reserved[EndMarker] = true
	reserved[Epsilon] = true
	return reserved
}

func isReservedGrammarName(name string) bool {
	return name == EndMarker || name == Epsilon
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return map[string]bool{}
	}
	dst := make(map[string]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

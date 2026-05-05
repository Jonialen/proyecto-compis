// Package yapar define el modelo formal mínimo de gramáticas para YAPar.
package yapar

import "fmt"

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
	if spec.StartSymbol == "" {
		return nil, &SpecError{Message: "missing start symbol"}
	}

	g := &Grammar{
		Start:        spec.StartSymbol,
		Augmented:    augmentedName(spec.StartSymbol, collectHeads(spec)),
		Terminals:    make(map[string]bool),
		NonTerminals: collectHeads(spec),
		IgnoreSet:    cloneBoolMap(spec.IgnoreTokens),
	}

	for _, tok := range spec.Tokens {
		g.Terminals[tok] = true
	}
	g.Terminals["$"] = true

	g.Productions = append(g.Productions, Production{
		ID:   0,
		Head: g.Augmented,
		Body: []Symbol{{Name: g.Start, Terminal: false}},
	})

	nextID := 1
	for _, raw := range spec.Productions {
		for _, body := range raw.Bodies {
			production := Production{ID: nextID, Head: raw.Head}
			for _, name := range body {
				terminal, err := resolveTerminal(name, g.Terminals, g.NonTerminals)
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

func collectHeads(spec *YaparSpec) map[string]bool {
	headSet := make(map[string]bool)
	if spec == nil {
		return headSet
	}
	for _, production := range spec.Productions {
		headSet[production.Head] = true
	}
	return headSet
}

func augmentedName(start string, reserved map[string]bool) string {
	name := start + "'"
	for reserved[name] {
		name += "'"
	}
	return name
}

func resolveTerminal(name string, terminals, nonTerminals map[string]bool) (bool, error) {
	switch {
	case terminals[name]:
		return true, nil
	case nonTerminals[name]:
		return false, nil
	case name == "":
		return false, &SpecError{Message: "empty symbol in production body"}
	default:
		return false, &SpecError{Message: fmt.Sprintf("symbol %q is neither token nor production head", name)}
	}
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

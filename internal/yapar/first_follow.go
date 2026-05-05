// Package yapar contiene los helpers base para FIRST, FOLLOW y nullable.
package yapar

import "sort"

// Set es la representación mínima reutilizable para conjuntos de símbolos.
type Set map[string]bool

// FirstFollow agrupa los resultados del cálculo de FIRST/FOLLOW/nullable.
type FirstFollow struct {
	First    map[string]Set
	Follow   map[string]Set
	Nullable map[string]bool
}

// NewSet crea un conjunto inicial con elementos opcionales.
func NewSet(items ...string) Set {
	set := make(Set, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// Add inserta un valor y reporta si el conjunto cambió.
func (s Set) Add(v string) bool {
	if s[v] {
		return false
	}
	s[v] = true
	return true
}

// Has reporta si el valor existe en el conjunto.
func (s Set) Has(v string) bool {
	return s[v]
}

// Merge une otro conjunto y reporta si hubo cambios.
func (s Set) Merge(other Set) bool {
	changed := false
	for item := range other {
		if s.Add(item) {
			changed = true
		}
	}
	return changed
}

// Clone crea una copia superficial segura del conjunto.
func (s Set) Clone() Set {
	clone := make(Set, len(s))
	for item := range s {
		clone[item] = true
	}
	return clone
}

// Sorted devuelve una vista ordenada útil para errores y snapshots.
func (s Set) Sorted() []string {
	items := make([]string, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

// ComputeFirstFollow calcula nullable, FIRST y FOLLOW para una gramática formal.
func ComputeFirstFollow(g *Grammar) (*FirstFollow, error) {
	ff := &FirstFollow{
		First:    make(map[string]Set),
		Follow:   make(map[string]Set),
		Nullable: make(map[string]bool),
	}
	if g == nil {
		return ff, nil
	}

	for terminal := range g.Terminals {
		ff.First[terminal] = NewSet(terminal)
	}
	for _, nonTerminal := range grammarNonTerminals(g) {
		ff.First[nonTerminal] = NewSet()
		ff.Follow[nonTerminal] = NewSet()
	}
	if g.Augmented != "" {
		ff.Follow[g.Augmented].Add(EndMarker)
	} else if g.Start != "" {
		ff.Follow[g.Start].Add(EndMarker)
	}

	computeNullable(g, ff)
	computeFirst(g, ff)
	computeFollow(g, ff)

	return ff, nil
}

// FirstOfSequence devuelve FIRST para una secuencia ya tipada.
func FirstOfSequence(seq []Symbol, ff *FirstFollow) Set {
	result := NewSet()
	if len(seq) == 0 {
		result.Add(Epsilon)
		return result
	}

	for _, symbol := range seq {
		if symbol.Terminal {
			result.Add(symbol.Name)
			return result
		}
		if ff == nil {
			return result
		}
		current := ff.First[symbol.Name]
		for token := range current {
			if token != Epsilon {
				result.Add(token)
			}
		}
		if !ff.Nullable[symbol.Name] {
			return result
		}
	}
	result.Add(Epsilon)
	return result
}

func grammarNonTerminals(g *Grammar) []string {
	set := make(map[string]bool, len(g.NonTerminals)+1)
	for nonTerminal := range g.NonTerminals {
		set[nonTerminal] = true
	}
	if g.Augmented != "" {
		set[g.Augmented] = true
	}
	result := make([]string, 0, len(set))
	for nonTerminal := range set {
		result = append(result, nonTerminal)
	}
	sort.Strings(result)
	return result
}

func computeNullable(g *Grammar, ff *FirstFollow) {
	changed := true
	for changed {
		changed = false
		for _, production := range g.Productions {
			if ff.Nullable[production.Head] {
				continue
			}
			if productionIsNullable(production, ff.Nullable) {
				ff.Nullable[production.Head] = true
				changed = true
			}
		}
	}
}

func productionIsNullable(production Production, nullable map[string]bool) bool {
	if production.IsEpsilon() {
		return true
	}
	for _, symbol := range production.Body {
		if symbol.Terminal || !nullable[symbol.Name] {
			return false
		}
	}
	return true
}

func computeFirst(g *Grammar, ff *FirstFollow) {
	changed := true
	for changed {
		changed = false
		for _, production := range g.Productions {
			if ff.First[production.Head].Merge(FirstOfSequence(production.Body, ff)) {
				changed = true
			}
		}
	}
}

func computeFollow(g *Grammar, ff *FirstFollow) {
	changed := true
	for changed {
		changed = false
		for _, production := range g.Productions {
			for i, symbol := range production.Body {
				if symbol.Terminal {
					continue
				}

				target := ff.Follow[symbol.Name]
				beta := production.Body[i+1:]
				firstBeta := FirstOfSequence(beta, ff)
				if target.Merge(withoutEpsilon(firstBeta)) {
					changed = true
				}
				if firstBeta.Has(Epsilon) && target.Merge(ff.Follow[production.Head]) {
					changed = true
				}
			}
		}
	}
}

func withoutEpsilon(set Set) Set {
	result := NewSet()
	for item := range set {
		if item != Epsilon {
			result.Add(item)
		}
	}
	return result
}

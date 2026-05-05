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

// ComputeFirstFollow reserva la estructura base; el algoritmo completo queda pendiente.
func ComputeFirstFollow(g *Grammar) (*FirstFollow, error) {
	ff := &FirstFollow{
		First:    make(map[string]Set),
		Follow:   make(map[string]Set),
		Nullable: make(map[string]bool),
	}
	if g != nil {
		for terminal := range g.Terminals {
			ff.First[terminal] = NewSet(terminal)
		}
		for nonTerminal := range g.NonTerminals {
			ff.First[nonTerminal] = NewSet()
			ff.Follow[nonTerminal] = NewSet()
		}
		if g.Start != "" {
			ff.Follow[g.Start] = NewSet(EndMarker)
		}
	}
	return ff, ErrNotImplemented
}

// FirstOfSequence devuelve un cálculo defensivo mínimo para secuencias ya tipadas.
func FirstOfSequence(seq []Symbol, ff *FirstFollow) Set {
	result := NewSet()
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
	if len(seq) == 0 {
		result.Add(Epsilon)
	}
	return result
}

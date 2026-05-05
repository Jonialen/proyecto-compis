// Package yapar define la base de items LR y estados canónicos.
package yapar

import (
	"sort"
	"strconv"
	"strings"
)

// Item representa una producción con la posición actual del punto LR.
type Item struct {
	ProductionID int
	Dot          int
}

// State representa un estado de la colección canónica LR.
type State struct {
	ID    int
	Items []Item
}

// Closure calcula el cierre LR(0) de un conjunto inicial de items.
func Closure(g *Grammar, items []Item) []Item {
	if len(items) == 0 {
		return nil
	}

	closure := make([]Item, 0, len(items))
	queue := make([]Item, 0, len(items))
	seen := make(map[Item]bool, len(items))

	push := func(item Item) {
		if seen[item] {
			return
		}
		seen[item] = true
		queue = append(queue, item)
	}

	for _, item := range items {
		push(item)
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		closure = append(closure, item)

		production, ok := productionByID(g, item.ProductionID)
		if !ok || item.Dot < 0 || item.Dot >= len(production.Body) {
			continue
		}

		symbol := production.Body[item.Dot]
		if symbol.Terminal || !g.IsNonTerminal(symbol.Name) {
			continue
		}

		for _, production := range g.ProductionsFor(symbol.Name) {
			push(Item{ProductionID: production.ID, Dot: 0})
		}
	}

	sortItems(closure)
	return closure
}

// Goto desplaza el punto LR(0) sobre un símbolo y devuelve el cierre resultante.
func Goto(g *Grammar, items []Item, symbol string) []Item {
	if len(items) == 0 || symbol == "" {
		return nil
	}

	moved := make([]Item, 0, len(items))
	for _, item := range items {
		production, ok := productionByID(g, item.ProductionID)
		if !ok || item.Dot < 0 || item.Dot >= len(production.Body) {
			continue
		}
		if production.Body[item.Dot].Name != symbol {
			continue
		}
		moved = append(moved, Item{ProductionID: item.ProductionID, Dot: item.Dot + 1})
	}

	if len(moved) == 0 {
		return nil
	}

	return Closure(g, moved)
}

// BuildCanonicalCollection construye la colección canónica LR(0) y sus transiciones.
func BuildCanonicalCollection(g *Grammar) ([]State, map[int]map[string]int, error) {
	transitions := make(map[int]map[string]int)
	if g == nil || len(g.Productions) == 0 {
		return nil, transitions, nil
	}

	initial := Closure(g, []Item{{ProductionID: 0, Dot: 0}})
	if len(initial) == 0 {
		return nil, transitions, nil
	}

	states := []State{{ID: 0, Items: initial}}
	known := map[string]int{stateSignature(initial): 0}

	for idx := 0; idx < len(states); idx++ {
		state := states[idx]
		for _, symbol := range symbolsAfterDot(g, state.Items) {
			nextItems := Goto(g, state.Items, symbol)
			if len(nextItems) == 0 {
				continue
			}

			signature := stateSignature(nextItems)
			nextID, exists := known[signature]
			if !exists {
				nextID = len(states)
				known[signature] = nextID
				states = append(states, State{ID: nextID, Items: nextItems})
			}

			if transitions[state.ID] == nil {
				transitions[state.ID] = make(map[string]int)
			}
			transitions[state.ID][symbol] = nextID
		}
	}

	return states, transitions, nil
}

func productionByID(g *Grammar, id int) (Production, bool) {
	if g == nil {
		return Production{}, false
	}
	for _, production := range g.Productions {
		if production.ID == id {
			return production, true
		}
	}
	return Production{}, false
}

func symbolsAfterDot(g *Grammar, items []Item) []string {
	if len(items) == 0 {
		return nil
	}

	set := make(map[string]bool)
	for _, item := range items {
		production, ok := productionByID(g, item.ProductionID)
		if !ok || item.Dot < 0 || item.Dot >= len(production.Body) {
			continue
		}
		set[production.Body[item.Dot].Name] = true
	}

	result := make([]string, 0, len(set))
	for symbol := range set {
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

func stateSignature(items []Item) string {
	if len(items) == 0 {
		return ""
	}

	normalized := append([]Item(nil), items...)
	sortItems(normalized)

	parts := make([]string, len(normalized))
	for i, item := range normalized {
		parts[i] = itemKey(item)
	}
	return strings.Join(parts, "|")
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ProductionID != items[j].ProductionID {
			return items[i].ProductionID < items[j].ProductionID
		}
		return items[i].Dot < items[j].Dot
	})
}

func itemKey(item Item) string {
	return strings.Join([]string{strconv.Itoa(item.ProductionID), strconv.Itoa(item.Dot)}, ":")
}

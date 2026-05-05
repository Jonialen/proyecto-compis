// Package yapar define la base de items LR y estados canónicos.
package yapar

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

// Closure reserva la firma del cálculo de cierre LR(0).
func Closure(g *Grammar, items []Item) []Item {
	cloned := make([]Item, len(items))
	copy(cloned, items)
	return cloned
}

// Goto reserva la firma del desplazamiento LR(0) sobre un símbolo.
func Goto(g *Grammar, items []Item, symbol string) []Item {
	_ = g
	_ = symbol
	return nil
}

// BuildCanonicalCollection deja explícita la frontera pendiente del autómata LR(0).
func BuildCanonicalCollection(g *Grammar) ([]State, map[int]map[string]int, error) {
	_ = g
	return nil, map[int]map[string]int{}, ErrNotImplemented
}

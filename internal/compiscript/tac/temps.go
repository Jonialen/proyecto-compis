package tac

import (
	"fmt"
	"sort"

	"genanalex/internal/compiscript/model"
)

// temporaryPool recycles only values whose final consumer has been emitted.
// A reused slot can have a different type because every operand carries its type.
type temporaryPool struct {
	next, peak int
	free       []int
	live       map[string]int
}

func (p *temporaryPool) acquire(kind model.TypeKind) Operand {
	if p.live == nil {
		p.live = make(map[string]int)
	}
	id := p.next
	if len(p.free) > 0 {
		id, p.free = p.free[0], p.free[1:]
	} else {
		p.next++
	}
	name := fmt.Sprintf("t%d", id)
	p.live[name] = id
	if len(p.live) > p.peak {
		p.peak = len(p.live)
	}
	return Operand{Kind: Temporary, Value: name, Type: kind}
}

func (p *temporaryPool) release(value Operand) {
	if value.Kind != Temporary {
		return
	}
	if id, ok := p.live[value.Value]; ok {
		delete(p.live, value.Value)
		p.free = append(p.free, id)
		sort.Ints(p.free)
	}
}

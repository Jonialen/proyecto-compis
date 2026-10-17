package tac

import (
	"testing"

	"genanalex/internal/compiscript/model"
)

func TestTemporaryLifetimes(t *testing.T) {
	var pool temporaryPool
	first := pool.acquire(model.TypeInteger)
	second := pool.acquire(model.TypeFloat)
	if first.Value != "t0" || second.Value != "t1" || pool.peak != 2 {
		t.Fatalf("temporaries = %#v, %#v; peak = %d", first, second, pool.peak)
	}
	pool.release(first)
	reused := pool.acquire(model.TypeFloat)
	if reused.Value != first.Value || reused.Type != model.TypeFloat || reused.Kind != Temporary {
		t.Fatalf("reused = %#v, want t0 with the new value's type", reused)
	}
	pool.release(reused)
	pool.release(second)
	for i := 0; i < 10; i++ {
		value := pool.acquire(model.TypeInteger)
		if value.Value != "t0" || pool.peak != 2 {
			t.Fatalf("iteration %d: value = %#v; peak = %d", i, value, pool.peak)
		}
		pool.release(value)
	}
	if len(pool.live) != 0 {
		t.Fatalf("unconsumed temporaries = %#v", pool.live)
	}
}

func TestTemporaryReleaseDoesNotFreeSymbolsOrDuplicateSlots(t *testing.T) {
	var pool temporaryPool
	value := pool.acquire(model.TypeInteger)
	pool.release(Operand{Kind: Symbol, Value: value.Value, Type: model.TypeInteger})
	if next := pool.acquire(model.TypeInteger); next.Value != "t1" {
		t.Fatalf("live slot was reused: %#v", next)
	}
	pool.release(value)
	pool.release(value)
	first, second := pool.acquire(model.TypeInteger), pool.acquire(model.TypeInteger)
	if first.Value != "t0" || second.Value != "t2" {
		t.Fatalf("duplicate release created overlapping slots: %#v, %#v", first, second)
	}
}

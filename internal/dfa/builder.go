package dfa

import "genanalex/internal/regex"

// DFA represents a deterministic finite automaton.
type DFA struct {
	States      map[int]map[int]bool // state id → set of positions
	Transitions map[int]map[rune]int // state id → (symbol → state id)
	Start       int
	Accepting   map[int]bool
	StateToken  map[int]string // state id → token name (for accepting states)
}

// BuildDFA constructs a DFA directly from the syntax tree using the
// direct DFA construction algorithm (Dragon Book, Section 3.9.5).
//
// Parameters:
//   - root: the syntax tree root (should be augmented with #)
//   - posToSymbol: maps position number → character symbol
//   - tokenName: the name of the token this DFA recognizes
func BuildDFA(root *Node, posToSymbol map[int]rune, tokenName string) *DFA {
	followPos := ComputeFollowPos(root)

	// Find the end-marker position
	endPos := -1
	for pos, sym := range posToSymbol {
		if sym == regex.EndMarker {
			endPos = pos
		}
	}

	dfa := &DFA{
		States:      make(map[int]map[int]bool),
		Transitions: make(map[int]map[rune]int),
		Start:       0,
		Accepting:   make(map[int]bool),
		StateToken:  make(map[int]string),
	}

	// Initial state = firstpos(root)
	initial := FirstPos(root)
	stateID := 0
	keyToID := make(map[string]int)

	initialKey := setKey(initial)
	keyToID[initialKey] = stateID
	dfa.States[stateID] = initial

	if initial[endPos] {
		dfa.Accepting[stateID] = true
		dfa.StateToken[stateID] = tokenName
	}

	// WorkList
	var worklist []int
	worklist = append(worklist, stateID)
	stateID++

	for len(worklist) > 0 {
		curr := worklist[0]
		worklist = worklist[1:]

		currState := dfa.States[curr]

		// Find all symbols appearing in this state (excluding end-marker)
		symbols := make(map[rune]bool)
		for pos := range currState {
			sym := posToSymbol[pos]
			if sym != regex.EndMarker {
				symbols[sym] = true
			}
		}

		// For each symbol, compute the transition
		for sym := range symbols {
			// U = union of followpos(i) for all i in state where posToSymbol[i] == sym
			U := make(map[int]bool)
			for pos := range currState {
				if posToSymbol[pos] == sym {
					for fp := range followPos[pos] {
						U[fp] = true
					}
				}
			}

			if len(U) == 0 {
				continue
			}

			key := setKey(U)
			nextID, exists := keyToID[key]
			if !exists {
				nextID = stateID
				stateID++
				keyToID[key] = nextID
				dfa.States[nextID] = U
				worklist = append(worklist, nextID)

				if U[endPos] {
					dfa.Accepting[nextID] = true
					dfa.StateToken[nextID] = tokenName
				}
			}

			// Add transition
			if dfa.Transitions[curr] == nil {
				dfa.Transitions[curr] = make(map[rune]int)
			}
			dfa.Transitions[curr][sym] = nextID
		}
	}

	return dfa
}

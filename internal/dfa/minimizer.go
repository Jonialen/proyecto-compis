package dfa

// Minimize applies the Table-Filling algorithm to reduce a DFA to its
// minimal number of states by merging equivalent states.
// Two states are equivalent if for every possible input string, they both
// either lead to an accepting state or both lead to a non-accepting state.
func Minimize(d *DFA) *DFA {
	if len(d.States) == 0 {
		return d
	}

	// 1. Preparation: Get a stable, sorted list of current state IDs.
	stateList := sortedStateIDs(d)
	n := len(stateList)
	if n == 0 {
		return d
	}

	// Map each state ID to its index in the stateList for efficient table lookup.
	stateIndex := make(map[int]int, n)
	for i, s := range stateList {
		stateIndex[s] = i
	}

	// 2. The Distinguishability Table:
	// A triangular matrix where marked[i][j] = true means states i and j
	// have been proven to be non-equivalent (distinguishable).
	marked := make([][]bool, n)
	for i := range marked {
		marked[i] = make([]bool, n)
	}

	// Helper function to mark a pair of states as distinguishable.
	mark := func(i, j int) {
		if i > j {
			i, j = j, i
		}
		marked[i][j] = true
	}
	// Helper function to check if a pair of states is already marked.
	isMarked := func(i, j int) bool {
		if i > j {
			i, j = j, i
		}
		return marked[i][j]
	}

	// Initial Marking: A pair of states is distinguishable if one is an
	// accepting state and the other is not.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			si, sj := stateList[i], stateList[j]
			if d.Accepting[si] != d.Accepting[sj] {
				mark(i, j)
			}
		}
	}

	// 3. Iterative Propagation:
	// For every pair of non-marked states (i, j), check if they transition
	// to a pair of states (ri, rj) that are already known to be distinguishable.
	// Repeat this process until no more markings can be made.
	changed := true
	for changed {
		changed = false
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if isMarked(i, j) {
					continue
				}
				si, sj := stateList[i], stateList[j]

				// Collect the union of all symbols used in transitions from both states.
				symbols := make(map[rune]bool)
				for sym := range d.Transitions[si] {
					symbols[sym] = true
				}
				for sym := range d.Transitions[sj] {
					symbols[sym] = true
				}

				for sym := range symbols {
					ri, hasRi := d.Transitions[si][sym]
					rj, hasRj := d.Transitions[sj][sym]

					if hasRi != hasRj {
						// One state has a transition for 'sym' and the other doesn't.
						mark(i, j)
						changed = true
						break
					}
					if !hasRi && !hasRj {
						continue
					}
					if ri == rj {
						continue
					}
					// If the targets (ri, rj) are already marked as distinguishable, then (si, sj) are too.
					riIdx := stateIndex[ri]
					rjIdx := stateIndex[rj]
					if isMarked(riIdx, rjIdx) {
						mark(i, j)
						changed = true
						break
					}
				}
			}
		}
	}

	// 4. State Grouping:
	// Use Union-Find to group all non-distinguishable states into equivalence classes.
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(x, y int) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if !isMarked(i, j) {
				union(i, j)
			}
		}
	}

	// Map each equivalence class to a new, unique state ID for the minimal DFA.
	classRep := make(map[int]int)
	newStateID := 0
	indexToNew := make(map[int]int)

	for i := 0; i < n; i++ {
		rep := find(i)
		if _, ok := classRep[rep]; !ok {
			classRep[rep] = newStateID
			newStateID++
		}
		indexToNew[i] = classRep[rep]
	}

	// 5. Reconstruction: Build the minimized DFA using the merged states.
	newDFA := &DFA{
		States:      make(map[int]map[int]bool),
		Transitions: make(map[int]map[rune]int),
		Accepting:   make(map[int]bool),
		StateToken:  make(map[int]string),
	}

	// Map the old start state to its corresponding class representative in the new DFA.
	startIdx := stateIndex[d.Start]
	newDFA.Start = indexToNew[startIdx]

	for i, s := range stateList {
		newS := indexToNew[i]

		// Merge the syntax tree position sets from all states in the same class.
		if newDFA.States[newS] == nil {
			newDFA.States[newS] = make(map[int]bool)
		}
		for p := range d.States[s] {
			newDFA.States[newS][p] = true
		}

		// Preserve acceptance status and token association.
		if d.Accepting[s] {
			newDFA.Accepting[newS] = true
			if tok, ok := d.StateToken[s]; ok {
				newDFA.StateToken[newS] = tok
			}
		}

		// Rebuild transitions: Transitions from any state in a class point to the class representative of the target.
		for sym, next := range d.Transitions[s] {
			nextIdx := stateIndex[next]
			newNext := indexToNew[nextIdx]
			if newDFA.Transitions[newS] == nil {
				newDFA.Transitions[newS] = make(map[rune]int)
			}
			newDFA.Transitions[newS][sym] = newNext
		}
	}

	return newDFA
}

// sortedStateIDs provides a stable, deterministic ordering of state IDs.
func sortedStateIDs(d *DFA) []int {
	ids := make([]int, 0, len(d.States))
	for id := range d.States {
		ids = append(ids, id)
	}
	// Using a simple insertion sort to avoid external dependencies.
	for i := 1; i < len(ids); i++ {
		key := ids[i]
		j := i - 1
		for j >= 0 && ids[j] > key {
			ids[j+1] = ids[j]
			j--
		}
		ids[j+1] = key
	}
	return ids
}

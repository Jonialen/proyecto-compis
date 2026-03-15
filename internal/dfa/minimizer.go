package dfa

// Minimize minimizes a DFA using the table-filling algorithm.
func Minimize(d *DFA) *DFA {
	if len(d.States) == 0 {
		return d
	}

	// Collect all state IDs as a sorted list
	stateList := sortedStateIDs(d)
	n := len(stateList)
	if n == 0 {
		return d
	}

	// Map state ID → index in stateList
	stateIndex := make(map[int]int, n)
	for i, s := range stateList {
		stateIndex[s] = i
	}

	// Triangular table: marked[i][j] = true means states i and j are distinguishable
	// We use indices in stateList, with i < j
	marked := make([][]bool, n)
	for i := range marked {
		marked[i] = make([]bool, n)
	}

	mark := func(i, j int) {
		if i > j {
			i, j = j, i
		}
		marked[i][j] = true
	}
	isMarked := func(i, j int) bool {
		if i > j {
			i, j = j, i
		}
		return marked[i][j]
	}

	// Step 2: mark pairs where one is accepting and the other is not
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			si, sj := stateList[i], stateList[j]
			if d.Accepting[si] != d.Accepting[sj] {
				mark(i, j)
			}
		}
	}

	// Step 3: propagate markings until no change
	changed := true
	for changed {
		changed = false
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if isMarked(i, j) {
					continue
				}
				si, sj := stateList[i], stateList[j]
				// Collect all symbols that have transitions from si or sj
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
						// One has a transition, the other doesn't → distinguishable
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

	// Step 4: build equivalence classes (union-find)
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

	// Build representative map: index → class representative
	classRep := make(map[int]int) // class rep (index) → new state ID
	newStateID := 0
	indexToNew := make(map[int]int) // old index → new state ID

	for i := 0; i < n; i++ {
		rep := find(i)
		if _, ok := classRep[rep]; !ok {
			classRep[rep] = newStateID
			newStateID++
		}
		indexToNew[i] = classRep[rep]
	}

	// Build new DFA
	newDFA := &DFA{
		States:      make(map[int]map[int]bool),
		Transitions: make(map[int]map[rune]int),
		Accepting:   make(map[int]bool),
		StateToken:  make(map[int]string),
	}

	// Set start state
	startIdx := stateIndex[d.Start]
	newDFA.Start = indexToNew[startIdx]

	// Build states and transitions
	for i, s := range stateList {
		newS := indexToNew[i]

		// Merge position sets
		if newDFA.States[newS] == nil {
			newDFA.States[newS] = make(map[int]bool)
		}
		for p := range d.States[s] {
			newDFA.States[newS][p] = true
		}

		// Accepting
		if d.Accepting[s] {
			newDFA.Accepting[newS] = true
			if tok, ok := d.StateToken[s]; ok {
				newDFA.StateToken[newS] = tok
			}
		}

		// Transitions
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

// sortedStateIDs returns the state IDs of a DFA in sorted order.
func sortedStateIDs(d *DFA) []int {
	ids := make([]int, 0, len(d.States))
	for id := range d.States {
		ids = append(ids, id)
	}
	// Simple insertion sort
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

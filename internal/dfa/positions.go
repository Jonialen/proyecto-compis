package dfa

import "genanalex/internal/regex"

// Nullable returns true if the node can match the empty string.
func Nullable(n *Node) bool {
	if n == nil {
		return false
	}
	if n.nullableCache != nil {
		return *n.nullableCache
	}
	var result bool
	switch n.Kind {
	case NodeEpsilon:
		result = true
	case NodeLeaf:
		result = false
	case NodeOr:
		result = Nullable(n.Left) || Nullable(n.Right)
	case NodeCat:
		result = Nullable(n.Left) && Nullable(n.Right)
	case NodeStar:
		result = true
	case NodePlus:
		result = Nullable(n.Left)
	case NodeOpt:
		result = true
	}
	n.nullableCache = &result
	return result
}

// FirstPos returns the set of positions that can match the first character.
func FirstPos(n *Node) map[int]bool {
	if n == nil {
		return map[int]bool{}
	}
	if n.firstPosCache != nil {
		return n.firstPosCache
	}
	var result map[int]bool
	switch n.Kind {
	case NodeEpsilon:
		result = map[int]bool{}
	case NodeLeaf:
		result = map[int]bool{n.Pos: true}
	case NodeOr:
		result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
	case NodeCat:
		if Nullable(n.Left) {
			result = unionSets(FirstPos(n.Left), FirstPos(n.Right))
		} else {
			result = FirstPos(n.Left)
		}
	case NodeStar, NodePlus, NodeOpt:
		result = FirstPos(n.Left)
	default:
		result = map[int]bool{}
	}
	n.firstPosCache = result
	return result
}

// LastPos returns the set of positions that can match the last character.
func LastPos(n *Node) map[int]bool {
	if n == nil {
		return map[int]bool{}
	}
	if n.lastPosCache != nil {
		return n.lastPosCache
	}
	var result map[int]bool
	switch n.Kind {
	case NodeEpsilon:
		result = map[int]bool{}
	case NodeLeaf:
		result = map[int]bool{n.Pos: true}
	case NodeOr:
		result = unionSets(LastPos(n.Left), LastPos(n.Right))
	case NodeCat:
		if Nullable(n.Right) {
			result = unionSets(LastPos(n.Left), LastPos(n.Right))
		} else {
			result = LastPos(n.Right)
		}
	case NodeStar, NodePlus, NodeOpt:
		result = LastPos(n.Left)
	default:
		result = map[int]bool{}
	}
	n.lastPosCache = result
	return result
}

// unionSets returns the union of two position sets.
func unionSets(a, b map[int]bool) map[int]bool {
	result := make(map[int]bool)
	for k := range a {
		result[k] = true
	}
	for k := range b {
		result[k] = true
	}
	return result
}

// setKey converts a position set to a canonical string key for map lookups.
func setKey(s map[int]bool) string {
	if len(s) == 0 {
		return ""
	}
	keys := sortedKeys(s)
	result := make([]byte, 0, len(keys)*4)
	for i, k := range keys {
		if i > 0 {
			result = append(result, ',')
		}
		result = appendInt(result, k)
	}
	return string(result)
}

func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	return append(b, tmp[i:]...)
}

func sortedKeys(s map[int]bool) []int {
	keys := make([]int, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// EndMarkerSym is used to find the end-marker position.
var EndMarkerSym = regex.EndMarker

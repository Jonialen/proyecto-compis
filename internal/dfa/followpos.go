package dfa

// ComputeFollowPos computes the followpos function for all positions in the tree.
// Returns a map from position → set of positions that can follow it.
func ComputeFollowPos(root *Node) map[int]map[int]bool {
	followPos := make(map[int]map[int]bool)
	computeFollow(root, followPos)
	return followPos
}

// computeFollow recursively fills the followpos map.
func computeFollow(n *Node, fp map[int]map[int]bool) {
	if n == nil {
		return
	}

	switch n.Kind {
	case NodeCat:
		// For each i in lastpos(n.Left): followpos(i) ∪= firstpos(n.Right)
		lp := LastPos(n.Left)
		fp1 := FirstPos(n.Right)
		for i := range lp {
			if fp[i] == nil {
				fp[i] = make(map[int]bool)
			}
			for j := range fp1 {
				fp[i][j] = true
			}
		}
		computeFollow(n.Left, fp)
		computeFollow(n.Right, fp)

	case NodeStar, NodePlus:
		// For each i in lastpos(n): followpos(i) ∪= firstpos(n)
		lp := LastPos(n.Left)
		fp1 := FirstPos(n.Left)
		for i := range lp {
			if fp[i] == nil {
				fp[i] = make(map[int]bool)
			}
			for j := range fp1 {
				fp[i][j] = true
			}
		}
		computeFollow(n.Left, fp)

	case NodeOr, NodeOpt:
		computeFollow(n.Left, fp)
		if n.Right != nil {
			computeFollow(n.Right, fp)
		}

	case NodeLeaf, NodeEpsilon:
		// no action
	}
}

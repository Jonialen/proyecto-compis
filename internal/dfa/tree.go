package dfa

import (
	"fmt"
	"strings"

	"genanalex/internal/regex"
)

// NodeKind represents the type of a syntax tree node.
type NodeKind int

const (
	NodeLeaf    NodeKind = iota // leaf with a symbol
	NodeEpsilon                 // ε (epsilon)
	NodeCat                     // concatenation ·
	NodeOr                      // alternation |
	NodeStar                    // Kleene star *
	NodePlus                    // one-or-more +
	NodeOpt                     // zero-or-one ?
)

// Node is a node in the regex syntax tree.
type Node struct {
	Kind   NodeKind
	Symbol rune // only valid for NodeLeaf
	Pos    int  // position number, only for NodeLeaf (1-indexed, 0 means epsilon)
	Left   *Node
	Right  *Node

	// Memoization caches for Nullable/FirstPos/LastPos
	nullableCache *bool
	firstPosCache map[int]bool
	lastPosCache  map[int]bool
}

// BuildTree constructs a syntax tree from a postfix []RegexToken.
// Returns the root node and a map from position → symbol.
func BuildTree(postfix []regex.RegexToken) (*Node, map[int]rune, error) {
	var stack []*Node
	posCounter := 0
	posToSymbol := make(map[int]rune)

	for _, tok := range postfix {
		switch tok.Kind {
		case regex.TokAtom:
			posCounter++
			leaf := &Node{
				Kind:   NodeLeaf,
				Symbol: tok.Atom,
				Pos:    posCounter,
			}
			posToSymbol[posCounter] = tok.Atom
			stack = append(stack, leaf)

		case regex.TokOp:
			switch tok.Op {
			case '|':
				if len(stack) < 2 {
					return nil, nil, fmt.Errorf("not enough operands for |")
				}
				r := stack[len(stack)-1]
				l := stack[len(stack)-2]
				stack = stack[:len(stack)-2]
				stack = append(stack, &Node{Kind: NodeOr, Left: l, Right: r})

			case regex.ConcatOp:
				if len(stack) < 2 {
					return nil, nil, fmt.Errorf("not enough operands for concat")
				}
				r := stack[len(stack)-1]
				l := stack[len(stack)-2]
				stack = stack[:len(stack)-2]
				stack = append(stack, &Node{Kind: NodeCat, Left: l, Right: r})

			case '*':
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for *")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodeStar, Left: n})

			case '+':
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for +")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodePlus, Left: n})

			case '?':
				if len(stack) < 1 {
					return nil, nil, fmt.Errorf("not enough operands for ?")
				}
				n := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stack = append(stack, &Node{Kind: NodeOpt, Left: n})

			default:
				return nil, nil, fmt.Errorf("unknown operator: %q", tok.Op)
			}

		case regex.TokOpen, regex.TokClose:
			// Should not appear in postfix
			return nil, nil, fmt.Errorf("unexpected parenthesis in postfix")
		}
	}

	if len(stack) != 1 {
		return nil, nil, fmt.Errorf("syntax tree build error: %d nodes remaining (expected 1)", len(stack))
	}

	return stack[0], posToSymbol, nil
}

// ToDOT generates a Graphviz DOT representation of the syntax tree.
func ToDOT(root *Node) string {
	var sb strings.Builder
	sb.WriteString("digraph syntaxtree {\n")
	sb.WriteString("  node [shape=circle];\n")
	counter := 0
	var visit func(n *Node) int
	visit = func(n *Node) int {
		if n == nil {
			return -1
		}
		id := counter
		counter++
		label := nodeLabel(n)
		sb.WriteString(fmt.Sprintf("  %d [label=%q];\n", id, label))
		if n.Left != nil {
			leftID := visit(n.Left)
			sb.WriteString(fmt.Sprintf("  %d -> %d;\n", id, leftID))
		}
		if n.Right != nil {
			rightID := visit(n.Right)
			sb.WriteString(fmt.Sprintf("  %d -> %d;\n", id, rightID))
		}
		return id
	}
	visit(root)
	sb.WriteString("}\n")
	return sb.String()
}

func nodeLabel(n *Node) string {
	switch n.Kind {
	case NodeLeaf:
		if n.Symbol == regex.EndMarker {
			return fmt.Sprintf("#(%d)", n.Pos)
		}
		return fmt.Sprintf("%q(%d)", n.Symbol, n.Pos)
	case NodeEpsilon:
		return "ε"
	case NodeCat:
		return "·"
	case NodeOr:
		return "|"
	case NodeStar:
		return "*"
	case NodePlus:
		return "+"
	case NodeOpt:
		return "?"
	}
	return "?"
}

// Package dfa provides tools for building and minimizing Deterministic Finite Automata
// from regular expression syntax trees.
package dfa

import (
	"fmt"
	"strings"

	"genanalex/internal/regex"
)

// NodeKind identifies the type of an operation or leaf in the syntax tree.
type NodeKind int

const (
	NodeLeaf    NodeKind = iota // A leaf node containing an input symbol.
	NodeEpsilon                 // An epsilon (ε) node representing the empty string.
	NodeCat                     // A concatenation (·) node.
	NodeOr                      // An alternation (|) node.
	NodeStar                    // A Kleene star (*) node for 0 or more repetitions.
	NodePlus                    // A plus (+) node for 1 or more repetitions.
	NodeOpt                     // An optional (?) node for 0 or 1 occurrence.
)

// Node represents a single element in the syntax tree of a regular expression.
type Node struct {
	Kind   NodeKind
	Symbol rune // The input symbol (valid for NodeLeaf).
	Pos    int  // The unique position ID (1-indexed) assigned to leaf nodes.
	Left   *Node
	Right  *Node

	// Memoization caches used to optimize the computation of DFA properties.
	nullableCache *bool
	firstPosCache map[int]bool
	lastPosCache  map[int]bool
}

// BuildTree transforms a postfix regular expression into a syntax tree.
// It returns the root node and a mapping of leaf positions to their symbols.
func BuildTree(postfix []regex.RegexToken) (*Node, map[int]rune, error) {
	var stack []*Node
	posCounter := 0
	posToSymbol := make(map[int]rune)

	for _, tok := range postfix {
		switch tok.Kind {
		case regex.TokAtom:
			// Each literal symbol becomes a leaf node with a unique position.
			posCounter++
			leaf := &Node{
				Kind:   NodeLeaf,
				Symbol: tok.Atom,
				Pos:    posCounter,
			}
			posToSymbol[posCounter] = tok.Atom
			stack = append(stack, leaf)

		case regex.TokOp:
			// Operators pop their operands from the stack and push a new subtree.
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
			// Parentheses are used during postfix conversion but should not exist in the final postfix.
			return nil, nil, fmt.Errorf("unexpected parenthesis in postfix")
		}
	}

	// At the end, there should be exactly one node on the stack (the root).
	if len(stack) != 1 {
		return nil, nil, fmt.Errorf("syntax tree build error: %d nodes remaining (expected 1)", len(stack))
	}

	return stack[0], posToSymbol, nil
}

// ToDOT generates a Graphviz DOT representation for visualizing the syntax tree.
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

// nodeLabel provides a descriptive label for each node in the DOT representation.
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

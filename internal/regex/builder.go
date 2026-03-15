package regex

import (
	"fmt"
)

// BuildPostfix converts a normalized token slice into a postfix token slice
// using the Shunting Yard algorithm.
// It also augments the regex: wraps in (regex)·#
func BuildPostfix(normalized []RegexToken) ([]RegexToken, error) {
	// Augment: add end-marker: normalized ++ [·, #]
	augmented := make([]RegexToken, 0, len(normalized)+4)
	augmented = append(augmented, openTok())
	augmented = append(augmented, normalized...)
	augmented = append(augmented, closeTok())
	augmented = append(augmented, opTok(ConcatOp))
	augmented = append(augmented, atomTok(EndMarker))

	return shuntingYard(augmented)
}

// precedence returns the operator precedence.
func precedence(op rune) int {
	switch op {
	case '*', '+', '?':
		return 3
	case ConcatOp:
		return 2
	case '|':
		return 1
	}
	return 0
}

// isUnary returns true if the operator is a unary postfix operator.
func isUnary(op rune) bool {
	return op == '*' || op == '+' || op == '?'
}

// shuntingYard implements the Shunting Yard algorithm on a RegexToken slice.
func shuntingYard(tokens []RegexToken) ([]RegexToken, error) {
	var output []RegexToken
	var opStack []RegexToken

	for _, tok := range tokens {
		switch tok.Kind {
		case TokAtom:
			output = append(output, tok)

		case TokOpen:
			opStack = append(opStack, tok)

		case TokClose:
			// Pop until matching open
			for len(opStack) > 0 && opStack[len(opStack)-1].Kind != TokOpen {
				output = append(output, opStack[len(opStack)-1])
				opStack = opStack[:len(opStack)-1]
			}
			if len(opStack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses: no matching (")
			}
			opStack = opStack[:len(opStack)-1] // pop the (

		case TokOp:
			op := tok.Op
			if isUnary(op) {
				// Unary postfix operators are emitted immediately to the output.
				// They always bind to the immediately preceding operand/group
				// and have the highest precedence — no need for the operator stack.
				output = append(output, tok)
			} else {
				// Binary operators use the standard Shunting-Yard logic.
				for len(opStack) > 0 {
					top := opStack[len(opStack)-1]
					if top.Kind == TokOpen {
						break
					}
					if top.Kind != TokOp {
						break
					}
					topOp := top.Op
					// Binary left-associative: pop if top has >= precedence
					if precedence(topOp) >= precedence(op) {
						output = append(output, top)
						opStack = opStack[:len(opStack)-1]
					} else {
						break
					}
				}
				opStack = append(opStack, tok)
			}
		}
	}

	// Pop remaining operators
	for len(opStack) > 0 {
		top := opStack[len(opStack)-1]
		opStack = opStack[:len(opStack)-1]
		if top.Kind == TokOpen {
			return nil, fmt.Errorf("mismatched parentheses: unclosed (")
		}
		output = append(output, top)
	}

	return output, nil
}

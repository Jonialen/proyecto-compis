package tac

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func (l *lowerer) expression(env *environment, expression ast.Expression) (Operand, bool) {
	switch n := expression.(type) {
	case ast.NewExpr:
		return l.newObject(env, n)
	case ast.ThisExpr:
		return l.thisValue(env, n.Span)
	case ast.PropertyAccessExpr:
		if n.Receiver == nil {
			return Operand{}, l.unsupported(n.Span, "unsupported expression in numeric TAC")
		}
		return l.property(env, n.Receiver, n.Name, n.Span)
	case ast.PropertyAssignExpr:
		if n.Receiver == nil || n.Value == nil {
			return Operand{}, l.unsupported(n.Span, "unsupported expression in numeric TAC")
		}
		return l.propertyStore(env, n.Receiver, n.Name, n.Value, n.Span)
	case ast.CallExpr:
		return l.call(env, n, false)
	case ast.LiteralExpr:
		if strings.HasPrefix(n.Lexeme, "\"") {
			return Operand{Kind: Literal, Value: n.Lexeme, Type: model.TypeString}, true
		}
		if n.Lexeme == "null" {
			return Operand{Kind: Literal, Value: n.Lexeme, Type: model.TypeNull}, true
		}
		if n.Lexeme == "true" || n.Lexeme == "false" {
			return Operand{Kind: Literal, Value: n.Lexeme, Type: model.TypeBoolean}, true
		}
		if numericLexeme(n.Lexeme) {
			kind := model.TypeInteger
			if strings.Contains(n.Lexeme, ".") {
				kind = model.TypeFloat
			}
			return Operand{Kind: Literal, Value: n.Lexeme, Type: kind}, true
		}
	case ast.IdentifierExpr:
		if value, ok := env.lookup(n.Name); ok {
			if value.field != nil || (value.function != nil && value.function.method != nil) {
				return l.property(env, ast.ThisExpr{Span: n.Span}, n.Name, n.Span)
			}
			if value.function != nil {
				return l.closure(value.function, n.Span), true
			}
			l.reference(value.operand)
			return value.operand, true
		}
		if env.unavailable(n.Name) {
			return Operand{}, l.unsupported(n.Span, "switch binding is not guaranteed initialized on this path")
		}
	case ast.GroupExpr:
		return l.expression(env, n.Expression)
	case ast.UnaryExpr:
		if n.Operator == "-" || n.Operator == "!" {
			value, ok := l.expression(env, n.Operand)
			if !ok {
				return Operand{}, false
			}
			if n.Operator == "-" && numeric(value.Type) {
				return l.operation(Negate, value, nil, value.Type, n.Span), true
			}
			if n.Operator == "!" && value.Type == model.TypeBoolean {
				return l.operation(LogicalNot, value, nil, model.TypeBoolean, n.Span), true
			}
		}
	case ast.BinaryExpr:
		switch n.Operator {
		case "&&", "||":
			return l.logicalValue(env, n)
		case "+", "-", "*", "/", "==", "!=", "<", "<=", ">", ">=":
			left, ok := l.expression(env, n.Left)
			if !ok {
				return Operand{}, false
			}
			// Keep the left value live through the complete recursive RHS.
			if hasCall(n.Right) {
				left = l.freeze(left, n.Left.SourceSpan())
			}
			right, ok := l.expression(env, n.Right)
			if !ok {
				return Operand{}, false
			}
			comparison := n.Operator == "==" || n.Operator == "!=" || n.Operator == "<" || n.Operator == "<=" || n.Operator == ">" || n.Operator == ">="
			if !numeric(left.Type) || !numeric(right.Type) {
				if n.Operator == "+" && left.Type == model.TypeString && right.Type == model.TypeString {
					return l.operation(Add, left, &right, model.TypeString, n.Span), true
				}
				if (n.Operator == "==" || n.Operator == "!=") && ((left.Type == model.TypeString && right.Type == model.TypeString) || (left.Type == model.TypeClass && right.Type == model.TypeClass) || left.Type == model.TypeNull || right.Type == model.TypeNull) {
					return l.operation(Opcode(n.Operator), left, &right, model.TypeBoolean, n.Span), true
				}
				if (n.Operator == "==" || n.Operator == "!=") && left.Type == model.TypeBoolean && right.Type == model.TypeBoolean {
					return l.operation(Opcode(n.Operator), left, &right, model.TypeBoolean, n.Span), true
				}
				return Operand{}, l.unsupported(n.Span, "unsupported operand types in TAC")
			}
			kind := model.TypeInteger
			if left.Type == model.TypeFloat || right.Type == model.TypeFloat || n.Operator == "/" {
				kind = model.TypeFloat
				left = l.widen(left, n.Left.SourceSpan())
				right = l.widen(right, n.Right.SourceSpan())
			}
			if comparison {
				kind = model.TypeBoolean
			}
			return l.operation(Opcode(n.Operator), left, &right, kind, n.Span), true
		}
	}
	return Operand{}, l.unsupported(expression.SourceSpan(), "unsupported expression in numeric TAC")
}

// Validate the grammar's numeric spelling without imposing a machine integer width.
func numericLexeme(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return false
			}
		}
	}
	return true
}

func (l *lowerer) widen(value Operand, span ast.Span) Operand {
	if value.Type == model.TypeInteger {
		return l.operation(IntegerToFloat, value, nil, model.TypeFloat, span)
	}
	return value
}

// Every instruction reads operands before writing. Reuse a consumed operand's
// slot without releasing it; only other consumed temporary slots become free.
func (l *lowerer) operation(op Opcode, left Operand, right *Operand, kind model.TypeKind, span ast.Span) Operand {
	var target Operand
	if left.Kind == Temporary {
		target = left
	} else if right != nil && right.Kind == Temporary {
		target = *right
	} else {
		target = l.pool.acquire(kind)
	}
	target.Type = kind
	target.Class = ""
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: op, Target: target, Left: left, Right: right, Span: span})
	if left.Kind == Temporary && left.Value != target.Value {
		l.pool.release(left)
	}
	if right != nil && right.Kind == Temporary && right.Value != target.Value {
		l.pool.release(*right)
	}
	return target
}

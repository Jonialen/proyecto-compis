package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func statementCall(expression ast.Expression) (ast.CallExpr, bool) {
	if group, ok := expression.(ast.GroupExpr); ok {
		return statementCall(group.Expression)
	}
	call, ok := expression.(ast.CallExpr)
	return call, ok
}

func (l *lowerer) call(env *environment, n ast.CallExpr, statement bool) (Operand, bool) {
	identifier, direct := n.Callee.(ast.IdentifierExpr)
	bound, found := env.lookup(identifier.Name)
	if !direct || !found || bound.function == nil {
		return Operand{}, l.unsupported(n.Span, "only direct global function calls are supported")
	}
	f := bound.function
	kind := f.typeOf.Result.Kind
	if kind == model.TypeNull && !statement {
		return Operand{}, l.unsupported(n.Span, "null-result calls are statement-only")
	}
	if len(n.Arguments) != len(f.typeOf.Params) {
		return Operand{}, l.unsupported(n.Span, "call argument count mismatch")
	}
	arguments := make([]Operand, 0, len(n.Arguments))
	for i, expression := range n.Arguments {
		value, ok := l.expression(env, expression)
		if !ok {
			return Operand{}, false
		}
		if f.typeOf.Params[i].Kind == model.TypeFloat {
			value = l.widen(value, expression.SourceSpan())
		}
		if value.Type != f.typeOf.Params[i].Kind {
			return Operand{}, l.unsupported(expression.SourceSpan(), "call argument type mismatch")
		}
		for _, later := range n.Arguments[i+1:] {
			if hasCall(later) {
				value = l.freeze(value, expression.SourceSpan())
				break
			}
		}
		arguments = append(arguments, value)
	}
	if l.current != nil {
		l.current.edges = append(l.current.edges, f)
	} else {
		root := env
		for root.parent != nil {
			root = root.parent
		}
		available := map[string]bool{}
		for name, binding := range root.active {
			if binding.function == nil && !root.blocked[name] {
				available[binding.operand.Value] = true
			}
		}
		l.registry.calls = append(l.registry.calls, mainCall{callee: f, span: n.Span, available: available})
	}
	var target Operand
	if scalar(kind) {
		target = l.pool.acquire(kind)
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Call, Target: target, Callee: f.label, Arguments: arguments, Span: n.Span})
	for _, argument := range arguments {
		l.pool.release(argument)
	}
	return target, true
}

// Freeze only symbol references: literals are values and live temporaries are
// already protected by the existing pool until their consuming instruction.
func (l *lowerer) freeze(value Operand, span ast.Span) Operand {
	if value.Kind != Symbol {
		return value
	}
	target := l.pool.acquire(value.Type)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: target, Left: value, Span: span})
	return target
}

func hasCall(expression ast.Expression) bool {
	switch n := expression.(type) {
	case ast.CallExpr:
		return true
	case ast.GroupExpr:
		return hasCall(n.Expression)
	case ast.UnaryExpr:
		return hasCall(n.Operand)
	case ast.BinaryExpr:
		return hasCall(n.Left) || hasCall(n.Right)
	case ast.TernaryExpr:
		return hasCall(n.Condition) || hasCall(n.Then) || hasCall(n.Else)
	case ast.AssignExpr:
		return hasCall(n.Target) || hasCall(n.Value)
	case ast.IndexExpr:
		return hasCall(n.Collection) || hasCall(n.Index)
	case ast.PropertyAccessExpr:
		return hasCall(n.Receiver)
	case ast.PropertyAssignExpr:
		return hasCall(n.Receiver) || hasCall(n.Value)
	case ast.NewExpr:
		for _, arg := range n.Arguments {
			if hasCall(arg) {
				return true
			}
		}
	case ast.ArrayExpr:
		for _, arg := range n.Elements {
			if hasCall(arg) {
				return true
			}
		}
	}
	return false
}

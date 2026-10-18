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
	if !direct || !found || (bound.function == nil && bound.typeOf.Kind != model.TypeFunction) {
		return Operand{}, l.unsupported(n.Span, "only direct global function calls are supported")
	}
	f := bound.function
	signature := bound.typeOf
	if f != nil {
		signature = f.typeOf
	}
	if signature.Result == nil {
		return Operand{}, l.unsupported(n.Span, "missing function value signature")
	}
	kind := signature.Result.Kind
	if !scalar(kind) && kind != model.TypeNull {
		return Operand{}, l.unsupported(n.Span, "unsupported function signature")
	}
	if kind == model.TypeNull && !statement {
		return Operand{}, l.unsupported(n.Span, "null-result calls are statement-only")
	}
	if len(n.Arguments) != len(signature.Params) {
		return Operand{}, l.unsupported(n.Span, "call argument count mismatch")
	}
	// Select and freeze the function value BEFORE any argument can reassign an
	// alias or reenter its defining block. Direct global calls keep their old IR.
	indirect := f == nil || f.definingScope != l.globalID
	var selected Operand
	if indirect {
		var ok bool
		selected, ok = l.expression(env, n.Callee)
		if !ok {
			return Operand{}, false
		}
		selected = l.freeze(selected, n.Callee.SourceSpan())
	}
	arguments := make([]Operand, 0, len(n.Arguments))
	for i, expression := range n.Arguments {
		value, ok := l.expression(env, expression)
		if !ok {
			return Operand{}, false
		}
		if signature.Params[i].Kind == model.TypeFloat {
			value = l.widen(value, expression.SourceSpan())
		}
		if !scalar(signature.Params[i].Kind) || value.Type != signature.Params[i].Kind {
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
	if f != nil {
		if l.current != nil {
			l.current.edges = append(l.current.edges, f)
		}
		l.registry.calls = append(l.registry.calls, mainCall{callee: f, span: n.Span, available: availableBindings(env), caller: l.current})
	}
	var target Operand
	if scalar(kind) {
		target = l.pool.acquire(kind)
	}
	if indirect {
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: CallIndirect, Target: target, Left: selected, Arguments: arguments, Signature: &signature, Span: n.Span})
		l.pool.release(selected)
	} else {
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: Call, Target: target, Callee: f.label, Arguments: arguments, Span: n.Span})
	}
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

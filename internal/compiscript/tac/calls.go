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
	if direct && (!found || (bound.function == nil && bound.typeOf.Kind != model.TypeFunction)) {
		return Operand{}, l.unsupported(n.Span, "only direct global function calls are supported")
	}
	f := bound.function
	signature := bound.typeOf
	if !direct {
		signature = l.expressionType(env, n.Callee)
	}
	if f != nil {
		signature = f.typeOf
	}
	typeClasses := l.expressionTypeClasses(env, n.Callee)
	if signature.Result == nil {
		return Operand{}, l.unsupported(n.Span, "missing function value signature")
	}
	kind := signature.Result.Kind
	if !callValue(kind) && kind != model.TypeNull {
		return Operand{}, l.unsupported(n.Span, "unsupported function signature")
	}
	if len(n.Arguments) != len(signature.Params) {
		return Operand{}, l.unsupported(n.Span, "call argument count mismatch")
	}
	// Select and freeze the function value BEFORE any argument can reassign an
	// alias or reenter its defining block. Direct global calls keep their old IR.
	indirect := f == nil || f.definingScope != l.globalID || f.method != nil
	var selected Operand
	if indirect {
		var ok bool
		selected, ok = l.expression(env, n.Callee)
		if !ok {
			return Operand{}, false
		}
		selected = l.freeze(selected, n.Callee.SourceSpan())
	}
	arguments, ok := l.arguments(env, n.Arguments, signature.Params, typeClasses)
	if !ok {
		return Operand{}, false
	}
	if f != nil && f.method == nil {
		if l.current != nil {
			l.current.edges = append(l.current.edges, f)
		}
		l.registry.calls = append(l.registry.calls, mainCall{callee: f, span: n.Span, available: availableBindings(env), caller: l.current})
	}
	var target Operand
	if callValue(kind) {
		target = l.pool.acquire(kind)
		target = l.typedOperand(env, target, *signature.Result)
		if len(typeClasses) != 0 {
			target.Class = typeClasses[len(signature.Params)]
		}
	}
	if indirect {
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: CallIndirect, Target: target, Left: selected, Arguments: arguments, Signature: &signature, TypeClasses: typeClasses, Span: n.Span})
		l.pool.release(selected)
	} else {
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: Call, Target: target, Callee: f.label, Arguments: arguments, Span: n.Span})
	}
	for _, argument := range arguments {
		l.pool.release(argument)
	}
	if kind == model.TypeNull && !statement {
		return Operand{Kind: Literal, Value: "null", Type: model.TypeNull}, true
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
	target.Class = value.Class
	target.Shape = value.Shape
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
		return true
	case ast.IndexExpr:
		return hasCall(n.Collection) || hasCall(n.Index)
	case ast.PropertyAccessExpr:
		return hasCall(n.Receiver)
	case ast.PropertyAssignExpr:
		return true
	case ast.NewExpr:
		return true // Field initializers and the constructor can call arbitrary code.
	case ast.ArrayExpr:
		for _, arg := range n.Elements {
			if hasCall(arg) {
				return true
			}
		}
	}
	return false
}

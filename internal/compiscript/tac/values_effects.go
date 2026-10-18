package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func (l *lowerer) identifierAssignment(env *environment, n ast.AssignExpr) (Operand, bool) {
	target, ok := n.Target.(ast.IdentifierExpr)
	if !ok {
		return Operand{}, l.unsupported(n.Span, "assignment requires an identifier or index")
	}
	b, ok := env.lookup(target.Name)
	if !ok || !b.mutable {
		return Operand{}, l.unsupported(n.Span, "assignment requires an active mutable binding")
	}
	if b.field != nil {
		return l.propertyStore(env, ast.ThisExpr{Span: target.Span}, target.Name, n.Value, n.Span)
	}
	l.reference(b.operand)
	if !l.store(env, b.operand, n.Value, n.Span) {
		return Operand{}, false
	}
	return b.operand, true
}

func (l *lowerer) ternary(env *environment, n ast.TernaryExpr, expected Operand) (Operand, bool) {
	if expected.Type == "" {
		t := l.expressionType(env, n)
		expected = l.typedOperand(env, Operand{Type: t.Kind}, t)
		expected.Class = l.expressionClass(env, n)
	}
	if expected.Type == model.TypeClass || expected.Type == model.TypeList {
		// Origin recovery uses binding identities, never a textual type lookup
		// at a shadowing use site. Explicit contexts still constrain each arm.
		if expected.Class == "" {
			expected.Class = l.expressionClass(env, n)
		}
	}
	if !storedValue(expected.Type) {
		return Operand{}, l.unsupported(n.Span, "dynamic ternary values require guarded lowering")
	}
	result := l.pool.acquire(expected.Type)
	result.Class, result.Shape = expected.Class, expected.Shape
	yes, no, join := l.labelName(), l.labelName(), l.labelName()
	condition, ok := l.expression(env, n.Condition)
	if !ok {
		return Operand{}, false
	}
	// branchTrue explicitly requires a boolean at runtime; never use truthiness.
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: BranchTrue, Left: condition, Label: yes, Span: n.Condition.SourceSpan()})
	l.pool.release(condition)
	l.control(Jump, no, n.Span)
	l.control(Label, yes, n.Then.SourceSpan())
	if !l.store(env, result, n.Then, n.Then.SourceSpan()) {
		return Operand{}, false
	}
	l.control(Jump, join, n.Span)
	l.control(Label, no, n.Else.SourceSpan())
	if !l.store(env, result, n.Else, n.Else.SourceSpan()) {
		return Operand{}, false
	}
	l.control(Label, join, n.Span)
	return result, true
}

package tac

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// Shape is the complete slash-separated element-kind chain. Class retains the
// declaration identity of its nominal leaf, including through nested lists.
func listElement(list Operand) Operand {
	kind, rest, _ := strings.Cut(list.Shape, "/")
	if kind == "" {
		kind = string(model.TypeError)
	}
	return Operand{Type: model.TypeKind(kind), Shape: rest, Class: list.Class}
}

func (l *lowerer) elementAssignable(target, value Operand) bool {
	return (target.Type == model.TypeFloat && value.Type == model.TypeInteger) || l.assignable(target, value)
}

// Recover the homogeneous shape already accepted by semantics, including numeric
// widening below arbitrary list nesting. Copy nodes rather than semantic types.
func mergeListElementTypes(a, b model.Type) model.Type {
	if a.Kind == model.TypeError || b.Kind == model.TypeError {
		return model.Type{Kind: model.TypeError}
	}
	if a.Kind == "" || a.Kind == model.TypeNull || (a.Kind == model.TypeInteger && b.Kind == model.TypeFloat) {
		return b
	}
	if a.Kind == model.TypeList && b.Kind == model.TypeList {
		if a.Element == nil {
			return b
		}
		if b.Element != nil {
			e := mergeListElementTypes(*a.Element, *b.Element)
			a.Element = &e
		}
	}
	if a.Kind != "" && b.Kind != "" && a.Kind != b.Kind && b.Kind != model.TypeNull && !(numeric(a.Kind) && numeric(b.Kind)) {
		return model.Type{Kind: model.TypeError}
	}
	return a
}

func (l *lowerer) listStorageType(scope int, o Operand, t model.Type) Operand {
	if t.Kind != model.TypeList || t.Element == nil {
		return o
	}
	e := l.listStorageType(scope, Operand{Type: t.Element.Kind}, *t.Element)
	o.Shape = string(e.Type)
	if e.Shape != "" {
		o.Shape += "/" + e.Shape
	}
	for leaf := t.Element; leaf != nil; leaf = leaf.Element {
		if leaf.Kind == model.TypeClass {
			o.Class = l.classForType(scope, leaf.Name)
		}
	}
	return o
}

func (l *lowerer) expressionExpected(env *environment, e ast.Expression, expected Operand) (Operand, bool) {
	switch n := e.(type) {
	case ast.GroupExpr:
		return l.expressionExpected(env, n.Expression, expected)
	case ast.ArrayExpr:
		return l.listLiteral(env, n, expected)
	case ast.TernaryExpr:
		return l.ternary(env, n, expected)
	default:
		return l.expression(env, e)
	}
}

func (l *lowerer) listLiteral(env *environment, n ast.ArrayExpr, expected Operand) (Operand, bool) {
	if expected.Type != model.TypeList || expected.Shape == "" {
		t := l.expressionType(env, n)
		expected = l.typedOperand(env, Operand{Type: model.TypeList}, t)
		expected.Class = l.expressionClass(env, n)
	}
	element := listElement(expected)
	values := make([]Operand, 0, len(n.Elements))
	for _, e := range n.Elements {
		value, ok := l.expressionExpected(env, e, element)
		if !ok {
			return Operand{}, false
		}
		if value.Type == model.TypeError && element.Type != model.TypeError {
			value = l.checkedValue(element, value, e.SourceSpan())
		}
		if element.Type == model.TypeFloat {
			value = l.widen(value, e.SourceSpan())
		}
		if !l.assignable(element, value) {
			return Operand{}, l.unsupported(e.SourceSpan(), "list element type mismatch")
		}
		// Every element is a value at its source position, even if a later
		// element mutates the binding. Keep all temporaries live until allocation.
		values = append(values, l.freeze(value, e.SourceSpan()))
	}
	target := l.pool.acquire(model.TypeList)
	target.Shape, target.Class = expected.Shape, expected.Class
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: NewList, Target: target, Arguments: values, Span: n.Span})
	for _, v := range values {
		l.pool.release(v)
	}
	return target, true
}

func (l *lowerer) indexTarget(env *environment, n ast.IndexExpr) (Operand, Operand, bool) {
	list, ok := l.expression(env, n.Collection)
	if !ok {
		return Operand{}, Operand{}, false
	}
	if list.Type != model.TypeList && list.Type != model.TypeError {
		return Operand{}, Operand{}, l.unsupported(n.Span, "index requires a typed list")
	}
	list = l.freeze(list, n.Collection.SourceSpan())
	index, ok := l.expression(env, n.Index)
	if !ok {
		return Operand{}, Operand{}, false
	}
	if index.Type != model.TypeInteger && index.Type != model.TypeError {
		return Operand{}, Operand{}, l.unsupported(n.Span, "index requires an integer")
	}
	return list, index, true
}

func (l *lowerer) indexRead(env *environment, n ast.IndexExpr) (Operand, bool) {
	list, index, ok := l.indexTarget(env, n)
	if !ok {
		return Operand{}, false
	}
	element := listElement(list)
	target := l.pool.acquire(element.Type)
	target.Class, target.Shape = element.Class, element.Shape
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: LoadIndex, Target: target, Left: list, Index: &index, Span: n.Span})
	l.pool.release(list)
	l.pool.release(index)
	return target, true
}

func (l *lowerer) indexStore(env *environment, n ast.IndexExpr, e ast.Expression, span ast.Span) bool {
	value, ok := l.indexAssignment(env, n, e, span)
	if ok {
		l.pool.release(value)
	}
	return ok
}

func (l *lowerer) indexAssignment(env *environment, n ast.IndexExpr, e ast.Expression, span ast.Span) (Operand, bool) {
	list, index, ok := l.indexTarget(env, n)
	if !ok {
		return Operand{}, false
	}
	index = l.freeze(index, n.Index.SourceSpan())
	element := listElement(list)
	value, ok := l.expressionExpected(env, e, element)
	if !ok {
		return Operand{}, false
	}
	if value.Type == model.TypeError && element.Type != model.TypeError {
		value = l.checkedValue(element, value, e.SourceSpan())
	}
	if element.Type == model.TypeFloat {
		value = l.widen(value, e.SourceSpan())
	}
	if !l.assignable(element, value) {
		return Operand{}, l.unsupported(span, "list element type mismatch")
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: StoreIndex, Left: list, Index: &index, Right: &value, Span: span})
	l.pool.release(list)
	l.pool.release(index)
	return value, true
}

func (l *lowerer) foreachStatement(env *environment, n ast.ForeachStmt) bool {
	list, ok := l.expression(env, n.Iterable)
	if !ok {
		return false
	}
	if (list.Type != model.TypeList && list.Type != model.TypeError) || n.Body == nil {
		return l.unsupported(n.Span, "foreach requires a typed list and body")
	}
	list = l.freeze(list, n.Iterable.SourceSpan())
	length := l.pool.acquire(model.TypeInteger)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: ListLength, Target: length, Left: list, Span: n.Span})
	index := l.pool.acquire(model.TypeInteger)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: index, Left: Operand{Kind: Literal, Type: model.TypeInteger, Value: "0"}, Span: n.Span})
	check, body, post, exit := l.labelName(), l.labelName(), l.labelName(), l.labelName()
	l.control(Label, check, n.Span)
	// Loop-state cells stay live across the body and must never be consumed by
	// operation(), which deliberately recycles its input temporaries.
	condition := l.pool.acquire(model.TypeBoolean)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Less, Target: condition, Left: index, Right: &length, Span: n.Span})
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: BranchTrue, Left: condition, Label: body, Span: n.Span})
	l.pool.release(condition)
	l.control(Jump, exit, n.Span)
	l.control(Label, body, n.Body.Span)
	child := l.environment(env, model.ScopeBlock, n.Body.Span)
	if child == nil {
		return false
	}
	element := listElement(list)
	var target Operand
	for _, s := range l.scopes {
		if s.ID == child.id {
			for _, symbol := range s.Symbols {
				if symbol.Name == n.Name && symbol.Span == n.Span {
					target = symbolOperand(child.id, symbol)
					target.Class, target.Shape = element.Class, element.Shape
					child.active[n.Name] = binding{operand: target, mutable: true, typeOf: symbol.Type, typeClasses: l.expressionTypeClasses(env, n.Iterable)}
					l.recordStorageIdentity(child.id, target)
				}
			}
		}
	}
	if target.Kind == "" {
		return l.unsupported(n.Span, "missing matching foreach binding")
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: LoadIndex, Target: target, Left: list, Index: &index, Span: n.Span})
	if !l.transferBody(exit, post, func() bool { return l.statements(child, n.Body.Statements) }) {
		return false
	}
	l.control(Label, post, n.Span)
	one := Operand{Kind: Literal, Type: model.TypeInteger, Value: "1"}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Add, Target: index, Left: index, Right: &one, Span: n.Span})
	l.control(Jump, check, n.Span)
	l.control(Label, exit, n.Span)
	l.pool.release(index)
	l.pool.release(length)
	l.pool.release(list)
	return true
}

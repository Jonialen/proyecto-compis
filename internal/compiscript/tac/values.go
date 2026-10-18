package tac

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func (l *lowerer) arguments(env *environment, expressions ast.Expressions, params model.Types, typeClasses []string) ([]Operand, bool) {
	if len(expressions) != len(params) {
		return nil, l.unsupported(ast.Span{}, "call argument count mismatch")
	}
	args := make([]Operand, 0, len(expressions))
	for i, expression := range expressions {
		expected := l.typedOperand(env, Operand{Type: params[i].Kind}, params[i])
		if len(typeClasses) != 0 {
			expected.Class = typeClasses[i]
		}
		value, ok := l.expressionExpected(env, expression, expected)
		if !ok {
			return nil, false
		}
		if params[i].Kind == model.TypeFloat {
			value = l.widen(value, expression.SourceSpan())
		}
		if !callValue(params[i].Kind) || !l.assignable(expected, value) {
			return nil, l.unsupported(expression.SourceSpan(), "call argument type mismatch")
		}
		for _, later := range expressions[i+1:] {
			if hasCall(later) {
				value = l.freeze(value, expression.SourceSpan())
				break
			}
		}
		args = append(args, value)
	}
	return args, true
}

// Recover static types from already validated declarations without a second
// frontend pass. Runtime member selection still uses the actual receiver class.
func (l *lowerer) expressionType(env *environment, expression ast.Expression) model.Type {
	switch n := expression.(type) {
	case ast.AssignExpr:
		return l.expressionType(env, n.Target)
	case ast.PropertyAssignExpr:
		return l.expressionType(env, ast.PropertyAccessExpr{Receiver: n.Receiver, Name: n.Name})
	case ast.TernaryExpr:
		return mergeListElementTypes(l.expressionType(env, n.Then), l.expressionType(env, n.Else))
	case ast.ArrayExpr:
		var element model.Type
		for _, e := range n.Elements {
			t := l.expressionType(env, e)
			element = mergeListElementTypes(element, t)
		}
		t := model.Type{Kind: model.TypeList, Name: "list"}
		if element.Kind != "" {
			t.Element = &element
		}
		return t
	case ast.IndexExpr:
		t := l.expressionType(env, n.Collection)
		if t.Element != nil {
			return *t.Element
		}
	case ast.LiteralExpr:
		kind := model.TypeInteger
		switch {
		case n.Lexeme == "null":
			kind = model.TypeNull
		case n.Lexeme == "true" || n.Lexeme == "false":
			kind = model.TypeBoolean
		case strings.HasPrefix(n.Lexeme, "\""):
			kind = model.TypeString
		case strings.Contains(n.Lexeme, "."):
			kind = model.TypeFloat
		}
		return model.Type{Kind: kind}
	case ast.UnaryExpr:
		if n.Operator == "!" {
			return model.Type{Kind: model.TypeBoolean}
		}
		return l.expressionType(env, n.Operand)
	case ast.BinaryExpr:
		if n.Operator != "+" && n.Operator != "-" && n.Operator != "*" && n.Operator != "/" && n.Operator != "%" {
			return model.Type{Kind: model.TypeBoolean}
		}
		a, b := l.expressionType(env, n.Left), l.expressionType(env, n.Right)
		if n.Operator == "/" || b.Kind == model.TypeFloat {
			return model.Type{Kind: model.TypeFloat}
		}
		return a
	case ast.IdentifierExpr:
		b, _ := env.lookup(n.Name)
		if b.function != nil {
			return b.function.typeOf
		}
		return b.typeOf
	case ast.GroupExpr:
		return l.expressionType(env, n.Expression)
	case ast.ThisExpr:
		b, _ := env.lookup("$this")
		c := l.classID(b.operand.Class)
		if c != nil {
			return model.Type{Kind: model.TypeClass, Name: c.layout.Name}
		}
	case ast.NewExpr:
		return model.Type{Kind: model.TypeClass, Name: n.ClassName}
	case ast.PropertyAccessExpr:
		t := l.expressionType(env, n.Receiver)
		o := l.typedOperand(env, Operand{Type: t.Kind}, t)
		o.Class = l.expressionClass(env, n.Receiver)
		field, method := l.member(o, n.Name)
		if field != nil {
			return field.Type
		}
		if method != nil {
			return method.Signature
		}
	case ast.CallExpr:
		t := l.expressionType(env, n.Callee)
		if t.Result != nil {
			return *t.Result
		}
	}
	return model.Type{}
}

// Nominal origins travel with private bindings and function signatures. These
// queries never evaluate the source again or mutate semantic snapshots.
func (l *lowerer) expressionClass(env *environment, expression ast.Expression) string {
	switch n := expression.(type) {
	case ast.AssignExpr:
		return l.expressionClass(env, n.Target)
	case ast.PropertyAssignExpr:
		return l.expressionClass(env, ast.PropertyAccessExpr{Receiver: n.Receiver, Name: n.Name})
	case ast.TernaryExpr:
		a, b := l.expressionClass(env, n.Then), l.expressionClass(env, n.Else)
		if a == "" {
			return b
		}
		if b == "" || l.assignable(Operand{Type: model.TypeClass, Class: a}, Operand{Type: model.TypeClass, Class: b}) {
			return a
		}
		return b
	case ast.IndexExpr:
		return l.expressionClass(env, n.Collection)
	case ast.ArrayExpr:
		var origin string
		for _, e := range n.Elements {
			if c := l.expressionClass(env, e); c != "" {
				if origin == "" || l.assignable(Operand{Type: model.TypeClass, Class: c}, Operand{Type: model.TypeClass, Class: origin}) {
					origin = c
				}
			}
		}
		return origin
	case ast.IdentifierExpr:
		b, _ := env.lookup(n.Name)
		if b.field != nil {
			return b.field.Class
		}
		return b.operand.Class
	case ast.ThisExpr:
		b, _ := env.lookup("$this")
		return b.operand.Class
	case ast.GroupExpr:
		return l.expressionClass(env, n.Expression)
	case ast.NewExpr:
		if c := l.findClass(env, n.ClassName); c != nil {
			return c.layout.ID
		}
	case ast.PropertyAccessExpr:
		field, _ := l.member(Operand{Class: l.expressionClass(env, n.Receiver)}, n.Name)
		if field != nil {
			return field.Class
		}
	case ast.CallExpr:
		classes := l.expressionTypeClasses(env, n.Callee)
		if len(classes) != 0 {
			return classes[len(classes)-1]
		}
	}
	return ""
}

func (l *lowerer) functionTypeClasses(label string) []string {
	for _, f := range l.registry.ordered {
		if f.label == label {
			return f.typeClasses
		}
	}
	return nil
}

func (l *lowerer) expressionTypeClasses(env *environment, expression ast.Expression) []string {
	switch n := expression.(type) {
	case ast.ArrayExpr:
		// A function-valued list retains its element signature's declaration
		// origins, not names resolved at a later indexing/alias use site. Empty
		// nested literals do not establish the homogeneous element signature.
		for _, element := range n.Elements {
			t := l.expressionType(env, element)
			for t.Kind == model.TypeList && t.Element != nil {
				t = *t.Element
			}
			if t.Kind == model.TypeFunction {
				return l.expressionTypeClasses(env, element)
			}
		}
	case ast.IndexExpr:
		return l.expressionTypeClasses(env, n.Collection)
	case ast.AssignExpr:
		return l.expressionTypeClasses(env, n.Target)
	case ast.TernaryExpr:
		return l.expressionTypeClasses(env, n.Then)
	case ast.IdentifierExpr:
		b, _ := env.lookup(n.Name)
		if b.function != nil {
			return b.function.typeClasses
		}
		return b.typeClasses
	case ast.GroupExpr:
		return l.expressionTypeClasses(env, n.Expression)
	case ast.PropertyAccessExpr:
		_, method := l.member(Operand{Class: l.expressionClass(env, n.Receiver)}, n.Name)
		if method != nil {
			return l.functionTypeClasses(method.Callee)
		}
	}
	return nil
}

func (l *lowerer) recordStorageIdentity(scope int, operand Operand) {
	layout, ok := l.registry.environments[scope]
	if !ok {
		return
	}
	for i, binding := range layout.Bindings {
		if binding.Value == operand.Value {
			layout.Bindings[i] = operand
			break
		}
	}
	l.registry.environments[scope] = layout
}

func (l *lowerer) storageIdentity(operand Operand) Operand {
	for _, binding := range l.registry.environments[operandScope(operand)].Bindings {
		if binding.Value == operand.Value {
			return binding
		}
	}
	return operand
}

// Reference storage is separate from numeric arithmetic. Nominal identities are
// semantic scope-qualified class declarations, never merely the kind "class".
func callValue(kind model.TypeKind) bool {
	return scalar(kind) || kind == model.TypeString || kind == model.TypeClass || kind == model.TypeList
}

func (l *lowerer) typedOperand(env *environment, o Operand, t model.Type) Operand {
	if t.Kind == model.TypeList && t.Element != nil {
		e := l.typedOperand(env, Operand{Type: t.Element.Kind}, *t.Element)
		o.Shape = string(e.Type)
		if e.Shape != "" {
			o.Shape += "/" + e.Shape
		}
		o.Class = e.Class
	}
	if t.Kind == model.TypeClass {
		if c := l.findClass(env, t.Name); c != nil {
			o.Class = c.layout.ID
		}
	}
	return o
}

func (l *lowerer) assignable(target, value Operand) bool {
	if value.Type == model.TypeNull {
		return target.Type == model.TypeNull || target.Type == model.TypeClass || target.Type == model.TypeString || target.Type == model.TypeList
	}
	if target.Type != value.Type {
		return false
	}
	if target.Type == model.TypeList {
		return target.Shape != "" && value.Shape != "" && l.elementAssignable(listElement(target), listElement(value))
	}
	if target.Type != model.TypeClass {
		return true
	}
	for c := l.classID(value.Class); c != nil; c = c.parent {
		if c.layout.ID == target.Class {
			return true
		}
	}
	return false
}

func (l *lowerer) classForType(scope int, name string) string {
	for scope != 0 {
		for _, c := range l.registry.classes {
			if c.owner == scope && c.layout.Name == name {
				return c.layout.ID
			}
		}
		next := 0
		for _, s := range l.scopes {
			if s.ID == scope {
				next = s.ParentID
				break
			}
		}
		scope = next
	}
	return ""
}

func (l *lowerer) resolveEnvironmentTypes() {
	for id, layout := range l.registry.environments {
		for i, o := range layout.Bindings {
			if o.Type == model.TypeList && o.Shape == "" {
				for _, scope := range l.scopes {
					if scope.ID == id {
						for _, symbol := range scope.Symbols {
							if symbolOperand(id, symbol).Value == o.Value {
								layout.Bindings[i] = l.listStorageType(id, o, symbol.Type)
							}
						}
					}
				}
				continue
			}
			if o.Type != model.TypeClass || o.Class != "" {
				continue
			}
			for _, scope := range l.scopes {
				if scope.ID != id {
					continue
				}
				for _, symbol := range scope.Symbols {
					if symbolOperand(id, symbol).Value == o.Value {
						layout.Bindings[i].Class = l.classForType(id, symbol.Type.Name)
					}
				}
			}
		}
		l.registry.environments[id] = layout
	}
}

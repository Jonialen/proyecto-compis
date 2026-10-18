package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func (l *lowerer) arguments(env *environment, expressions ast.Expressions, params model.Types, typeClasses []string) ([]Operand, bool) {
	if len(expressions) != len(params) {
		return nil, l.unsupported(ast.Span{}, "call argument count mismatch")
	}
	args := make([]Operand, 0, len(expressions))
	for i, expression := range expressions {
		value, ok := l.expression(env, expression)
		if !ok {
			return nil, false
		}
		if params[i].Kind == model.TypeFloat {
			value = l.widen(value, expression.SourceSpan())
		}
		expected := l.typedOperand(env, Operand{Type: params[i].Kind}, params[i])
		if len(typeClasses) != 0 {
			expected.Class = typeClasses[i]
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
	if operand.Type != model.TypeClass {
		return
	}
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
	return scalar(kind) || kind == model.TypeString || kind == model.TypeClass
}

func (l *lowerer) typedOperand(env *environment, o Operand, t model.Type) Operand {
	if t.Kind == model.TypeClass {
		if c := l.findClass(env, t.Name); c != nil {
			o.Class = c.layout.ID
		}
	}
	return o
}

func (l *lowerer) assignable(target, value Operand) bool {
	if value.Type == model.TypeNull {
		return target.Type == model.TypeClass || target.Type == model.TypeString
	}
	if target.Type != value.Type {
		return false
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

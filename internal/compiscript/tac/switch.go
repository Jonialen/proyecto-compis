package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// A fork preserves semantic identity and incoming initialized bindings, not a
// fictitious case scope. Activation in one body never initializes another path.
func (e *environment) fork() *environment {
	result := &environment{id: e.id, parent: e.parent, active: map[string]binding{}, blocked: map[string]bool{}}
	for name, value := range e.active {
		result.active[name] = value
	}
	for name := range e.blocked {
		result.blocked[name] = true
	}
	return result
}

// Only switch bodies share their incoming semantic scope. Real blocks and
// if/loop bodies have their own snapshots, so their declarations do not escape.
func (e *environment) blockSwitchDeclarations(statements ast.Statements) {
	if e.blocked == nil {
		e.blocked = map[string]bool{}
	}
	for _, statement := range statements {
		name := ""
		switch n := statement.(type) {
		case ast.VarDeclStmt:
			name = n.Name
		case ast.ConstDeclStmt:
			name = n.Name
		case ast.SwitchStmt:
			for _, c := range n.Cases {
				e.blockSwitchDeclarations(c.Statements)
			}
		}
		if name != "" {
			if _, initialized := e.active[name]; !initialized {
				e.blocked[name] = true
			}
		}
	}
}

func (l *lowerer) switchStatement(env *environment, n ast.SwitchStmt) bool {
	selector, ok := l.expression(env, n.Value)
	if !ok {
		return false
	}
	if selector.Type == model.TypeError {
		selector = l.checkedValue(Operand{Type: model.TypeBoolean}, selector, n.Value.SourceSpan())
	}
	if selector.Type != model.TypeBoolean {
		return l.unsupported(n.Value.SourceSpan(), "condition requires a boolean value")
	}
	for _, c := range n.Cases {
		if !c.Default && hasCall(c.Value) {
			selector = l.freeze(selector, n.Value.SourceSpan())
			break
		}
	}
	exit := l.labelName()
	labels := make([]string, len(n.Cases))
	fallback := exit
	dispatch := env.fork()
	for i, c := range n.Cases {
		labels[i] = l.labelName()
		if c.Default {
			fallback = labels[i]
		} else {
			value, ok := l.expression(dispatch, c.Value)
			if !ok {
				return false
			}
			if value.Type == model.TypeError {
				value = l.checkedValue(Operand{Type: model.TypeBoolean}, value, c.Value.SourceSpan())
			}
			if value.Type != model.TypeBoolean {
				return l.unsupported(c.Span, "switch case requires a boolean value")
			}
			// operation would consume/reuse the selector's slot. Reserve an
			// independent result and retain the selector through every test.
			comparison := l.pool.acquire(model.TypeBoolean)
			l.program.Instructions = append(l.program.Instructions,
				Instruction{Op: Equal, Target: comparison, Left: selector, Right: &value, Span: c.Value.SourceSpan()},
				Instruction{Op: BranchTrue, Left: comparison, Label: labels[i], Span: c.Span})
			l.pool.release(comparison)
			l.pool.release(value)
		}
		dispatch.blockSwitchDeclarations(c.Statements)
	}
	l.control(Jump, fallback, n.Span)
	l.pool.release(selector)
	return l.transferBody(exit, "", func() bool {
		for i, c := range n.Cases {
			l.control(Label, labels[i], c.Span)
			if !l.statements(env.fork(), c.Statements) {
				return false
			}
			l.control(Jump, exit, c.Span)
			// Later bodies must not fall back to an outer shadowed name.
			env.blockSwitchDeclarations(c.Statements)
		}
		l.control(Label, exit, n.Span)
		return true
	})
}

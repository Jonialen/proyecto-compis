package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// TypeError is the analyzer's absent-information marker. Private guarded TAC
// carries tagged values in such cells; it does not add a source-language type.
func (l *lowerer) checkedValue(expected, value Operand, span ast.Span) Operand {
	target := l.pool.acquire(expected.Type)
	target.Class, target.Shape = expected.Class, expected.Shape
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: CheckValue, Target: target, Left: value, Span: span})
	l.pool.release(value)
	return target
}

func (l *lowerer) guardedOperation(op Opcode, left Operand, right *Operand, kind model.TypeKind, span ast.Span) Operand {
	target := l.operation(op, left, right, kind, span)
	l.program.Instructions[len(l.program.Instructions)-1].Guarded = true
	return target
}

func (l *lowerer) dynamicCall(env *environment, n ast.CallExpr) (Operand, bool) {
	selected, ok := l.expression(env, n.Callee)
	if !ok {
		return Operand{}, false
	}
	selected = l.freeze(selected, n.Callee.SourceSpan())
	args := make([]Operand, 0, len(n.Arguments))
	for _, e := range n.Arguments {
		value, ok := l.expression(env, e)
		if !ok {
			return Operand{}, false
		}
		args = append(args, l.freeze(value, e.SourceSpan()))
	}
	target := l.pool.acquire(model.TypeError)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: CallIndirect, Target: target, Left: selected, Arguments: args, Guarded: true, Span: n.Span})
	l.pool.release(selected)
	for _, arg := range args {
		l.pool.release(arg)
	}
	return target, true
}

func (l *lowerer) dynamicProperty(receiver Operand, name string, span ast.Span) Operand {
	target := l.pool.acquire(model.TypeError)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: DynamicMember, Target: target, Left: receiver, Member: name, Span: span})
	l.pool.release(receiver)
	return target
}

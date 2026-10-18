package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// Cleanup is emitted on escaping edges only. The lexical depth remains intact
// while lowering unreachable source so atomic validation still visits it.
func (l *lowerer) removeHandlers(depth int, span ast.Span) {
	for n := l.handlers; n > depth; n-- {
		l.control(PopHandler, "", span)
	}
}

func (l *lowerer) tryCatch(env *environment, n ast.TryCatchStmt) bool {
	if n.Try == nil || n.Catch == nil {
		return l.unsupported(n.Span, "try/catch requires both bodies")
	}
	caught, done := l.labelName(), l.labelName()
	l.control(PushHandler, caught, n.Span)
	l.handlers++
	ok := l.statement(env, *n.Try)
	l.handlers--
	if !ok {
		return false
	}
	l.control(PopHandler, "", n.Try.Span)
	l.control(Jump, done, n.Span)
	l.control(Label, caught, n.Catch.Span)
	child := l.environment(env, model.ScopeCatch, n.Catch.Span)
	if child == nil {
		return false
	}
	var target Operand
	for _, s := range l.scopes {
		if s.ID != child.id {
			continue
		}
		for _, symbol := range s.Symbols {
			if symbol.Kind == model.SymbolCatch && symbol.Name == n.Name && symbol.Span == n.NameSpan {
				target = symbolOperand(child.id, symbol)
				child.active[n.Name] = binding{operand: target, typeOf: symbol.Type}
				l.recordStorageIdentity(child.id, target)
			}
		}
	}
	if target.Kind == "" {
		return l.unsupported(n.NameSpan, "missing matching catch binding")
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: CatchException, Target: target, Span: n.NameSpan})
	if !l.statements(child, n.Catch.Statements) {
		return false
	}
	l.control(Label, done, n.Span)
	return true
}

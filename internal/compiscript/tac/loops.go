package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type loopTargets struct {
	exit, next string
}

func (l *lowerer) loopTransfer(span ast.Span, continuing bool) bool {
	if len(l.loops) == 0 {
		return l.unsupported(span, "loop transfer requires an enclosing loop")
	}
	target := l.loops[len(l.loops)-1].exit
	if continuing {
		target = l.loops[len(l.loops)-1].next
	}
	l.control(Jump, target, span)
	return true
}

func (l *lowerer) loopBody(env *environment, body *ast.BlockStmt, exit, next string) bool {
	l.loops = append(l.loops, loopTargets{exit: exit, next: next})
	defer func() { l.loops = l.loops[:len(l.loops)-1] }()
	return l.statement(env, *body)
}

func (l *lowerer) whileStatement(env *environment, n ast.WhileStmt) bool {
	check, body, exit := l.labelName(), l.labelName(), l.labelName()
	l.control(Label, check, n.Condition.SourceSpan())
	if !l.condition(env, n.Condition, body, exit) {
		return false
	}
	l.control(Label, body, n.Body.Span)
	if !l.loopBody(env, n.Body, exit, check) {
		return false
	}
	l.control(Jump, check, n.Span)
	l.control(Label, exit, n.Span)
	return true
}

func (l *lowerer) doWhileStatement(env *environment, n ast.DoWhileStmt) bool {
	body, check, exit := l.labelName(), l.labelName(), l.labelName()
	l.control(Label, body, n.Body.Span)
	if !l.loopBody(env, n.Body, exit, check) {
		return false
	}
	l.control(Label, check, n.Condition.SourceSpan())
	if !l.condition(env, n.Condition, body, exit) {
		return false
	}
	l.control(Label, exit, n.Span)
	return true
}

func (l *lowerer) forStatement(env *environment, n ast.ForStmt) bool {
	loop := l.environment(env, model.ScopeBlock, n.Span)
	if loop == nil || (n.Init != nil && !l.statement(loop, n.Init)) {
		return false
	}
	check, body, post, exit := l.labelName(), l.labelName(), l.labelName(), l.labelName()
	l.control(Label, check, n.Span)
	if n.Condition != nil {
		if !l.condition(loop, n.Condition, body, exit) {
			return false
		}
	} else {
		l.control(Jump, body, n.Span)
	}
	l.control(Label, body, n.Body.Span)
	if !l.loopBody(loop, n.Body, exit, post) {
		return false
	}
	l.control(Label, post, n.Span)
	if n.Post != nil && !l.forPost(loop, n.Post) {
		return false
	}
	l.control(Jump, check, n.Span)
	l.control(Label, exit, n.Span)
	return true
}

// Only a whole for post may use assignment-expression syntax. Reuse statement
// validation and stores; nested assignments remain unsupported expressions.
func (l *lowerer) forPost(env *environment, expression ast.Expression) bool {
	switch n := expression.(type) {
	case ast.GroupExpr:
		return l.forPost(env, n.Expression)
	case ast.AssignExpr:
		return l.statement(env, ast.AssignStmt{Span: n.Span, Target: n.Target, Value: n.Value})
	default:
		return l.statement(env, ast.ExprStmt{Span: expression.SourceSpan(), Expression: expression})
	}
}

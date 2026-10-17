package tac

import (
	"fmt"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func (l *lowerer) labelName() string {
	name := fmt.Sprintf("L%d", l.nextLabel)
	l.nextLabel++
	return name
}

func (l *lowerer) control(op Opcode, label string, span ast.Span) {
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: op, Label: label, Span: span})
}

// condition always lowers both source operands. Runtime edges, not compile-time
// folding, decide whether the emitted RHS instructions execute.
func (l *lowerer) condition(env *environment, expression ast.Expression, yes, no string) bool {
	switch n := expression.(type) {
	case ast.GroupExpr:
		return l.condition(env, n.Expression, yes, no)
	case ast.UnaryExpr:
		if n.Operator == "!" {
			return l.condition(env, n.Operand, no, yes)
		}
	case ast.BinaryExpr:
		if n.Operator == "&&" || n.Operator == "||" {
			mid := l.labelName()
			leftYes, leftNo := mid, no
			if n.Operator == "||" {
				leftYes, leftNo = yes, mid
			}
			if !l.condition(env, n.Left, leftYes, leftNo) {
				return false
			}
			l.control(Label, mid, n.Right.SourceSpan())
			return l.condition(env, n.Right, yes, no)
		}
	}
	value, ok := l.expression(env, expression)
	if !ok {
		return false
	}
	if value.Type != model.TypeBoolean {
		return l.unsupported(expression.SourceSpan(), "condition requires a boolean value")
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: BranchTrue, Left: value, Label: yes, Span: expression.SourceSpan()})
	// Release only after the consuming branch, never before its operand read.
	l.pool.release(value)
	l.control(Jump, no, expression.SourceSpan())
	return true
}

func (l *lowerer) logicalValue(env *environment, expression ast.BinaryExpr) (Operand, bool) {
	// Reserve before recursive lowering: every incoming edge writes this same
	// result, and no RHS operation may borrow its slot before the outer consumer.
	result := l.pool.acquire(model.TypeBoolean)
	yes, no, join := l.labelName(), l.labelName(), l.labelName()
	if !l.condition(env, expression, yes, no) {
		return Operand{}, false
	}
	l.control(Label, yes, expression.Span)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: result, Left: Operand{Kind: Literal, Value: "true", Type: model.TypeBoolean}, Span: expression.Span})
	l.control(Jump, join, expression.Span)
	l.control(Label, no, expression.Span)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: result, Left: Operand{Kind: Literal, Value: "false", Type: model.TypeBoolean}, Span: expression.Span})
	l.control(Label, join, expression.Span)
	return result, true
}

func (l *lowerer) ifStatement(env *environment, statement ast.IfStmt) bool {
	yes, no := l.labelName(), l.labelName()
	if !l.condition(env, statement.Condition, yes, no) {
		return false
	}
	l.control(Label, yes, statement.Then.Span)
	if !l.statement(env, *statement.Then) {
		return false
	}
	if statement.Else == nil {
		l.control(Label, no, statement.Span)
		return true
	}
	join := l.labelName()
	l.control(Jump, join, statement.Span)
	l.control(Label, no, statement.Else.Span)
	if !l.statement(env, *statement.Else) {
		return false
	}
	l.control(Label, join, statement.Span)
	return true
}

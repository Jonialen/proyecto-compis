package frontend

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func recoverProgram(program ast.Program, diagnostics model.Diagnostics) ast.Program {
	for _, diagnostic := range diagnostics {
		if diagnostic.Phase != model.PhaseSyntax {
			continue
		}
		for i, statement := range program.Statements {
			if containsSpan(statement.SourceSpan(), diagnostic.Span) {
				program.Statements[i] = recoverStatement(statement, diagnostic.Span)
			}
		}
	}
	for i, statement := range program.Statements {
		program.Statements[i] = normalizeRecoveredStatement(statement)
	}
	return program
}

func normalizeRecoveredStatement(statement ast.Statement) ast.Statement {
	switch statement := statement.(type) {
	case ast.PrintStmt:
		statement.Value = normalizeRecoveredExpression(statement.Value)
		return statement
	case ast.ExprStmt:
		statement.Expression = normalizeRecoveredExpression(statement.Expression)
		return statement
	default:
		return statement
	}
}

func recoverStatement(statement ast.Statement, span ast.Span) ast.Statement {
	switch statement := statement.(type) {
	case ast.PrintStmt:
		statement.Value = recoverExpression(statement.Value, span)
		return statement
	case ast.ExprStmt:
		statement.Expression = recoverExpression(statement.Expression, span)
		return statement
	default:
		return ast.BadStmt{Span: statement.SourceSpan()}
	}
}

func recoverExpression(expression ast.Expression, span ast.Span) ast.Expression {
	expression = normalizeRecoveredExpression(expression)
	if !containsSpan(expression.SourceSpan(), span) {
		return expression
	}
	if expression, ok := expression.(ast.BinaryExpr); ok {
		expression.Left = recoverExpression(expression.Left, span)
		expression.Right = recoverExpression(expression.Right, span)
		return expression
	}
	return ast.BadExpr{Span: expression.SourceSpan()}
}

func normalizeRecoveredExpression(expression ast.Expression) ast.Expression {
	switch expression := expression.(type) {
	case ast.BinaryExpr:
		expression.Left = normalizeRecoveredExpression(expression.Left)
		expression.Right = normalizeRecoveredExpression(expression.Right)
		return expression
	case ast.UnaryExpr:
		if expression.Operator == "" {
			return ast.BadExpr{Span: expression.Span}
		}
	}
	return expression
}

func containsSpan(outer, inner ast.Span) bool {
	return outer.Start.Offset <= inner.Start.Offset && inner.End.Offset <= outer.End.Offset
}

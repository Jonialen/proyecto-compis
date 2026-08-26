package frontend

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
)

type expressionMapper struct {
	index sourceIndex
}

func newExpressionMapper(source []byte) expressionMapper {
	return expressionMapper{index: newSourceIndex(source)}
}

func (m expressionMapper) mapExpression(ctx generated.IExpressionContext) ast.Expression {
	return m.mapTree(ctx)
}

func (m expressionMapper) mapType(ctx generated.ITypeContext) ast.TypeRef {
	text := ctx.GetText()
	name := strings.TrimSuffix(text, strings.Repeat("[]", strings.Count(text, "[]")))
	return ast.TypeRef{Span: m.span(ctx), Name: name, Dimensions: strings.Count(text, "[]")}
}

func (m expressionMapper) mapTree(tree antlr.Tree) ast.Expression {
	switch ctx := tree.(type) {
	case *generated.ExpressionContext, *generated.ExprNoAssignContext:
		return m.mapTree(tree.GetChild(0))
	case *generated.AssignExprContext:
		target, value := m.mapTree(ctx.GetChild(0)), m.mapTree(ctx.GetChild(2))
		if property, ok := target.(ast.PropertyAccessExpr); ok {
			return ast.PropertyAssignExpr{Span: m.span(ctx), Receiver: property.Receiver, Name: property.Name, Value: value}
		}
		return ast.AssignExpr{Span: m.span(ctx), Target: target, Value: value}
	case *generated.PropertyAssignExprContext:
		return ast.PropertyAssignExpr{Span: m.span(ctx), Receiver: m.mapTree(ctx.GetChild(0)), Name: treeText(ctx.GetChild(2)), Value: m.mapTree(ctx.GetChild(4))}
	case *generated.TernaryExprContext:
		if ctx.GetChildCount() == 1 {
			return m.mapTree(ctx.GetChild(0))
		}
		return ast.TernaryExpr{Span: m.span(ctx), Condition: m.mapTree(ctx.GetChild(0)), Then: m.mapTree(ctx.GetChild(2)), Else: m.mapTree(ctx.GetChild(4))}
	case *generated.LogicalOrExprContext, *generated.LogicalAndExprContext, *generated.EqualityExprContext, *generated.RelationalExprContext, *generated.AdditiveExprContext, *generated.MultiplicativeExprContext:
		return m.foldBinary(tree)
	case *generated.UnaryExprContext:
		if ctx.GetChildCount() == 1 {
			return m.mapTree(ctx.GetChild(0))
		}
		return ast.UnaryExpr{Span: m.span(ctx), Operator: treeText(ctx.GetChild(0)), Operand: m.mapTree(ctx.GetChild(1))}
	case *generated.PrimaryExprContext:
		if treeText(ctx.GetChild(0)) == "(" {
			return ast.GroupExpr{Span: m.span(ctx), Expression: m.mapTree(ctx.GetChild(1))}
		}
		return m.mapTree(ctx.GetChild(0))
	case *generated.LiteralExprContext:
		if _, ok := ctx.GetChild(0).(*generated.ArrayLiteralContext); ok {
			return m.mapTree(ctx.GetChild(0))
		}
		return ast.LiteralExpr{Span: m.span(ctx), Lexeme: ctx.GetText()}
	case *generated.LeftHandSideContext:
		expression := m.mapTree(ctx.GetChild(0))
		for i := 1; i < ctx.GetChildCount(); i++ {
			expression = m.applySuffix(expression, ctx.GetChild(i))
		}
		return expression
	case *generated.IdentifierExprContext:
		return ast.IdentifierExpr{Span: m.span(ctx), Name: ctx.GetText()}
	case *generated.NewExprContext:
		return ast.NewExpr{Span: m.span(ctx), ClassName: treeText(ctx.GetChild(1)), Arguments: m.arguments(ctx)}
	case *generated.ThisExprContext:
		return ast.ThisExpr{Span: m.span(ctx)}
	case *generated.ArrayLiteralContext:
		return ast.ArrayExpr{Span: m.span(ctx), Elements: m.expressions(ctx)}
	default:
		return ast.BadExpr{Span: m.span(tree)}
	}
}

func (m expressionMapper) foldBinary(tree antlr.Tree) ast.Expression {
	expression := m.mapTree(tree.GetChild(0))
	for i := 1; i < tree.GetChildCount(); i += 2 {
		expression = ast.BinaryExpr{Span: m.span(tree), Left: expression, Operator: treeText(tree.GetChild(i)), Right: m.mapTree(tree.GetChild(i + 1))}
	}
	return expression
}

func (m expressionMapper) applySuffix(receiver ast.Expression, tree antlr.Tree) ast.Expression {
	switch ctx := tree.(type) {
	case *generated.CallExprContext:
		return ast.CallExpr{Span: m.suffixSpan(receiver, ctx), Callee: receiver, Arguments: m.arguments(ctx)}
	case *generated.IndexExprContext:
		return ast.IndexExpr{Span: m.suffixSpan(receiver, ctx), Collection: receiver, Index: m.mapTree(ctx.GetChild(1))}
	case *generated.PropertyAccessExprContext:
		return ast.PropertyAccessExpr{Span: m.suffixSpan(receiver, ctx), Receiver: receiver, Name: treeText(ctx.GetChild(1))}
	default:
		return ast.BadExpr{Span: m.span(tree)}
	}
}

func (m expressionMapper) suffixSpan(receiver ast.Expression, suffix antlr.Tree) ast.Span {
	span := m.span(suffix)
	span.Start = receiver.SourceSpan().Start
	return span
}

func (m expressionMapper) arguments(tree antlr.Tree) ast.Expressions {
	for i := 0; i < tree.GetChildCount(); i++ {
		if arguments, ok := tree.GetChild(i).(*generated.ArgumentsContext); ok {
			return m.expressions(arguments)
		}
	}
	return nil
}

func (m expressionMapper) expressions(tree antlr.Tree) ast.Expressions {
	var expressions ast.Expressions
	for i := 0; i < tree.GetChildCount(); i++ {
		if expression, ok := tree.GetChild(i).(generated.IExpressionContext); ok {
			expressions = append(expressions, m.mapExpression(expression))
		}
	}
	return expressions
}

func (m expressionMapper) span(tree antlr.Tree) ast.Span {
	context, ok := tree.(antlr.ParserRuleContext)
	if !ok || context.GetStart() == nil {
		return ast.Span{}
	}
	stop := context.GetStop()
	if stop == nil {
		stop = context.GetStart()
	}
	return m.index.spanFromScalars(context.GetStart().GetStart(), stop.GetStop())
}

func treeText(tree antlr.Tree) string {
	if value, ok := tree.(interface{ GetText() string }); ok {
		return value.GetText()
	}
	return ""
}

package frontend

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
)

type expressionMapper struct {
	*generated.BaseCompiscriptVisitor
	index          sourceIndex
	suffixReceiver ast.Expression
}

var _ generated.CompiscriptVisitor = (*expressionMapper)(nil)

func newExpressionMapper(source []byte) *expressionMapper {
	return &expressionMapper{
		BaseCompiscriptVisitor: &generated.BaseCompiscriptVisitor{BaseParseTreeVisitor: &antlr.BaseParseTreeVisitor{}},
		index:                  newSourceIndex(source),
	}
}

func (m *expressionMapper) mapExpression(ctx generated.IExpressionContext) ast.Expression {
	return m.expression(ctx)
}

func (m *expressionMapper) mapType(ctx generated.ITypeContext) ast.TypeRef {
	if mapped, ok := ctx.(antlr.ParseTree).Accept(m).(ast.TypeRef); ok {
		return mapped
	}
	return ast.TypeRef{Span: m.span(ctx)}
}

func (m *expressionMapper) expression(tree antlr.Tree) ast.Expression {
	parseTree, ok := tree.(antlr.ParseTree)
	if !ok {
		return ast.BadExpr{Span: m.span(tree)}
	}
	mapped, ok := parseTree.Accept(m).(ast.Expression)
	if !ok || mapped == nil {
		return ast.BadExpr{Span: m.span(tree)}
	}
	return mapped
}

func (m *expressionMapper) VisitExpression(ctx *generated.ExpressionContext) interface{} {
	return m.expression(ctx.GetChild(0))
}

func (m *expressionMapper) VisitAssignExpr(ctx *generated.AssignExprContext) interface{} {
	target, value := m.expression(ctx.GetChild(0)), m.expression(ctx.GetChild(2))
	if property, ok := target.(ast.PropertyAccessExpr); ok {
		return ast.PropertyAssignExpr{Span: m.span(ctx), Receiver: property.Receiver, Name: property.Name, Value: value}
	}
	return ast.AssignExpr{Span: m.span(ctx), Target: target, Value: value}
}

func (m *expressionMapper) VisitPropertyAssignExpr(ctx *generated.PropertyAssignExprContext) interface{} {
	return ast.PropertyAssignExpr{Span: m.span(ctx), Receiver: m.expression(ctx.GetChild(0)), Name: treeText(ctx.GetChild(2)), Value: m.expression(ctx.GetChild(4))}
}

func (m *expressionMapper) VisitExprNoAssign(ctx *generated.ExprNoAssignContext) interface{} {
	return m.expression(ctx.GetChild(0))
}

func (m *expressionMapper) VisitTernaryExpr(ctx *generated.TernaryExprContext) interface{} {
	if ctx.GetChildCount() == 1 {
		return m.expression(ctx.GetChild(0))
	}
	return ast.TernaryExpr{Span: m.span(ctx), Condition: m.expression(ctx.GetChild(0)), Then: m.expression(ctx.GetChild(2)), Else: m.expression(ctx.GetChild(4))}
}

func (m *expressionMapper) VisitLogicalOrExpr(ctx *generated.LogicalOrExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitLogicalAndExpr(ctx *generated.LogicalAndExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitEqualityExpr(ctx *generated.EqualityExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitRelationalExpr(ctx *generated.RelationalExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitAdditiveExpr(ctx *generated.AdditiveExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitMultiplicativeExpr(ctx *generated.MultiplicativeExprContext) interface{} {
	return m.foldBinary(ctx)
}

func (m *expressionMapper) VisitUnaryExpr(ctx *generated.UnaryExprContext) interface{} {
	if ctx.GetChildCount() == 1 {
		return m.expression(ctx.GetChild(0))
	}
	return ast.UnaryExpr{Span: m.span(ctx), Operator: treeText(ctx.GetChild(0)), Operand: m.expression(ctx.GetChild(1))}
}

func (m *expressionMapper) VisitPrimaryExpr(ctx *generated.PrimaryExprContext) interface{} {
	if treeText(ctx.GetChild(0)) == "(" {
		return ast.GroupExpr{Span: m.span(ctx), Expression: m.expression(ctx.GetChild(1))}
	}
	return m.expression(ctx.GetChild(0))
}

func (m *expressionMapper) VisitLiteralExpr(ctx *generated.LiteralExprContext) interface{} {
	if array, ok := ctx.GetChild(0).(*generated.ArrayLiteralContext); ok {
		return array.Accept(m)
	}
	return ast.LiteralExpr{Span: m.span(ctx), Lexeme: ctx.GetText()}
}

func (m *expressionMapper) VisitLeftHandSide(ctx *generated.LeftHandSideContext) interface{} {
	expression := m.expression(ctx.GetChild(0))
	for i := 1; i < ctx.GetChildCount(); i++ {
		previous := m.suffixReceiver
		m.suffixReceiver = expression
		expression = m.expression(ctx.GetChild(i))
		m.suffixReceiver = previous
	}
	return expression
}

func (m *expressionMapper) VisitIdentifierExpr(ctx *generated.IdentifierExprContext) interface{} {
	return ast.IdentifierExpr{Span: m.span(ctx), Name: ctx.GetText()}
}

func (m *expressionMapper) VisitNewExpr(ctx *generated.NewExprContext) interface{} {
	return ast.NewExpr{Span: m.span(ctx), ClassName: treeText(ctx.GetChild(1)), Arguments: m.arguments(ctx)}
}

func (m *expressionMapper) VisitThisExpr(ctx *generated.ThisExprContext) interface{} {
	return ast.ThisExpr{Span: m.span(ctx)}
}

func (m *expressionMapper) VisitCallExpr(ctx *generated.CallExprContext) interface{} {
	receiver := m.suffixReceiver
	return ast.CallExpr{Span: m.suffixSpan(receiver, ctx), Callee: receiver, Arguments: m.arguments(ctx)}
}

func (m *expressionMapper) VisitIndexExpr(ctx *generated.IndexExprContext) interface{} {
	receiver := m.suffixReceiver
	return ast.IndexExpr{Span: m.suffixSpan(receiver, ctx), Collection: receiver, Index: m.expression(ctx.GetChild(1))}
}

func (m *expressionMapper) VisitPropertyAccessExpr(ctx *generated.PropertyAccessExprContext) interface{} {
	receiver := m.suffixReceiver
	return ast.PropertyAccessExpr{Span: m.suffixSpan(receiver, ctx), Receiver: receiver, Name: treeText(ctx.GetChild(1))}
}

func (m *expressionMapper) VisitArrayLiteral(ctx *generated.ArrayLiteralContext) interface{} {
	return ast.ArrayExpr{Span: m.span(ctx), Elements: m.expressions(ctx)}
}

func (m *expressionMapper) VisitType(ctx *generated.TypeContext) interface{} {
	text := ctx.GetText()
	name := strings.TrimSuffix(text, strings.Repeat("[]", strings.Count(text, "[]")))
	return ast.TypeRef{Span: m.span(ctx), Name: name, Dimensions: strings.Count(text, "[]")}
}

func (m *expressionMapper) foldBinary(tree antlr.Tree) ast.Expression {
	expression := m.expression(tree.GetChild(0))
	for i := 1; i < tree.GetChildCount(); i += 2 {
		expression = ast.BinaryExpr{Span: m.span(tree), Left: expression, Operator: treeText(tree.GetChild(i)), Right: m.expression(tree.GetChild(i + 1))}
	}
	return expression
}

func (m *expressionMapper) suffixSpan(receiver ast.Expression, suffix antlr.Tree) ast.Span {
	span := m.span(suffix)
	span.Start = receiver.SourceSpan().Start
	return span
}

func (m *expressionMapper) arguments(tree antlr.Tree) ast.Expressions {
	for i := 0; i < tree.GetChildCount(); i++ {
		if arguments, ok := tree.GetChild(i).(*generated.ArgumentsContext); ok {
			return m.expressions(arguments)
		}
	}
	return nil
}

func (m *expressionMapper) expressions(tree antlr.Tree) (expressions ast.Expressions) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if expression, ok := tree.GetChild(i).(generated.IExpressionContext); ok {
			expressions = append(expressions, m.mapExpression(expression))
		}
	}
	return expressions
}

func (m *expressionMapper) span(tree antlr.Tree) ast.Span {
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

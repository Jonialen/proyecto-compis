package frontend

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
)

type statementMapper struct{ *expressionMapper }

var _ generated.CompiscriptVisitor = (*statementMapper)(nil)

func newStatementMapper(source []byte) *statementMapper {
	return &statementMapper{newExpressionMapper(source)}
}

func (m *statementMapper) mapProgram(ctx generated.IProgramContext) ast.Program {
	if program, ok := ctx.(antlr.ParseTree).Accept(m).(ast.Program); ok {
		return program
	}
	return ast.Program{Span: m.span(ctx)}
}

func (m *statementMapper) VisitProgram(ctx *generated.ProgramContext) interface{} {
	return ast.Program{Span: m.span(ctx), Statements: m.statements(ctx)}
}

func (m *statementMapper) VisitStatement(ctx *generated.StatementContext) interface{} {
	if ctx.GetChildCount() == 0 {
		return ast.BadStmt{Span: m.span(ctx)}
	}
	return m.visitStatementTree(ctx.GetChild(0))
}

func (m *statementMapper) VisitBlock(ctx *generated.BlockContext) interface{} {
	return ast.BlockStmt{Span: m.span(ctx), Statements: m.statements(ctx)}
}

func (m *statementMapper) VisitVariableDeclaration(ctx *generated.VariableDeclarationContext) interface{} {
	return ast.VarDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Type: m.typeRef(ctx), Initializer: m.firstExpression(ctx)}
}

func (m *statementMapper) VisitConstantDeclaration(ctx *generated.ConstantDeclarationContext) interface{} {
	return ast.ConstDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Type: m.typeRef(ctx), Value: m.firstExpression(ctx)}
}

func (m *statementMapper) VisitAssignment(ctx *generated.AssignmentContext) interface{} {
	expressions := m.directExpressions(ctx)
	if len(expressions) == 1 {
		return ast.AssignStmt{Span: m.span(ctx), Target: ast.IdentifierExpr{Span: m.span(ctx), Name: m.identifiers(ctx)[0]}, Value: expressions[0]}
	}
	name := ctx.Identifier()
	targetSpan := expressions[0].SourceSpan()
	targetSpan.End = m.terminalSpan(name).End
	return ast.AssignStmt{Span: m.span(ctx), Target: ast.PropertyAccessExpr{Span: targetSpan, Receiver: expressions[0], Name: name.GetText()}, Value: expressions[1]}
}

func (m *statementMapper) VisitExpressionStatement(ctx *generated.ExpressionStatementContext) interface{} {
	return ast.ExprStmt{Span: m.span(ctx), Expression: m.firstExpression(ctx)}
}

func (m *statementMapper) VisitPrintStatement(ctx *generated.PrintStatementContext) interface{} {
	return ast.PrintStmt{Span: m.span(ctx), Value: m.firstExpression(ctx)}
}

func (m *statementMapper) VisitIfStatement(ctx *generated.IfStatementContext) interface{} {
	bodies := m.bodies(ctx)
	statement := ast.IfStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Then: bodies[0]}
	if len(bodies) > 1 {
		statement.Else = bodies[1]
	}
	return statement
}

func (m *statementMapper) VisitWhileStatement(ctx *generated.WhileStatementContext) interface{} {
	return ast.WhileStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
}

func (m *statementMapper) VisitDoWhileStatement(ctx *generated.DoWhileStatementContext) interface{} {
	return ast.DoWhileStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
}

func (m *statementMapper) VisitForStatement(ctx *generated.ForStatementContext) interface{} {
	statement := ast.ForStmt{Span: m.span(ctx), Body: m.firstBlock(ctx)}
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if _, variable := child.(*generated.VariableDeclarationContext); variable && statement.Init == nil {
			statement.Init = m.visitStatementTree(child)
		}
		if _, assignment := child.(*generated.AssignmentContext); assignment && statement.Init == nil {
			statement.Init = m.visitStatementTree(child)
		}
	}
	expressions := m.directExpressions(ctx)
	if len(expressions) > 0 {
		statement.Condition = expressions[0]
	}
	if len(expressions) > 1 {
		statement.Post = expressions[1]
	}
	return statement
}

func (m *statementMapper) VisitForeachStatement(ctx *generated.ForeachStatementContext) interface{} {
	return ast.ForeachStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Iterable: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
}

func (m *statementMapper) VisitBreakStatement(ctx *generated.BreakStatementContext) interface{} {
	return ast.BreakStmt{Span: m.span(ctx)}
}

func (m *statementMapper) VisitContinueStatement(ctx *generated.ContinueStatementContext) interface{} {
	return ast.ContinueStmt{Span: m.span(ctx)}
}

func (m *statementMapper) VisitReturnStatement(ctx *generated.ReturnStatementContext) interface{} {
	return ast.ReturnStmt{Span: m.span(ctx), Value: m.firstExpression(ctx)}
}

func (m *statementMapper) VisitTryCatchStatement(ctx *generated.TryCatchStatementContext) interface{} {
	blocks := m.blocks(ctx)
	name := ctx.Identifier()
	return ast.TryCatchStmt{Span: m.span(ctx), Try: blocks[0], Name: name.GetText(), NameSpan: m.terminalSpan(name), Catch: blocks[1]}
}

func (m *statementMapper) VisitSwitchStatement(ctx *generated.SwitchStatementContext) interface{} {
	return ast.SwitchStmt{Span: m.span(ctx), Value: m.firstExpression(ctx), Cases: m.cases(ctx)}
}

func (m *statementMapper) VisitSwitchCase(ctx *generated.SwitchCaseContext) interface{} {
	return ast.SwitchCase{Span: m.span(ctx), Value: m.firstExpression(ctx), Statements: m.statements(ctx)}
}

func (m *statementMapper) VisitDefaultCase(ctx *generated.DefaultCaseContext) interface{} {
	return ast.SwitchCase{Span: m.span(ctx), Default: true, Statements: m.statements(ctx)}
}

func (m *statementMapper) VisitFunctionDeclaration(ctx *generated.FunctionDeclarationContext) interface{} {
	return ast.FunctionDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Parameters: m.parameters(ctx.Parameters()), Result: m.directType(ctx), Body: m.firstBlock(ctx)}
}

func (m *statementMapper) VisitClassDeclaration(ctx *generated.ClassDeclarationContext) interface{} {
	ids := m.identifiers(ctx)
	class := ast.ClassDeclStmt{Span: m.span(ctx), Name: ids[0]}
	if len(ids) > 1 {
		class.Parent = ids[1]
	}
	for i := 0; i < ctx.GetChildCount(); i++ {
		if member, ok := ctx.GetChild(i).(*generated.ClassMemberContext); ok {
			class.Members = append(class.Members, m.visitStatementTree(member))
		}
	}
	return class
}

func (m *statementMapper) VisitClassMember(ctx *generated.ClassMemberContext) interface{} {
	if ctx.GetChildCount() == 0 {
		return ast.BadStmt{Span: m.span(ctx)}
	}
	return m.visitStatementTree(ctx.GetChild(0))
}

func (m *statementMapper) visitStatementTree(tree antlr.Tree) ast.Statement {
	parseTree, ok := tree.(antlr.ParseTree)
	if !ok {
		return ast.BadStmt{Span: m.span(tree)}
	}
	statement, ok := parseTree.Accept(m).(ast.Statement)
	if !ok || statement == nil {
		return ast.BadStmt{Span: m.span(tree)}
	}
	return statement
}

func (m *statementMapper) statements(tree antlr.Tree) (statements ast.Statements) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.IStatementContext); ok {
			statements = append(statements, m.visitStatementTree(ctx))
		}
	}
	return statements
}

func (m *statementMapper) block(ctx *generated.BlockContext) *ast.BlockStmt {
	mapped := ctx.Accept(m).(ast.BlockStmt)
	return &mapped
}

func (m *statementMapper) bodies(tree antlr.Tree) (bodies []*ast.BlockStmt) {
	for i := 0; i < tree.GetChildCount(); i++ {
		switch child := tree.GetChild(i).(type) {
		case *generated.BlockContext:
			bodies = append(bodies, m.block(child))
		case generated.IStatementContext:
			bodies = append(bodies, &ast.BlockStmt{Span: m.span(child), Statements: ast.Statements{m.visitStatementTree(child)}})
		}
	}
	return bodies
}

func (m *statementMapper) blocks(tree antlr.Tree) (blocks []*ast.BlockStmt) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(*generated.BlockContext); ok {
			blocks = append(blocks, m.block(ctx))
		}
	}
	return blocks
}

func (m *statementMapper) firstBlock(tree antlr.Tree) *ast.BlockStmt { return m.blocks(tree)[0] }

func (m *statementMapper) firstExpression(tree antlr.Tree) ast.Expression {
	if expressions := m.directExpressions(tree); len(expressions) > 0 {
		return expressions[0]
	}
	for i := 0; i < tree.GetChildCount(); i++ {
		if child, ok := tree.GetChild(i).(antlr.ParserRuleContext); ok {
			if expression := m.firstExpression(child); expression != nil {
				return expression
			}
		}
	}
	return nil
}

func (m *statementMapper) directExpressions(tree antlr.Tree) (expressions ast.Expressions) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.IExpressionContext); ok {
			expressions = append(expressions, m.mapExpression(ctx))
		}
	}
	return expressions
}

func (m *statementMapper) typeRef(tree antlr.Tree) *ast.TypeRef {
	if ctx, ok := tree.(generated.ITypeContext); ok {
		typeRef := m.mapType(ctx)
		return &typeRef
	}
	for i := 0; i < tree.GetChildCount(); i++ {
		if child, ok := tree.GetChild(i).(antlr.ParserRuleContext); ok {
			if typeRef := m.typeRef(child); typeRef != nil {
				return typeRef
			}
		}
	}
	return nil
}

func (m *statementMapper) directType(tree antlr.Tree) *ast.TypeRef {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.ITypeContext); ok {
			typeRef := m.mapType(ctx)
			return &typeRef
		}
	}
	return nil
}

func (m *statementMapper) identifiers(tree antlr.Tree) (identifiers []string) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if node, ok := tree.GetChild(i).(antlr.TerminalNode); ok && node.GetSymbol().GetTokenType() == generated.CompiscriptParserIdentifier {
			identifiers = append(identifiers, node.GetText())
		}
	}
	return identifiers
}

func (m *statementMapper) terminalSpan(node antlr.TerminalNode) ast.Span {
	token := node.GetSymbol()
	return m.index.spanFromScalars(token.GetStart(), token.GetStop())
}

func (m *statementMapper) parameters(tree antlr.Tree) (parameters ast.Parameters) {
	if tree == nil {
		return nil
	}
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(*generated.ParameterContext); ok {
			parameter := ast.Parameter{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Type: m.typeRef(ctx)}
			parameters = append(parameters, parameter)
			continue
		}
		if child, ok := tree.GetChild(i).(antlr.ParserRuleContext); ok {
			parameters = append(parameters, m.parameters(child)...)
		}
	}
	return parameters
}

func (m *statementMapper) cases(tree antlr.Tree) (cases ast.SwitchCases) {
	for i := 0; i < tree.GetChildCount(); i++ {
		parseTree, ok := tree.GetChild(i).(antlr.ParseTree)
		if !ok {
			continue
		}
		switchCase, ok := parseTree.Accept(m).(ast.SwitchCase)
		if ok {
			cases = append(cases, switchCase)
		}
	}
	return cases
}

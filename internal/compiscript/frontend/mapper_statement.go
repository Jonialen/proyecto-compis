package frontend

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
)

type statementMapper struct{ expressionMapper }

func newStatementMapper(source []byte) statementMapper {
	return statementMapper{newExpressionMapper(source)}
}

func (m statementMapper) mapProgram(ctx generated.IProgramContext) ast.Program {
	return ast.Program{Span: m.span(ctx), Statements: m.statements(ctx)}
}

func (m statementMapper) statements(tree antlr.Tree) (statements ast.Statements) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.IStatementContext); ok {
			statements = append(statements, m.mapStatement(ctx))
		}
	}
	return statements
}

func (m statementMapper) mapStatement(ctx generated.IStatementContext) ast.Statement {
	return m.mapStatementTree(ctx)
}

func (m statementMapper) mapStatementTree(tree antlr.Tree) ast.Statement {
	switch ctx := tree.(type) {
	case *generated.StatementContext:
		return m.mapStatementTree(ctx.GetChild(0))
	case *generated.VariableDeclarationContext:
		return ast.VarDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Type: m.typeRef(ctx), Initializer: m.firstExpression(ctx)}
	case *generated.ConstantDeclarationContext:
		return ast.ConstDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Type: m.typeRef(ctx), Value: m.firstExpression(ctx)}
	case *generated.AssignmentContext:
		expressions := m.directExpressions(ctx)
		if len(expressions) == 1 {
			return ast.AssignStmt{Span: m.span(ctx), Target: ast.IdentifierExpr{Span: m.span(ctx), Name: m.identifiers(ctx)[0]}, Value: expressions[0]}
		}
		return ast.AssignStmt{Span: m.span(ctx), Target: expressions[0], Value: expressions[1]}
	case *generated.FunctionDeclarationContext:
		return ast.FunctionDeclStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Parameters: m.parameters(ctx), Result: m.directType(ctx), Body: m.firstBlock(ctx)}
	case *generated.ClassDeclarationContext:
		ids := m.identifiers(ctx)
		class := ast.ClassDeclStmt{Span: m.span(ctx), Name: ids[0]}
		if len(ids) > 1 {
			class.Parent = ids[1]
		}
		for i := 0; i < ctx.GetChildCount(); i++ {
			if member, ok := ctx.GetChild(i).(*generated.ClassMemberContext); ok {
				class.Members = append(class.Members, m.mapStatementTree(member.GetChild(0)))
			}
		}
		return class
	case *generated.ExpressionStatementContext:
		return ast.ExprStmt{Span: m.span(ctx), Expression: m.firstExpression(ctx)}
	case *generated.PrintStatementContext:
		return ast.PrintStmt{Span: m.span(ctx), Value: m.firstExpression(ctx)}
	case *generated.BlockContext:
		return *m.block(ctx)
	case *generated.IfStatementContext:
		bodies := m.bodies(ctx)
		stmt := ast.IfStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Then: bodies[0]}
		if len(bodies) > 1 {
			stmt.Else = bodies[1]
		}
		return stmt
	case *generated.WhileStatementContext:
		return ast.WhileStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
	case *generated.DoWhileStatementContext:
		return ast.DoWhileStmt{Span: m.span(ctx), Condition: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
	case *generated.ForStatementContext:
		stmt := ast.ForStmt{Span: m.span(ctx), Body: m.firstBlock(ctx)}
		for i := 0; i < ctx.GetChildCount(); i++ {
			switch child := ctx.GetChild(i).(type) {
			case *generated.VariableDeclarationContext, *generated.AssignmentContext:
				if stmt.Init == nil {
					stmt.Init = m.mapStatementTree(child)
				}
			}
		}
		expressions := m.directExpressions(ctx)
		if len(expressions) > 0 {
			stmt.Condition = expressions[0]
		}
		if len(expressions) > 1 {
			stmt.Post = expressions[1]
		}
		return stmt
	case *generated.ForeachStatementContext:
		return ast.ForeachStmt{Span: m.span(ctx), Name: m.identifiers(ctx)[0], Iterable: m.firstExpression(ctx), Body: m.firstBlock(ctx)}
	case *generated.TryCatchStatementContext:
		blocks := m.blocks(ctx)
		return ast.TryCatchStmt{Span: m.span(ctx), Try: blocks[0], Name: m.identifiers(ctx)[0], Catch: blocks[1]}
	case *generated.SwitchStatementContext:
		return ast.SwitchStmt{Span: m.span(ctx), Value: m.firstExpression(ctx), Cases: m.cases(ctx)}
	case *generated.BreakStatementContext:
		return ast.BreakStmt{Span: m.span(ctx)}
	case *generated.ContinueStatementContext:
		return ast.ContinueStmt{Span: m.span(ctx)}
	case *generated.ReturnStatementContext:
		return ast.ReturnStmt{Span: m.span(ctx), Value: m.firstExpression(ctx)}
	default:
		return ast.BadStmt{Span: m.span(tree)}
	}
}

func (m statementMapper) block(ctx *generated.BlockContext) *ast.BlockStmt {
	return &ast.BlockStmt{Span: m.span(ctx), Statements: m.statements(ctx)}
}

func (m statementMapper) bodies(tree antlr.Tree) (bodies []*ast.BlockStmt) {
	for i := 0; i < tree.GetChildCount(); i++ {
		switch child := tree.GetChild(i).(type) {
		case *generated.BlockContext:
			bodies = append(bodies, m.block(child))
		case generated.IStatementContext:
			bodies = append(bodies, &ast.BlockStmt{Span: m.span(child), Statements: ast.Statements{m.mapStatement(child)}})
		}
	}
	return bodies
}

func (m statementMapper) blocks(tree antlr.Tree) (blocks []*ast.BlockStmt) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(*generated.BlockContext); ok {
			blocks = append(blocks, m.block(ctx))
		}
	}
	return blocks
}

func (m statementMapper) firstBlock(tree antlr.Tree) *ast.BlockStmt { return m.blocks(tree)[0] }

func (m statementMapper) firstExpression(tree antlr.Tree) ast.Expression {
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

func (m statementMapper) directExpressions(tree antlr.Tree) (expressions ast.Expressions) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.IExpressionContext); ok {
			expressions = append(expressions, m.mapExpression(ctx))
		}
	}
	return expressions
}

func (m statementMapper) typeRef(tree antlr.Tree) *ast.TypeRef {
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

func (m statementMapper) directType(tree antlr.Tree) *ast.TypeRef {
	for i := 0; i < tree.GetChildCount(); i++ {
		if ctx, ok := tree.GetChild(i).(generated.ITypeContext); ok {
			typeRef := m.mapType(ctx)
			return &typeRef
		}
	}
	return nil
}

func (m statementMapper) identifiers(tree antlr.Tree) (identifiers []string) {
	for i := 0; i < tree.GetChildCount(); i++ {
		if node, ok := tree.GetChild(i).(antlr.TerminalNode); ok && node.GetSymbol().GetTokenType() == generated.CompiscriptParserIdentifier {
			identifiers = append(identifiers, node.GetText())
		}
	}
	return identifiers
}

func (m statementMapper) parameters(tree antlr.Tree) (parameters ast.Parameters) {
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

func (m statementMapper) cases(tree antlr.Tree) (cases ast.SwitchCases) {
	for i := 0; i < tree.GetChildCount(); i++ {
		child := tree.GetChild(i)
		caseCtx, regular := child.(*generated.SwitchCaseContext)
		_, fallback := child.(*generated.DefaultCaseContext)
		if !regular && !fallback {
			continue
		}
		mapped := ast.SwitchCase{Span: m.span(child), Default: fallback, Statements: m.statements(child)}
		if regular {
			mapped.Value = m.firstExpression(caseCtx)
		}
		cases = append(cases, mapped)
	}
	return cases
}

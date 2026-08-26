// Code generated from /home/jonialen/Documents/uvg/s8/compis/proyecto-compis/docs/semestre2/entrega1/Compiscript.g4 by ANTLR 4.13.2. DO NOT EDIT.

package generated // Compiscript
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by CompiscriptParser.
type CompiscriptVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by CompiscriptParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#block.
	VisitBlock(ctx *BlockContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#variableDeclaration.
	VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#constantDeclaration.
	VisitConstantDeclaration(ctx *ConstantDeclarationContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#typeAnnotation.
	VisitTypeAnnotation(ctx *TypeAnnotationContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#initializer.
	VisitInitializer(ctx *InitializerContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#assignment.
	VisitAssignment(ctx *AssignmentContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#expressionStatement.
	VisitExpressionStatement(ctx *ExpressionStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#printStatement.
	VisitPrintStatement(ctx *PrintStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#whileStatement.
	VisitWhileStatement(ctx *WhileStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#doWhileStatement.
	VisitDoWhileStatement(ctx *DoWhileStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#forStatement.
	VisitForStatement(ctx *ForStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#foreachStatement.
	VisitForeachStatement(ctx *ForeachStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#breakStatement.
	VisitBreakStatement(ctx *BreakStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#continueStatement.
	VisitContinueStatement(ctx *ContinueStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#returnStatement.
	VisitReturnStatement(ctx *ReturnStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#tryCatchStatement.
	VisitTryCatchStatement(ctx *TryCatchStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#switchStatement.
	VisitSwitchStatement(ctx *SwitchStatementContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#switchCase.
	VisitSwitchCase(ctx *SwitchCaseContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#defaultCase.
	VisitDefaultCase(ctx *DefaultCaseContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#functionDeclaration.
	VisitFunctionDeclaration(ctx *FunctionDeclarationContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#parameters.
	VisitParameters(ctx *ParametersContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#parameter.
	VisitParameter(ctx *ParameterContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#classDeclaration.
	VisitClassDeclaration(ctx *ClassDeclarationContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#classMember.
	VisitClassMember(ctx *ClassMemberContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#AssignExpr.
	VisitAssignExpr(ctx *AssignExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#PropertyAssignExpr.
	VisitPropertyAssignExpr(ctx *PropertyAssignExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#ExprNoAssign.
	VisitExprNoAssign(ctx *ExprNoAssignContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#TernaryExpr.
	VisitTernaryExpr(ctx *TernaryExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#logicalOrExpr.
	VisitLogicalOrExpr(ctx *LogicalOrExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#logicalAndExpr.
	VisitLogicalAndExpr(ctx *LogicalAndExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#equalityExpr.
	VisitEqualityExpr(ctx *EqualityExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#relationalExpr.
	VisitRelationalExpr(ctx *RelationalExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#additiveExpr.
	VisitAdditiveExpr(ctx *AdditiveExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#multiplicativeExpr.
	VisitMultiplicativeExpr(ctx *MultiplicativeExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#unaryExpr.
	VisitUnaryExpr(ctx *UnaryExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#primaryExpr.
	VisitPrimaryExpr(ctx *PrimaryExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#literalExpr.
	VisitLiteralExpr(ctx *LiteralExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#leftHandSide.
	VisitLeftHandSide(ctx *LeftHandSideContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#IdentifierExpr.
	VisitIdentifierExpr(ctx *IdentifierExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#NewExpr.
	VisitNewExpr(ctx *NewExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#ThisExpr.
	VisitThisExpr(ctx *ThisExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#CallExpr.
	VisitCallExpr(ctx *CallExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#IndexExpr.
	VisitIndexExpr(ctx *IndexExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#PropertyAccessExpr.
	VisitPropertyAccessExpr(ctx *PropertyAccessExprContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#arguments.
	VisitArguments(ctx *ArgumentsContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#arrayLiteral.
	VisitArrayLiteral(ctx *ArrayLiteralContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#type.
	VisitType(ctx *TypeContext) interface{}

	// Visit a parse tree produced by CompiscriptParser#baseType.
	VisitBaseType(ctx *BaseTypeContext) interface{}
}

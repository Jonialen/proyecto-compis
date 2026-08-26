package compiscript

import (
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

func astView(program ast.Program) model.ASTView { return nodeView(program) }

func view(kind, label string, node ast.Node, children model.ASTViews) model.ASTView {
	if children == nil {
		children = model.ASTViews{}
	}
	return model.ASTView{Kind: kind, Label: label, Span: node.SourceSpan(), Children: children}
}

func statementViews(statements ast.Statements) model.ASTViews {
	result := make(model.ASTViews, len(statements))
	for i, statement := range statements {
		result[i] = nodeView(statement)
	}
	return result
}

func expressionViews(expressions ast.Expressions) model.ASTViews {
	result := make(model.ASTViews, len(expressions))
	for i, expression := range expressions {
		result[i] = nodeView(expression)
	}
	return result
}

func add(children model.ASTViews, nodes ...ast.Node) model.ASTViews {
	for _, node := range nodes {
		if node != nil {
			children = append(children, nodeView(node))
		}
	}
	return children
}

func nodeView(node ast.Node) model.ASTView {
	switch n := node.(type) {
	case ast.Program:
		return view("program", "Program", n, statementViews(n.Statements))
	case ast.TypeRef:
		return view("type", n.Name+strings.Repeat("[]", n.Dimensions), n, nil)
	case ast.Parameter:
		children := model.ASTViews{}
		if n.Type != nil {
			children = add(children, *n.Type)
		}
		return view("parameter", n.Name, n, children)
	case ast.SwitchCase:
		children := model.ASTViews{}
		label := "case"
		if n.Default {
			label = "default"
		}
		if n.Value != nil {
			children = add(children, n.Value)
		}
		children = append(children, statementViews(n.Statements)...)
		return view("case", label, n, children)
	case ast.BlockStmt:
		return view("block", "Block", n, statementViews(n.Statements))
	case ast.VarDeclStmt:
		children := model.ASTViews{}
		if n.Type != nil {
			children = add(children, *n.Type)
		}
		children = add(children, n.Initializer)
		return view("variable", n.Name, n, children)
	case ast.ConstDeclStmt:
		children := model.ASTViews{}
		if n.Type != nil {
			children = add(children, *n.Type)
		}
		children = add(children, n.Value)
		return view("constant", n.Name, n, children)
	case ast.AssignStmt:
		return view("assignment", "=", n, add(nil, n.Target, n.Value))
	case ast.FunctionDeclStmt:
		children := make(model.ASTViews, 0, len(n.Parameters)+2)
		for _, parameter := range n.Parameters {
			children = add(children, parameter)
		}
		if n.Result != nil {
			children = add(children, *n.Result)
		}
		if n.Body != nil {
			children = add(children, *n.Body)
		}
		return view("function", n.Name, n, children)
	case ast.ClassDeclStmt:
		label := n.Name
		if n.Parent != "" {
			label += " : " + n.Parent
		}
		return view("class", label, n, statementViews(n.Members))
	case ast.ExprStmt:
		return view("expression-statement", "Expression", n, add(nil, n.Expression))
	case ast.PrintStmt:
		return view("print", "Print", n, add(nil, n.Value))
	case ast.IfStmt:
		children := add(nil, n.Condition)
		if n.Then != nil {
			children = add(children, *n.Then)
		}
		if n.Else != nil {
			children = add(children, *n.Else)
		}
		return view("if", "If", n, children)
	case ast.WhileStmt:
		children := add(nil, n.Condition)
		if n.Body != nil {
			children = add(children, *n.Body)
		}
		return view("while", "While", n, children)
	case ast.DoWhileStmt:
		children := model.ASTViews{}
		if n.Body != nil {
			children = add(children, *n.Body)
		}
		return view("do-while", "DoWhile", n, add(children, n.Condition))
	case ast.ForStmt:
		children := model.ASTViews{}
		if n.Init != nil {
			children = add(children, n.Init)
		}
		children = add(children, n.Condition, n.Post)
		if n.Body != nil {
			children = add(children, *n.Body)
		}
		return view("for", "For", n, children)
	case ast.ForeachStmt:
		children := add(nil, n.Iterable)
		if n.Body != nil {
			children = add(children, *n.Body)
		}
		return view("foreach", n.Name, n, children)
	case ast.TryCatchStmt:
		children := model.ASTViews{}
		if n.Try != nil {
			children = add(children, *n.Try)
		}
		if n.Catch != nil {
			children = add(children, *n.Catch)
		}
		return view("try-catch", n.Name, n, children)
	case ast.SwitchStmt:
		children := add(nil, n.Value)
		for _, switchCase := range n.Cases {
			children = add(children, switchCase)
		}
		return view("switch", "Switch", n, children)
	case ast.BreakStmt:
		return view("break", "Break", n, nil)
	case ast.ContinueStmt:
		return view("continue", "Continue", n, nil)
	case ast.ReturnStmt:
		return view("return", "Return", n, add(nil, n.Value))
	case ast.BadStmt:
		return view("bad-statement", "Bad", n, nil)
	case ast.IdentifierExpr:
		return view("identifier", n.Name, n, nil)
	case ast.LiteralExpr:
		return view("literal", n.Lexeme, n, nil)
	case ast.ArrayExpr:
		return view("array", "Array", n, expressionViews(n.Elements))
	case ast.ThisExpr:
		return view("this", "this", n, nil)
	case ast.NewExpr:
		return view("new", n.ClassName, n, expressionViews(n.Arguments))
	case ast.GroupExpr:
		return view("group", "Group", n, add(nil, n.Expression))
	case ast.UnaryExpr:
		return view("unary", n.Operator, n, add(nil, n.Operand))
	case ast.BinaryExpr:
		return view("binary", n.Operator, n, add(nil, n.Left, n.Right))
	case ast.TernaryExpr:
		return view("ternary", "?:", n, add(nil, n.Condition, n.Then, n.Else))
	case ast.AssignExpr:
		return view("assignment-expression", "=", n, add(nil, n.Target, n.Value))
	case ast.PropertyAssignExpr:
		return view("property-assignment", n.Name, n, add(nil, n.Receiver, n.Value))
	case ast.CallExpr:
		return view("call", "Call", n, append(add(nil, n.Callee), expressionViews(n.Arguments)...))
	case ast.IndexExpr:
		return view("index", "Index", n, add(nil, n.Collection, n.Index))
	case ast.PropertyAccessExpr:
		return view("property", n.Name, n, add(nil, n.Receiver))
	case ast.BadExpr:
		return view("bad-expression", "Bad", n, nil)
	default:
		return model.ASTView{Kind: "unknown", Label: "Unknown", Children: model.ASTViews{}}
	}
}

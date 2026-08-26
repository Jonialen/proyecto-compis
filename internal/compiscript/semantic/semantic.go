package semantic

import (
	"sort"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type scope struct {
	id, parent int
	kind       model.ScopeKind
	span       ast.Span
	symbols    model.Symbols
	outer      *scope
}

type analyzer struct {
	scopes      []*scope
	diagnostics model.Diagnostics
}

// Analyze resolves lexical names and checks expression and assignment types.
func Analyze(program ast.Program) (model.ScopeSnapshots, model.Diagnostics) {
	a := &analyzer{diagnostics: model.Diagnostics{}}
	global := a.newScope(nil, model.ScopeGlobal, program.Span)
	a.statements(global, program.Statements)
	snapshots := make(model.ScopeSnapshots, len(a.scopes))
	for i, current := range a.scopes {
		sort.SliceStable(current.symbols, func(i, j int) bool {
			return current.symbols[i].Span.Start.Offset < current.symbols[j].Span.Start.Offset
		})
		snapshots[i] = model.ScopeSnapshot{ID: current.id, ParentID: current.parent, Kind: current.kind, Span: current.span, Symbols: current.symbols}
	}
	sort.SliceStable(a.diagnostics, func(i, j int) bool {
		x, y := a.diagnostics[i], a.diagnostics[j]
		if x.Span.Start.Offset != y.Span.Start.Offset {
			return x.Span.Start.Offset < y.Span.Start.Offset
		}
		if x.Span.End.Offset != y.Span.End.Offset {
			return x.Span.End.Offset < y.Span.End.Offset
		}
		return x.Code < y.Code
	})
	return snapshots, a.diagnostics
}

func (a *analyzer) newScope(outer *scope, kind model.ScopeKind, span ast.Span) *scope {
	s := &scope{id: len(a.scopes) + 1, kind: kind, span: span, symbols: model.Symbols{}, outer: outer}
	if outer != nil {
		s.parent = outer.id
	}
	a.scopes = append(a.scopes, s)
	return s
}

func (a *analyzer) statements(s *scope, statements ast.Statements) {
	for _, statement := range statements {
		if function, ok := statement.(ast.FunctionDeclStmt); ok {
			a.declare(s, model.Symbol{Name: function.Name, Kind: model.SymbolFunction, Type: functionType(function), Span: function.Span})
		}
	}
	for _, statement := range statements {
		a.statement(s, statement)
	}
}

func (a *analyzer) statement(s *scope, statement ast.Statement) {
	switch n := statement.(type) {
	case ast.VarDeclStmt:
		inferred := a.expression(s, n.Initializer)
		typeOf := declaredType(n.Type, inferred)
		a.declare(s, model.Symbol{Name: n.Name, Kind: model.SymbolVariable, Type: typeOf, Mutable: true, Span: n.Span})
		if n.Type != nil && !compatible(typeOf, inferred) {
			a.problem("SEM_TYPE", "initializer is not assignable to "+typeOf.Name, n.Span)
		}
	case ast.ConstDeclStmt:
		value := a.expression(s, n.Value)
		typeOf := declaredType(n.Type, value)
		a.declare(s, model.Symbol{Name: n.Name, Kind: model.SymbolConstant, Type: typeOf, Span: n.Span})
		if n.Type != nil && !compatible(typeOf, value) {
			a.problem("SEM_TYPE", "constant value is not assignable to "+typeOf.Name, n.Span)
		}
	case ast.AssignStmt:
		a.assignment(s, n.Target, n.Value, n.Span)
	case ast.FunctionDeclStmt:
		fn := a.newScope(s, model.ScopeFunction, n.Span)
		for _, parameter := range n.Parameters {
			a.declare(fn, model.Symbol{Name: parameter.Name, Kind: model.SymbolParameter, Type: declaredType(parameter.Type, errorType()), Mutable: true, Span: parameter.Span})
		}
		if n.Body != nil {
			a.statements(fn, n.Body.Statements)
		}
	case ast.BlockStmt:
		a.statements(a.newScope(s, model.ScopeBlock, n.Span), n.Statements)
	case ast.ExprStmt:
		a.expression(s, n.Expression)
	case ast.PrintStmt:
		a.expression(s, n.Value)
	case ast.ReturnStmt:
		a.expression(s, n.Value)
	case ast.IfStmt:
		a.expression(s, n.Condition)
		a.block(s, n.Then)
		a.block(s, n.Else)
	}
}

func (a *analyzer) block(s *scope, block *ast.BlockStmt) {
	if block != nil {
		a.statement(s, *block)
	}
}

func (a *analyzer) declare(s *scope, symbol model.Symbol) {
	for _, existing := range s.symbols {
		if existing.Name == symbol.Name {
			a.problem("SEM_DUPLICATE", "duplicate declaration of "+symbol.Name, symbol.Span)
			return
		}
	}
	s.symbols = append(s.symbols, symbol)
}

func lookup(s *scope, name string) (model.Symbol, bool) {
	for current := s; current != nil; current = current.outer {
		for _, symbol := range current.symbols {
			if symbol.Name == name {
				return symbol, true
			}
		}
	}
	return model.Symbol{}, false
}

func (a *analyzer) assignment(s *scope, target, value ast.Expression, span ast.Span) model.Type {
	right := a.expression(s, value)
	identifier, ok := target.(ast.IdentifierExpr)
	if !ok {
		a.expression(s, target)
		return errorType()
	}
	symbol, found := lookup(s, identifier.Name)
	if !found {
		a.problem("SEM_UNRESOLVED", "unresolved name "+identifier.Name, identifier.Span)
		return errorType()
	}
	if !symbol.Mutable {
		a.problem("SEM_CONSTANT_ASSIGNMENT", "cannot assign to constant "+identifier.Name, span)
		return errorType()
	}
	if !compatible(symbol.Type, right) {
		a.problem("SEM_TYPE", "value is not assignable to "+symbol.Type.Name, span)
		return errorType()
	}
	return symbol.Type
}

func (a *analyzer) expression(s *scope, expression ast.Expression) model.Type {
	if expression == nil {
		return errorType()
	}
	switch n := expression.(type) {
	case ast.LiteralExpr:
		return literalType(n.Lexeme)
	case ast.IdentifierExpr:
		if symbol, ok := lookup(s, n.Name); ok {
			return symbol.Type
		}
		a.problem("SEM_UNRESOLVED", "unresolved name "+n.Name, n.Span)
		return errorType()
	case ast.GroupExpr:
		return a.expression(s, n.Expression)
	case ast.UnaryExpr:
		operand := a.expression(s, n.Operand)
		if operand.Kind == model.TypeError {
			return operand
		}
		if (n.Operator == "!" && operand.Kind == model.TypeBoolean) || (n.Operator == "-" && numeric(operand)) {
			return operand
		}
		a.problem("SEM_OPERATOR", "invalid operand for "+n.Operator, n.Span)
		return errorType()
	case ast.BinaryExpr:
		return a.binary(s, n)
	case ast.AssignExpr:
		return a.assignment(s, n.Target, n.Value, n.Span)
	case ast.BadExpr:
		return errorType()
	default:
		return errorType()
	}
}

func (a *analyzer) binary(s *scope, n ast.BinaryExpr) model.Type {
	left, right := a.expression(s, n.Left), a.expression(s, n.Right)
	if left.Kind == model.TypeError || right.Kind == model.TypeError {
		return errorType()
	}
	switch n.Operator {
	case "+":
		if left.Kind == model.TypeString && right.Kind == model.TypeString {
			return primitive(model.TypeString)
		}
		fallthrough
	case "-", "*", "/", "%":
		if numeric(left) && numeric(right) {
			if n.Operator == "/" || left.Kind == model.TypeFloat || right.Kind == model.TypeFloat {
				return primitive(model.TypeFloat)
			}
			return primitive(model.TypeInteger)
		}
	case "&&", "||":
		if left.Kind == model.TypeBoolean && right.Kind == model.TypeBoolean {
			return primitive(model.TypeBoolean)
		}
	case "<", "<=", ">", ">=":
		if numeric(left) && numeric(right) {
			return primitive(model.TypeBoolean)
		}
	case "==", "!=":
		if compatible(left, right) || compatible(right, left) {
			return primitive(model.TypeBoolean)
		}
	}
	a.problem("SEM_OPERATOR", "invalid operands for "+n.Operator, n.Span)
	return errorType()
}

func (a *analyzer) problem(code, message string, span ast.Span) {
	a.diagnostics = append(a.diagnostics, model.Diagnostic{Code: code, Phase: model.PhaseSemantic, Message: message, Span: span})
}

func primitive(kind model.TypeKind) model.Type {
	return model.Type{Kind: kind, Name: string(kind), Params: model.Types{}}
}
func errorType() model.Type { return primitive(model.TypeError) }
func numeric(value model.Type) bool {
	return value.Kind == model.TypeInteger || value.Kind == model.TypeFloat
}

func literalType(lexeme string) model.Type {
	switch {
	case lexeme == "null":
		return primitive(model.TypeNull)
	case lexeme == "true" || lexeme == "false":
		return primitive(model.TypeBoolean)
	case strings.HasPrefix(lexeme, `"`):
		return primitive(model.TypeString)
	case strings.Contains(lexeme, "."):
		return primitive(model.TypeFloat)
	default:
		return primitive(model.TypeInteger)
	}
}

func declaredType(ref *ast.TypeRef, fallback model.Type) model.Type {
	if ref == nil {
		return fallback
	}
	kinds := map[string]model.TypeKind{"integer": model.TypeInteger, "float": model.TypeFloat, "boolean": model.TypeBoolean, "string": model.TypeString}
	result := primitive(kinds[ref.Name])
	if result.Kind == "" {
		result = model.Type{Kind: model.TypeClass, Name: ref.Name, Params: model.Types{}}
	}
	for range ref.Dimensions {
		element := result
		result = model.Type{Kind: model.TypeList, Name: "list", Element: &element, Params: model.Types{}}
	}
	return result
}

func functionType(function ast.FunctionDeclStmt) model.Type {
	params := make(model.Types, len(function.Parameters))
	for i, parameter := range function.Parameters {
		params[i] = declaredType(parameter.Type, errorType())
	}
	result := declaredType(function.Result, primitive(model.TypeNull))
	return model.Type{Kind: model.TypeFunction, Name: function.Name, Params: params, Result: &result}
}

func compatible(target, value model.Type) bool {
	if target.Kind == model.TypeError || value.Kind == model.TypeError || target.Kind == value.Kind {
		return true
	}
	if target.Kind == model.TypeFloat && value.Kind == model.TypeInteger {
		return true
	}
	return value.Kind == model.TypeNull && (target.Kind == model.TypeString || target.Kind == model.TypeList || target.Kind == model.TypeClass || target.Kind == model.TypeFunction)
}

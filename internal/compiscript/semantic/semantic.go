package semantic

import (
	"sort"
	"strconv"
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
	function   *scope
}

type analyzer struct {
	scopes                                []*scope
	diagnostics                           model.Diagnostics
	functionDepth, loopDepth, switchDepth int
	returnType                            *model.Type
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
		s.function = outer.function
	}
	if kind == model.ScopeFunction {
		s.function = s
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
	terminated := false
	for _, statement := range statements {
		if terminated {
			a.problem("SEM_UNREACHABLE", "unreachable statement", statement.SourceSpan())
		}
		a.statement(s, statement)
		terminated = terminated || a.terminates(statement)
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
		if n.Body == nil {
			break
		}
		depth, loops, switches, result := a.functionDepth, a.loopDepth, a.switchDepth, a.returnType
		functionResult := declaredType(n.Result, primitive(model.TypeNull))
		a.functionDepth, a.loopDepth, a.switchDepth, a.returnType = depth+1, 0, 0, &functionResult
		a.statements(fn, n.Body.Statements)
		if functionResult.Kind != model.TypeNull && !allReturns(n.Body.Statements) {
			a.problem("SEM_MISSING_RETURN", "function "+n.Name+" does not return on every path", n.Span)
		}
		a.functionDepth, a.loopDepth, a.switchDepth, a.returnType = depth, loops, switches, result
	case ast.BlockStmt:
		a.statements(a.newScope(s, model.ScopeBlock, n.Span), n.Statements)
	case ast.ExprStmt:
		a.expression(s, n.Expression)
	case ast.PrintStmt:
		a.expression(s, n.Value)
	case ast.ReturnStmt:
		value := primitive(model.TypeNull)
		if n.Value != nil {
			value = a.expression(s, n.Value)
		}
		if a.functionDepth == 0 {
			a.problem("SEM_TRANSFER", "return outside function", n.Span)
		} else if !compatible(*a.returnType, value) {
			a.problem("SEM_RETURN", "return value is not assignable to "+a.returnType.Name, n.Span)
		}
	case ast.IfStmt:
		a.condition(s, n.Condition)
		a.block(s, n.Then)
		a.block(s, n.Else)
	case ast.WhileStmt:
		a.condition(s, n.Condition)
		a.loopDepth++
		a.block(s, n.Body)
		a.loopDepth--
	case ast.DoWhileStmt:
		a.loopDepth++
		a.block(s, n.Body)
		a.loopDepth--
		a.condition(s, n.Condition)
	case ast.ForStmt:
		loop := a.newScope(s, model.ScopeBlock, n.Span)
		if n.Init != nil {
			a.statement(loop, n.Init)
		}
		if n.Condition != nil {
			a.condition(loop, n.Condition)
		}
		a.expression(loop, n.Post)
		a.loopDepth++
		a.block(loop, n.Body)
		a.loopDepth--
	case ast.ForeachStmt:
		a.expression(s, n.Iterable)
		a.block(s, n.Body)
	case ast.TryCatchStmt:
		a.block(s, n.Try)
		a.block(s, n.Catch)
	case ast.SwitchStmt:
		value := a.expression(s, n.Value)
		seen := map[string]bool{}
		a.switchDepth++
		for _, switchCase := range n.Cases {
			caseType := a.expression(s, switchCase.Value)
			if !switchCase.Default && !compatible(value, caseType) && !compatible(caseType, value) {
				a.problem("SEM_CASE_TYPE", "case is incompatible with switch expression", switchCase.Span)
			}
			if key, ok := caseKey(switchCase.Value); ok {
				if seen[key] {
					a.problem("SEM_DUPLICATE_CASE", "duplicate switch case", switchCase.Span)
				}
				seen[key] = true
			}
			a.statements(s, switchCase.Statements)
		}
		a.switchDepth--
	case ast.BreakStmt:
		if a.loopDepth == 0 && a.switchDepth == 0 {
			a.problem("SEM_TRANSFER", "break outside loop or switch", n.Span)
		}
	case ast.ContinueStmt:
		if a.loopDepth == 0 {
			a.problem("SEM_TRANSFER", "continue outside loop", n.Span)
		}
	case ast.ClassDeclStmt:
		a.statements(a.newScope(s, model.ScopeBlock, n.Span), n.Members)
	}
}

func (a *analyzer) block(s *scope, block *ast.BlockStmt) {
	if block != nil {
		a.statement(s, *block)
	}
}

func (a *analyzer) condition(s *scope, expression ast.Expression) {
	typeOf := a.expression(s, expression)
	if typeOf.Kind != model.TypeBoolean && typeOf.Kind != model.TypeError {
		a.problem("SEM_CONDITION", "condition must be boolean", expression.SourceSpan())
	}
}

func (a *analyzer) terminates(statement ast.Statement) bool {
	switch n := statement.(type) {
	case ast.ReturnStmt:
		return a.functionDepth > 0
	case ast.BreakStmt:
		return a.loopDepth > 0 || a.switchDepth > 0
	case ast.ContinueStmt:
		return a.loopDepth > 0
	case ast.BlockStmt:
		return a.sequenceTerminates(n.Statements)
	case ast.IfStmt:
		return n.Else != nil && a.sequenceTerminates(n.Then.Statements) && a.sequenceTerminates(n.Else.Statements)
	case ast.SwitchStmt:
		return switchReturns(n)
	}
	return false
}

func (a *analyzer) sequenceTerminates(statements ast.Statements) bool {
	for _, statement := range statements {
		if a.terminates(statement) {
			return true
		}
	}
	return false
}

func allReturns(statements ast.Statements) bool {
	for _, statement := range statements {
		switch n := statement.(type) {
		case ast.ReturnStmt:
			return true
		case ast.BlockStmt:
			if allReturns(n.Statements) {
				return true
			}
		case ast.IfStmt:
			if n.Else != nil && allReturns(n.Then.Statements) && allReturns(n.Else.Statements) {
				return true
			}
		case ast.SwitchStmt:
			if switchReturns(n) {
				return true
			}
		case ast.DoWhileStmt:
			if allReturns(n.Body.Statements) {
				return true
			}
		case ast.WhileStmt:
			if constantTrue(n.Condition) && allReturns(n.Body.Statements) {
				return true
			}
		case ast.ForStmt:
			if (n.Condition == nil || constantTrue(n.Condition)) && allReturns(n.Body.Statements) {
				return true
			}
		}
	}
	return false
}

func switchReturns(statement ast.SwitchStmt) bool {
	hasDefault := false
	for _, switchCase := range statement.Cases {
		hasDefault = hasDefault || switchCase.Default
		if !allReturns(switchCase.Statements) {
			return false
		}
	}
	return hasDefault
}

func caseKey(expression ast.Expression) (string, bool) {
	if unary, ok := expression.(ast.UnaryExpr); ok && unary.Operator == "-" {
		if literal, literalOK := unary.Operand.(ast.LiteralExpr); literalOK && numeric(literalType(literal.Lexeme)) {
			value, _ := strconv.ParseFloat("-"+literal.Lexeme, 64)
			if value == 0 {
				value = 0
			}
			return "number:" + strconv.FormatFloat(value, 'g', -1, 64), true
		}
	}
	literal, ok := expression.(ast.LiteralExpr)
	if !ok {
		return "", false
	}
	typeOf := literalType(literal.Lexeme)
	if numeric(typeOf) {
		value, _ := strconv.ParseFloat(literal.Lexeme, 64)
		return "number:" + strconv.FormatFloat(value, 'g', -1, 64), true
	}
	if typeOf.Kind == model.TypeString {
		if value, err := strconv.Unquote(literal.Lexeme); err == nil {
			return "string:" + value, true
		}
	}
	return string(typeOf.Kind) + ":" + literal.Lexeme, true
}

func constantTrue(expression ast.Expression) bool {
	literal, ok := expression.(ast.LiteralExpr)
	return ok && literal.Lexeme == "true"
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

func (a *analyzer) resolve(s *scope, name string) (*model.Symbol, bool) {
	for current := s; current != nil; current = current.outer {
		for i := range current.symbols {
			if current.symbols[i].Name == name {
				if s.function != nil && current.function != nil && s.function != current.function {
					current.symbols[i].Captured = true
				}
				return &current.symbols[i], true
			}
		}
	}
	return nil, false
}

func (a *analyzer) assignment(s *scope, target, value ast.Expression, span ast.Span) model.Type {
	right := a.expression(s, value)
	identifier, ok := target.(ast.IdentifierExpr)
	if !ok {
		a.expression(s, target)
		return errorType()
	}
	symbol, found := a.resolve(s, identifier.Name)
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
		if symbol, ok := a.resolve(s, n.Name); ok {
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
	case ast.ArrayExpr:
		for _, element := range n.Elements {
			a.expression(s, element)
		}
		return errorType()
	case ast.NewExpr:
		for _, argument := range n.Arguments {
			a.expression(s, argument)
		}
		return model.Type{Kind: model.TypeClass, Name: n.ClassName, Params: model.Types{}}
	case ast.TernaryExpr:
		a.expression(s, n.Condition)
		left, right := a.expression(s, n.Then), a.expression(s, n.Else)
		if compatible(left, right) {
			return left
		}
		if compatible(right, left) {
			return right
		}
		return errorType()
	case ast.PropertyAssignExpr:
		a.expression(s, n.Receiver)
		a.expression(s, n.Value)
		return errorType()
	case ast.CallExpr:
		callee := a.expression(s, n.Callee)
		arguments := make(model.Types, len(n.Arguments))
		for i, argument := range n.Arguments {
			arguments[i] = a.expression(s, argument)
		}
		if callee.Kind == model.TypeFunction && callee.Result != nil {
			if len(arguments) != len(callee.Params) {
				a.problem("SEM_ARITY", "wrong number of arguments", n.Span)
			}
			for i := 0; i < len(arguments) && i < len(callee.Params); i++ {
				if !compatible(callee.Params[i], arguments[i]) {
					a.problem("SEM_ARGUMENT", "argument is not assignable to "+callee.Params[i].Name, n.Arguments[i].SourceSpan())
				}
			}
			return *callee.Result
		}
		return errorType()
	case ast.IndexExpr:
		a.expression(s, n.Collection)
		a.expression(s, n.Index)
		return errorType()
	case ast.PropertyAccessExpr:
		a.expression(s, n.Receiver)
		return errorType()
	case ast.ThisExpr:
		return errorType()
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
		if left.Kind != model.TypeFunction && right.Kind != model.TypeFunction && (compatible(left, right) || compatible(right, left)) {
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
	return value.Kind == model.TypeNull && (target.Kind == model.TypeString || target.Kind == model.TypeList || target.Kind == model.TypeClass)
}

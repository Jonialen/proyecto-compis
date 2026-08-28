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
	shapes     map[string]*collectionShape
	constants  map[string]int
}

type collectionShape struct{ elements []*collectionShape }

type classInfo struct {
	declaration ast.ClassDeclStmt
	members     map[string]model.Symbol
}

type analyzer struct {
	scopes                                []*scope
	diagnostics                           model.Diagnostics
	functionDepth, loopDepth, switchDepth int
	returnType                            *model.Type
	classes                               map[string]*classInfo
	className                             string
}

// Analyze resolves lexical names and checks expression and assignment types.
func Analyze(program ast.Program) (model.ScopeSnapshots, model.Diagnostics) {
	a := &analyzer{diagnostics: model.Diagnostics{}, classes: map[string]*classInfo{}}
	global := a.newScope(nil, model.ScopeGlobal, program.Span)
	a.registerClasses(global, program.Statements)
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
	s := &scope{id: len(a.scopes) + 1, kind: kind, span: span, symbols: model.Symbols{}, shapes: map[string]*collectionShape{}, constants: map[string]int{}, outer: outer}
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
		expected := declaredType(n.Type, errorType())
		inferred := a.expressionExpected(s, n.Initializer, typePointer(n.Type, expected))
		typeOf := declaredType(n.Type, inferred)
		a.declare(s, model.Symbol{Name: n.Name, Kind: model.SymbolVariable, Type: typeOf, Mutable: true, Span: n.Span})
		s.shapes[n.Name] = a.staticShape(s, n.Initializer)
		if n.Type != nil && !a.compatible(typeOf, inferred) {
			a.problem("SEM_TYPE", "initializer is not assignable to "+typeOf.Name, n.Span)
		}
	case ast.ConstDeclStmt:
		expected := declaredType(n.Type, errorType())
		value := a.expressionExpected(s, n.Value, typePointer(n.Type, expected))
		typeOf := declaredType(n.Type, value)
		a.declare(s, model.Symbol{Name: n.Name, Kind: model.SymbolConstant, Type: typeOf, Span: n.Span})
		s.shapes[n.Name] = a.staticShape(s, n.Value)
		if constant, ok := a.constantIndex(s, n.Value); ok && typeOf.Kind == model.TypeInteger {
			s.constants[n.Name] = constant
		}
		if n.Type != nil && !a.compatible(typeOf, value) {
			a.problem("SEM_TYPE", "constant value is not assignable to "+typeOf.Name, n.Span)
		}
	case ast.AssignStmt:
		a.assignment(s, n.Target, n.Value, n.Span)
	case ast.FunctionDeclStmt:
		a.function(s, n)
	case ast.BlockStmt:
		a.statements(a.newScope(s, model.ScopeBlock, n.Span), n.Statements)
	case ast.ExprStmt:
		a.expression(s, n.Expression)
	case ast.PrintStmt:
		a.expression(s, n.Value)
	case ast.ReturnStmt:
		value := primitive(model.TypeNull)
		if n.Value != nil {
			value = a.expressionExpected(s, n.Value, a.returnType)
		}
		if a.functionDepth == 0 {
			a.problem("SEM_TRANSFER", "return outside function", n.Span)
		} else if !a.compatible(*a.returnType, value) {
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
		if n.Catch != nil {
			caught := a.newScope(s, model.ScopeCatch, n.Catch.Span)
			a.declare(caught, model.Symbol{Name: n.Name, Kind: model.SymbolCatch, Type: primitive(model.TypeException), Span: n.Catch.Span})
			a.statements(caught, n.Catch.Statements)
		}
	case ast.SwitchStmt:
		value := a.expression(s, n.Value)
		seen := map[string]bool{}
		a.switchDepth++
		for _, switchCase := range n.Cases {
			caseType := a.expression(s, switchCase.Value)
			if !switchCase.Default && !a.compatible(value, caseType) && !a.compatible(caseType, value) {
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
		a.class(s, n)
	}
}

func (a *analyzer) function(s *scope, n ast.FunctionDeclStmt) {
	fn := a.newScope(s, model.ScopeFunction, n.Span)
	for _, parameter := range n.Parameters {
		a.declare(fn, model.Symbol{Name: parameter.Name, Kind: model.SymbolParameter, Type: declaredType(parameter.Type, errorType()), Mutable: true, Span: parameter.Span})
	}
	if n.Body == nil {
		return
	}
	depth, loops, switches, result := a.functionDepth, a.loopDepth, a.switchDepth, a.returnType
	functionResult := declaredType(n.Result, primitive(model.TypeNull))
	a.functionDepth, a.loopDepth, a.switchDepth, a.returnType = depth+1, 0, 0, &functionResult
	a.statements(fn, n.Body.Statements)
	if functionResult.Kind != model.TypeNull && !allReturns(n.Body.Statements) {
		a.problem("SEM_MISSING_RETURN", "function "+n.Name+" does not return on every path", n.Span)
	}
	a.functionDepth, a.loopDepth, a.switchDepth, a.returnType = depth, loops, switches, result
}

func (a *analyzer) registerClasses(global *scope, statements ast.Statements) {
	for _, statement := range statements {
		if declaration, ok := statement.(ast.ClassDeclStmt); ok {
			a.declare(global, model.Symbol{Name: declaration.Name, Kind: model.SymbolClass, Type: model.Type{Kind: model.TypeClass, Name: declaration.Name, Params: model.Types{}}, Span: declaration.Span})
			if _, exists := a.classes[declaration.Name]; !exists {
				a.classes[declaration.Name] = &classInfo{declaration: declaration, members: map[string]model.Symbol{}}
			}
		}
	}
	cycles := map[string]bool{}
	for _, statement := range statements {
		declaration, ok := statement.(ast.ClassDeclStmt)
		if !ok || a.classes[declaration.Name].declaration.Span != declaration.Span {
			continue
		}
		info := a.classes[declaration.Name]
		if declaration.Parent != "" {
			if _, exists := a.classes[declaration.Parent]; !exists {
				a.problem("SEM_UNKNOWN_BASE", "unknown base class "+declaration.Parent, declaration.Span)
			}
		}
		for _, member := range declaration.Members {
			var symbol model.Symbol
			switch n := member.(type) {
			case ast.VarDeclStmt:
				symbol = model.Symbol{Name: n.Name, Kind: model.SymbolField, Type: declaredType(n.Type, errorType()), Mutable: true, Span: n.Span}
			case ast.ConstDeclStmt:
				symbol = model.Symbol{Name: n.Name, Kind: model.SymbolField, Type: declaredType(n.Type, errorType()), Span: n.Span}
			case ast.FunctionDeclStmt:
				symbol = model.Symbol{Name: n.Name, Kind: model.SymbolMethod, Type: functionType(n), Span: n.Span}
			}
			if _, duplicate := info.members[symbol.Name]; duplicate {
				a.problem("SEM_DUPLICATE", "duplicate declaration of "+symbol.Name, symbol.Span)
			} else {
				info.members[symbol.Name] = symbol
			}
		}
		path, positions := []string{}, map[string]int{}
		for name := declaration.Name; name != "" && a.classes[name] != nil; name = a.classes[name].declaration.Parent {
			if at, seen := positions[name]; seen {
				for _, cyclic := range path[at:] {
					cycles[cyclic] = true
				}
				break
			}
			positions[name], path = len(path), append(path, name)
		}
	}
	for _, statement := range statements {
		declaration, ok := statement.(ast.ClassDeclStmt)
		if !ok {
			continue
		}
		if cycles[declaration.Name] {
			a.problem("SEM_INHERITANCE_CYCLE", "inheritance cycle involving "+declaration.Name, declaration.Span)
			continue
		}
		for _, member := range declaration.Members {
			name := memberName(member)
			if inherited, _ := a.lookupMember(declaration.Parent, name); name != "constructor" && inherited != nil {
				a.problem("SEM_INHERITED_MEMBER", "member "+name+" redeclares an inherited name", member.SourceSpan())
			}
		}
	}
}

func memberName(statement ast.Statement) string {
	switch n := statement.(type) {
	case ast.VarDeclStmt:
		return n.Name
	case ast.ConstDeclStmt:
		return n.Name
	case ast.FunctionDeclStmt:
		return n.Name
	}
	return ""
}

func (a *analyzer) class(s *scope, declaration ast.ClassDeclStmt) {
	info := a.classes[declaration.Name]
	if info == nil || info.declaration.Span != declaration.Span {
		return
	}
	classScope := a.newScope(s, model.ScopeClass, declaration.Span)
	for _, member := range declaration.Members {
		if symbol, ok := info.members[memberName(member)]; ok && symbol.Span == member.SourceSpan() {
			classScope.symbols = append(classScope.symbols, symbol)
		}
	}
	previous := a.className
	a.className = declaration.Name
	for _, member := range declaration.Members {
		switch n := member.(type) {
		case ast.VarDeclStmt:
			if n.Initializer != nil {
				value := a.expressionExpected(classScope, n.Initializer, typePointer(n.Type, declaredType(n.Type, errorType())))
				if n.Type != nil && !a.compatible(declaredType(n.Type, errorType()), value) {
					a.problem("SEM_TYPE", "initializer is not assignable to "+n.Type.Name, n.Span)
				}
			}
		case ast.ConstDeclStmt:
			value := a.expressionExpected(classScope, n.Value, typePointer(n.Type, declaredType(n.Type, errorType())))
			if n.Type != nil && !a.compatible(declaredType(n.Type, errorType()), value) {
				a.problem("SEM_TYPE", "constant value is not assignable to "+n.Type.Name, n.Span)
			}
		case ast.FunctionDeclStmt:
			a.function(classScope, n)
		}
	}
	a.className = previous
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
	if property, ok := target.(ast.PropertyAccessExpr); ok {
		return a.assignMember(s, property.Receiver, property.Name, value, span)
	}
	identifier, direct := target.(ast.IdentifierExpr)
	root, found := rootIdentifier(target)
	if !found {
		a.expression(s, target)
		a.expression(s, value)
		return errorType()
	}
	symbol, found := a.resolve(s, root.Name)
	if !found {
		a.problem("SEM_UNRESOLVED", "unresolved name "+root.Name, root.Span)
		a.expression(s, value)
		return errorType()
	}
	if !symbol.Mutable {
		a.problem("SEM_CONSTANT_ASSIGNMENT", "cannot assign to constant "+root.Name, span)
		a.expression(s, value)
		return errorType()
	}
	targetType := symbol.Type
	if !direct {
		targetType = a.expression(s, target)
	}
	right := a.expressionExpected(s, value, &targetType)
	if targetType.Kind == model.TypeError {
		return errorType()
	}
	if right.Kind == model.TypeError {
		return errorType()
	}
	if !a.compatible(targetType, right) {
		a.problem("SEM_TYPE", "value is not assignable to "+targetType.Name, span)
		return errorType()
	}
	owner := a.scopeContaining(s, root.Name)
	if direct {
		owner.shapes[identifier.Name] = a.staticShape(s, value)
	} else {
		for _, current := range a.scopes {
			current.shapes = map[string]*collectionShape{}
		}
	}
	return targetType
}

func (a *analyzer) expression(s *scope, expression ast.Expression) model.Type {
	return a.expressionExpected(s, expression, nil)
}

func (a *analyzer) expressionExpected(s *scope, expression ast.Expression, expected *model.Type) model.Type {
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
		return a.expressionExpected(s, n.Expression, expected)
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
		return a.collection(s, n, expected)
	case ast.NewExpr:
		info := a.classes[n.ClassName]
		if info == nil {
			for _, argument := range n.Arguments {
				a.expression(s, argument)
			}
			a.problem("SEM_UNKNOWN_CLASS", "unknown class "+n.ClassName, n.Span)
			return errorType()
		}
		constructor, hasConstructor := info.members["constructor"]
		parameters := model.Types{}
		if hasConstructor && constructor.Kind == model.SymbolMethod {
			parameters = constructor.Type.Params
		}
		a.arguments(s, n.Arguments, parameters, n.Span)
		return model.Type{Kind: model.TypeClass, Name: n.ClassName, Params: model.Types{}}
	case ast.TernaryExpr:
		a.expression(s, n.Condition)
		left, right := a.expressionExpected(s, n.Then, expected), a.expressionExpected(s, n.Else, expected)
		if a.compatible(left, right) {
			return left
		}
		if a.compatible(right, left) {
			return right
		}
		return errorType()
	case ast.PropertyAssignExpr:
		return a.assignMember(s, n.Receiver, n.Name, n.Value, n.Span)
	case ast.CallExpr:
		callee := a.expression(s, n.Callee)
		arguments := make(model.Types, len(n.Arguments))
		for i, argument := range n.Arguments {
			var parameter *model.Type
			if callee.Kind == model.TypeFunction && i < len(callee.Params) {
				parameter = &callee.Params[i]
			}
			arguments[i] = a.expressionExpected(s, argument, parameter)
		}
		if callee.Kind == model.TypeFunction && callee.Result != nil {
			if len(arguments) != len(callee.Params) {
				a.problem("SEM_ARITY", "wrong number of arguments", n.Span)
			}
			for i := 0; i < len(arguments) && i < len(callee.Params); i++ {
				if !a.compatible(callee.Params[i], arguments[i]) {
					a.problem("SEM_ARGUMENT", "argument is not assignable to "+callee.Params[i].Name, n.Arguments[i].SourceSpan())
				}
			}
			return *callee.Result
		}
		return errorType()
	case ast.IndexExpr:
		collection, index := a.expression(s, n.Collection), a.expression(s, n.Index)
		if collection.Kind == model.TypeError || index.Kind == model.TypeError {
			return errorType()
		}
		if index.Kind != model.TypeInteger {
			a.problem("SEM_INDEX", "collection index must be integer", n.Index.SourceSpan())
			return errorType()
		}
		if collection.Kind != model.TypeList || collection.Element == nil {
			a.problem("SEM_INDEX", "cannot index non-list value", n.Collection.SourceSpan())
			return errorType()
		}
		shape := a.staticShape(s, n.Collection)
		value, constant := a.constantIndex(s, n.Index)
		if shape != nil && constant && (value < 0 || value >= len(shape.elements)) {
			a.problem("SEM_BOUNDS", "collection index is provably out of bounds", n.Index.SourceSpan())
			return errorType()
		}
		return *collection.Element
	case ast.PropertyAccessExpr:
		if member := a.member(s, n.Receiver, n.Name, n.Span); member != nil {
			return member.Type
		}
		return errorType()
	case ast.ThisExpr:
		if a.className == "" {
			a.problem("SEM_THIS", "this is only valid in a class member body", n.Span)
			return errorType()
		}
		return model.Type{Kind: model.TypeClass, Name: a.className, Params: model.Types{}}
	case ast.BadExpr:
		return errorType()
	default:
		return errorType()
	}
}

func (a *analyzer) arguments(s *scope, expressions ast.Expressions, parameters model.Types, span ast.Span) {
	if len(expressions) != len(parameters) {
		a.problem("SEM_ARITY", "wrong number of arguments", span)
	}
	for i, expression := range expressions {
		var expected *model.Type
		if i < len(parameters) {
			expected = &parameters[i]
		}
		value := a.expressionExpected(s, expression, expected)
		if expected != nil && !a.compatible(*expected, value) {
			a.problem("SEM_ARGUMENT", "argument is not assignable to "+expected.Name, expression.SourceSpan())
		}
	}
}

func (a *analyzer) member(s *scope, receiver ast.Expression, name string, span ast.Span) *model.Symbol {
	typeOf := a.expression(s, receiver)
	if typeOf.Kind == model.TypeError {
		return nil
	}
	if typeOf.Kind != model.TypeClass {
		a.problem("SEM_MEMBER", "member access requires a class value", span)
		return nil
	}
	member, _ := a.lookupMember(typeOf.Name, name)
	if member == nil {
		a.problem("SEM_MEMBER", "unknown member "+name+" on "+typeOf.Name, span)
	}
	return member
}

func (a *analyzer) assignMember(s *scope, receiver ast.Expression, name string, value ast.Expression, span ast.Span) model.Type {
	member := a.member(s, receiver, name, span)
	if member == nil {
		a.expression(s, value)
		return errorType()
	}
	right := a.expressionExpected(s, value, &member.Type)
	if !member.Mutable {
		a.problem("SEM_CONSTANT_ASSIGNMENT", "cannot assign to member "+name, span)
		return errorType()
	}
	if !a.compatible(member.Type, right) {
		a.problem("SEM_TYPE", "value is not assignable to "+member.Type.Name, span)
		return errorType()
	}
	return member.Type
}

func (a *analyzer) lookupMember(className, name string) (*model.Symbol, bool) {
	seen := map[string]bool{}
	for className != "" && !seen[className] {
		seen[className] = true
		info := a.classes[className]
		if info == nil {
			break
		}
		if member, ok := info.members[name]; ok {
			copy := member
			return &copy, className == info.declaration.Name
		}
		if name == "constructor" {
			break
		}
		className = info.declaration.Parent
	}
	return nil, false
}

func (a *analyzer) collection(s *scope, array ast.ArrayExpr, expected *model.Type) model.Type {
	var context, element *model.Type
	if expected != nil && expected.Kind == model.TypeList {
		context = expected.Element
	}
	if len(array.Elements) == 0 {
		if expected != nil && expected.Kind == model.TypeError {
			return errorType()
		}
		if context == nil {
			a.problem("SEM_EMPTY_COLLECTION", "empty collection requires an element type", array.Span)
			return errorType()
		}
		return model.Type{Kind: model.TypeList, Name: "list", Element: context, Params: model.Types{}}
	}
	pending := ast.Expressions{}
	childError, mismatch := false, false
	for _, expression := range array.Elements {
		if nested, ok := expression.(ast.ArrayExpr); ok && len(nested.Elements) == 0 {
			pending = append(pending, expression)
			continue
		}
		value := a.expressionExpected(s, expression, context)
		if value.Kind == model.TypeError {
			childError = true
			continue
		}
		if element == nil {
			element = &value
		} else if unified, ok := a.unifiedType(*element, value); ok {
			element = &unified
		} else {
			mismatch = true
		}
	}
	if element == nil {
		if context == nil {
			a.problem("SEM_EMPTY_COLLECTION", "empty collection requires an element type", array.Span)
			return errorType()
		}
		element = context
	}
	for _, expression := range pending {
		if element.Kind != model.TypeList {
			mismatch = true
			continue
		}
		if a.expressionExpected(s, expression, element).Kind == model.TypeError {
			childError = true
		}
	}
	if mismatch {
		a.problem("SEM_COLLECTION", "collection elements must have one homogeneous type", array.Span)
		return errorType()
	}
	if childError {
		return errorType()
	}
	return model.Type{Kind: model.TypeList, Name: "list", Element: element, Params: model.Types{}}
}

func (a *analyzer) unifiedType(left, right model.Type) (model.Type, bool) {
	if a.compatible(left, right) {
		return left, true
	}
	if a.compatible(right, left) {
		return right, true
	}
	return model.Type{}, false
}

func (a *analyzer) staticShape(s *scope, expression ast.Expression) *collectionShape {
	switch n := expression.(type) {
	case ast.ArrayExpr:
		shape := &collectionShape{elements: make([]*collectionShape, len(n.Elements))}
		for i, element := range n.Elements {
			shape.elements[i] = a.staticShape(s, element)
		}
		return shape
	case ast.IdentifierExpr:
		if owner := a.scopeContaining(s, n.Name); owner != nil {
			return owner.shapes[n.Name]
		}
	case ast.IndexExpr:
		shape := a.staticShape(s, n.Collection)
		index, ok := a.constantIndex(s, n.Index)
		if shape != nil && ok && index >= 0 && index < len(shape.elements) {
			return shape.elements[index]
		}
	}
	return nil
}

func (a *analyzer) constantIndex(s *scope, expression ast.Expression) (int, bool) {
	switch n := expression.(type) {
	case ast.LiteralExpr:
		value, err := strconv.Atoi(n.Lexeme)
		return value, err == nil
	case ast.IdentifierExpr:
		if owner := a.scopeContaining(s, n.Name); owner != nil {
			value, ok := owner.constants[n.Name]
			return value, ok
		}
	case ast.GroupExpr:
		return a.constantIndex(s, n.Expression)
	case ast.UnaryExpr:
		value, ok := a.constantIndex(s, n.Operand)
		return -value, ok && n.Operator == "-"
	case ast.BinaryExpr:
		left, leftOK := a.constantIndex(s, n.Left)
		right, rightOK := a.constantIndex(s, n.Right)
		if !leftOK || !rightOK {
			return 0, false
		}
		switch n.Operator {
		case "+":
			return left + right, true
		case "-":
			return left - right, true
		case "*":
			return left * right, true
		case "%":
			if right != 0 {
				return left % right, true
			}
		}
	}
	return 0, false
}

func rootIdentifier(expression ast.Expression) (ast.IdentifierExpr, bool) {
	switch n := expression.(type) {
	case ast.IdentifierExpr:
		return n, true
	case ast.IndexExpr:
		return rootIdentifier(n.Collection)
	default:
		return ast.IdentifierExpr{}, false
	}
}

func (a *analyzer) scopeContaining(s *scope, name string) *scope {
	for current := s; current != nil; current = current.outer {
		for i := range current.symbols {
			if current.symbols[i].Name == name {
				return current
			}
		}
	}
	return nil
}

func typePointer(ref *ast.TypeRef, value model.Type) *model.Type {
	if ref == nil {
		return nil
	}
	return &value
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
		if left.Kind != model.TypeFunction && right.Kind != model.TypeFunction && (a.compatible(left, right) || a.compatible(right, left)) {
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

func (a *analyzer) compatible(target, value model.Type) bool {
	if target.Kind == model.TypeError || value.Kind == model.TypeError {
		return true
	}
	if target.Kind == model.TypeList && value.Kind == model.TypeList {
		return target.Element != nil && value.Element != nil && a.compatible(*target.Element, *value.Element)
	}
	if target.Kind == model.TypeClass && value.Kind == model.TypeClass {
		seen := map[string]bool{}
		for name := value.Name; name != "" && !seen[name]; {
			seen[name] = true
			if name == target.Name {
				return true
			}
			info := a.classes[name]
			if info == nil {
				break
			}
			name = info.declaration.Parent
		}
		return false
	}
	if target.Kind == value.Kind {
		return true
	}
	if target.Kind == model.TypeFloat && value.Kind == model.TypeInteger {
		return true
	}
	return value.Kind == model.TypeNull && (target.Kind == model.TypeString || target.Kind == model.TypeList || target.Kind == model.TypeClass)
}

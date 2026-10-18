package tac

import (
	"fmt"
	"sort"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type functionInfo struct {
	label         string
	typeOf        model.Type
	edges         []*functionInfo
	globals       map[string]Operand
	captures      map[string]Operand
	definingScope int
	scopeID       int
	method        *classInfo
	typeClasses   []string
}

type mainCall struct {
	callee    *functionInfo
	span      ast.Span
	available map[string]bool
	caller    *functionInfo
}

type functionRegistry struct {
	ordered       []*functionInfo
	calls         []mainCall
	closures      bool
	needsClosures bool
	environments  map[int]EnvironmentLayout
	classes       []*classInfo
}

func (l *lowerer) predeclare(global *environment, statements ast.Statements) bool {
	for _, statement := range statements {
		n, ok := statement.(ast.FunctionDeclStmt)
		if !ok {
			continue
		}
		var found *model.Symbol
		for _, scope := range l.scopes {
			if scope.ID != global.id {
				continue
			}
			for _, symbol := range scope.Symbols {
				if symbol.Name == n.Name && symbol.Span == n.Span && (symbol.Kind == model.SymbolFunction || symbol.Kind == model.SymbolMethod) {
					copy := symbol
					found = &copy
					break
				}
			}
		}
		if found == nil || found.Type.Kind != model.TypeFunction || found.Type.Result == nil {
			return l.unsupported(n.Span, "missing matching function signature")
		}
		if n.Body == nil {
			return l.unsupported(n.Span, "bodyless functions are unsupported")
		}
		if len(found.Type.Params) != len(n.Parameters) || (!callValue(found.Type.Result.Kind) && found.Type.Result.Kind != model.TypeNull) {
			return l.unsupported(n.Span, "unsupported function signature")
		}
		for _, parameter := range found.Type.Params {
			if !callValue(parameter.Kind) {
				return l.unsupported(n.Span, "unsupported function signature")
			}
		}
		f := &functionInfo{label: symbolOperand(global.id, *found).Value, typeOf: found.Type, globals: map[string]Operand{}, captures: map[string]Operand{}, definingScope: global.id}
		f.typeClasses = l.signatureClasses(global, found.Type)
		if c := l.classScope(global.id); c != nil {
			f.method = c
			f.definingScope = c.owner
		}
		l.registry.ordered = append(l.registry.ordered, f)
		global.active[n.Name] = binding{function: f}
	}
	return true
}

func (l *lowerer) functionBody(global *environment, n ast.FunctionDeclStmt) bool {
	bound, ok := global.lookup(n.Name)
	if !ok || bound.function == nil {
		return l.unsupported(n.Span, "missing global function identity")
	}
	f := bound.function
	child := lowerer{scopes: l.scopes, program: Program{Instructions: []Instruction{}}, registry: l.registry, current: f, globalID: l.globalID}
	// Declaration-time visibility, rather than the final global inventory.
	env := child.environment(global.fork(), model.ScopeFunction, n.Span)
	if env == nil {
		l.diagnostics = append(l.diagnostics, child.diagnostics...)
		return false
	}
	p := Procedure{Label: f.label, ScopeID: env.id, Span: n.Span, ResultKind: f.typeOf.Result.Kind, Parameters: []Operand{}}
	p.ResultClass = child.typedOperand(global, Operand{Type: p.ResultKind}, *f.typeOf.Result).Class
	p.ResultShape = child.typedOperand(global, Operand{Type: p.ResultKind}, *f.typeOf.Result).Shape
	if len(f.typeClasses) != 0 {
		p.ResultClass = f.typeClasses[len(f.typeOf.Params)]
	}
	if f.method != nil {
		receiver := Operand{Kind: Symbol, Value: fmt.Sprintf("s%d.$this", env.id), Type: model.TypeClass, Class: f.method.layout.ID}
		p.Receiver = &receiver
		env.active["$this"] = binding{operand: receiver, typeOf: model.Type{Kind: model.TypeClass, Name: f.method.layout.Name}}
	}
	f.scopeID = env.id
	if l.registry.closures {
		p.LexicalScope = f.definingScope
	}
	for i, parameter := range n.Parameters {
		found := false
		for _, scope := range l.scopes {
			if scope.ID != env.id {
				continue
			}
			for _, symbol := range scope.Symbols {
				if symbol.Name == parameter.Name && symbol.Kind == model.SymbolParameter && symbol.Type.Kind == f.typeOf.Params[i].Kind {
					operand := child.typedOperand(env, symbolOperand(env.id, symbol), symbol.Type)
					if len(f.typeClasses) != 0 {
						operand.Class = f.typeClasses[i]
					}
					env.active[symbol.Name] = binding{operand: operand, mutable: symbol.Mutable, typeOf: symbol.Type}
					p.Parameters = append(p.Parameters, operand)
					found = true
					break
				}
			}
		}
		if !found {
			return l.unsupported(parameter.Span, "missing matching semantic parameter")
		}
	}
	if !child.statements(env, n.Body.Statements) {
		l.diagnostics = append(l.diagnostics, child.diagnostics...)
		return false
	}
	if p.ResultKind == model.TypeNull {
		child.program.Instructions = append(child.program.Instructions, Instruction{Op: Return, Span: n.Body.Span})
	}
	p.Instructions, p.PeakTemporaries = child.program.Instructions, child.pool.peak
	if !child.layout(&p) {
		l.diagnostics = append(l.diagnostics, child.diagnostics...)
		return false
	}
	l.program.Procedures = append(l.program.Procedures, p)
	// Nested bodies are emitted by their child lowerer, not into their caller's
	// instruction tape. Registry order is therefore NOT procedure slice order.
	l.program.Procedures = append(l.program.Procedures, child.program.Procedures...)
	return true
}

func (l *lowerer) returnStatement(env *environment, n ast.ReturnStmt) bool {
	if l.current == nil {
		return l.unsupported(n.Span, "return requires a function")
	}
	kind := l.current.typeOf.Result.Kind
	if kind == model.TypeNull {
		if n.Value != nil && !nullLiteral(n.Value) {
			value, ok := l.expression(env, n.Value)
			if !ok {
				return false
			}
			if value.Type == model.TypeError {
				value = l.checkedValue(Operand{Type: model.TypeNull}, value, n.Span)
			} else if value.Type != model.TypeNull {
				return l.unsupported(n.Span, "null-result procedures only return null or no value")
			}
			l.pool.release(value)
		}
		l.removeHandlers(0, n.Span)
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: Return, Span: n.Span})
		return true
	}
	if n.Value == nil {
		return l.unsupported(n.Span, "scalar functions require a return value")
	}
	expected := l.typedOperand(env, Operand{Type: kind}, *l.current.typeOf.Result)
	if len(l.current.typeClasses) != 0 {
		expected.Class = l.current.typeClasses[len(l.current.typeOf.Params)]
	}
	value, ok := l.expressionExpected(env, n.Value, expected)
	if !ok {
		return false
	}
	if value.Type == model.TypeError || l.handlers != 0 {
		value = l.checkedValue(expected, value, n.Value.SourceSpan())
	}
	if kind == model.TypeFloat {
		value = l.widen(value, n.Value.SourceSpan())
	}
	if !l.assignable(expected, value) {
		return l.unsupported(n.Span, "return type mismatch")
	}
	if value.Type == model.TypeNull {
		// Null receives its declared reference type at the result-cell boundary,
		// exactly as it does at a typed store or formal parameter boundary.
		target := l.typedOperand(env, l.pool.acquire(kind), *l.current.typeOf.Result)
		target.Class = expected.Class
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: target, Left: value, Span: n.Span})
		value = target
	}
	l.removeHandlers(0, n.Span)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Return, ReturnValue: &value, Span: n.Span})
	l.pool.release(value)
	return true
}

func nullLiteral(expression ast.Expression) bool {
	if n, ok := expression.(ast.GroupExpr); ok {
		return nullLiteral(n.Expression)
	}
	n, ok := expression.(ast.LiteralExpr)
	return ok && n.Lexeme == "null"
}

func (l *lowerer) reference(operand Operand) {
	if l.current != nil && operand.Kind == Symbol && strings.HasPrefix(operand.Value, fmt.Sprintf("s%d.", l.globalID)) {
		l.current.globals[operand.Value] = operand
	} else if l.current != nil && operand.Kind == Symbol && !l.owns(l.current, operand) {
		l.current.captures[operand.Value] = operand
	}
}

func (l *lowerer) resolveCalls() bool {
	// Every set starts with its direct references. Monotone union over the finite
	// binding inventory terminates even for self/mutual cycles; ignoring a DFS
	// backedge would leave some cycle members with incomplete requirements.
	for changed := true; changed; {
		changed = false
		for _, f := range l.registry.ordered {
			for _, callee := range f.edges {
				for name, operand := range callee.globals {
					if _, exists := f.globals[name]; !exists {
						f.globals[name] = operand
						changed = true
					}
				}
				for name, operand := range callee.captures {
					if !l.owns(f, operand) {
						if _, exists := f.captures[name]; !exists {
							f.captures[name] = operand
							changed = true
						}
					}
				}
			}
		}
	}
	for _, call := range l.registry.calls {
		if call.caller == nil {
			for _, operand := range sortedGlobals(call.callee.globals) {
				if !call.available[operand.Value] {
					return l.unsupported(call.span, "call requires initialized global binding "+operand.Value)
				}
			}
		}
		for _, operand := range sortedGlobals(call.callee.captures) {
			if (call.caller == nil || l.owns(call.caller, operand)) && !call.available[operand.Value] {
				return l.unsupported(call.span, "call requires initialized captured binding "+operand.Value)
			}
		}
	}
	for i := range l.program.Procedures {
		for _, f := range l.registry.ordered {
			if f.label == l.program.Procedures[i].Label {
				l.program.Procedures[i].Globals = sortedGlobals(f.globals)
				l.program.Procedures[i].Captures = sortedGlobals(f.captures)
				break
			}
		}
	}
	return true
}

func sortedGlobals(globals map[string]Operand) []Operand {
	var result []Operand
	for _, operand := range globals {
		result = append(result, operand)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Value < result[j].Value })
	return result
}

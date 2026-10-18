package tac

import (
	"fmt"
	"sort"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type functionInfo struct {
	label   string
	typeOf  model.Type
	span    ast.Span
	edges   []*functionInfo
	globals map[string]Operand
	state   int
}

type mainCall struct {
	callee    *functionInfo
	span      ast.Span
	available map[string]bool
}

type functionRegistry struct {
	ordered []*functionInfo
	calls   []mainCall
}

func (l *lowerer) predeclare(global *environment, statements ast.Statements) bool {
	l.registry = &functionRegistry{}
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
				if symbol.Name == n.Name && symbol.Span == n.Span && symbol.Kind == model.SymbolFunction {
					copy := symbol
					found = &copy
					break
				}
			}
		}
		if found == nil || found.Type.Kind != model.TypeFunction || found.Type.Result == nil {
			return l.unsupported(n.Span, "missing matching global function signature")
		}
		if n.Body == nil {
			return l.unsupported(n.Span, "bodyless functions are unsupported")
		}
		if len(found.Type.Params) != len(n.Parameters) || (!scalar(found.Type.Result.Kind) && found.Type.Result.Kind != model.TypeNull) {
			return l.unsupported(n.Span, "unsupported function signature")
		}
		for i, parameter := range found.Type.Params {
			if n.Parameters[i].Type == nil || !scalar(parameter.Kind) {
				return l.unsupported(n.Span, "unsupported function signature")
			}
		}
		f := &functionInfo{label: symbolOperand(global.id, *found).Value, typeOf: found.Type, span: n.Span, globals: map[string]Operand{}}
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
	for i, parameter := range n.Parameters {
		found := false
		for _, scope := range l.scopes {
			if scope.ID != env.id {
				continue
			}
			for _, symbol := range scope.Symbols {
				if symbol.Name == parameter.Name && symbol.Kind == model.SymbolParameter && symbol.Type.Kind == f.typeOf.Params[i].Kind {
					operand := symbolOperand(env.id, symbol)
					env.active[symbol.Name] = binding{operand: operand, mutable: symbol.Mutable}
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
	return true
}

func (l *lowerer) returnStatement(env *environment, n ast.ReturnStmt) bool {
	if l.current == nil {
		return l.unsupported(n.Span, "return requires a function")
	}
	kind := l.current.typeOf.Result.Kind
	if kind == model.TypeNull {
		if n.Value != nil && !nullLiteral(n.Value) {
			return l.unsupported(n.Span, "null-result procedures only return null or no value")
		}
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: Return, Span: n.Span})
		return true
	}
	if n.Value == nil {
		return l.unsupported(n.Span, "scalar functions require a return value")
	}
	value, ok := l.expression(env, n.Value)
	if !ok {
		return false
	}
	if kind == model.TypeFloat {
		value = l.widen(value, n.Value.SourceSpan())
	}
	if value.Type != kind {
		return l.unsupported(n.Span, "return type mismatch")
	}
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
	}
}

func (l *lowerer) resolveCalls() bool {
	var visit func(*functionInfo) bool
	visit = func(f *functionInfo) bool {
		if f.state == 1 {
			return l.unsupported(f.span, "recursive call cycles are unsupported")
		}
		if f.state == 2 {
			return true
		}
		f.state = 1
		for _, callee := range f.edges {
			if !visit(callee) {
				return false
			}
			for name, operand := range callee.globals {
				f.globals[name] = operand
			}
		}
		f.state = 2
		return true
	}
	for _, f := range l.registry.ordered {
		if !visit(f) {
			return false
		}
	}
	for _, call := range l.registry.calls {
		for _, operand := range sortedGlobals(call.callee.globals) {
			if !call.available[operand.Value] {
				return l.unsupported(call.span, "call requires initialized global binding "+operand.Value)
			}
		}
	}
	for i := range l.program.Procedures {
		l.program.Procedures[i].Globals = sortedGlobals(l.registry.ordered[i].globals)
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

package tac

import (
	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

type binding struct {
	operand  Operand
	mutable  bool
	function *functionInfo
	typeOf   model.Type
}

type environment struct {
	id      int
	parent  *environment
	active  map[string]binding
	blocked map[string]bool
}

func (e *environment) lookup(name string) (binding, bool) {
	for current := e; current != nil; current = current.parent {
		if value, ok := current.active[name]; ok {
			return value, true
		}
		if current.blocked[name] {
			return binding{}, false
		}
	}
	return binding{}, false
}

func (e *environment) unavailable(name string) bool {
	for current := e; current != nil; current = current.parent {
		if _, ok := current.active[name]; ok {
			return false
		}
		if current.blocked[name] {
			return true
		}
	}
	return false
}

type lowerer struct {
	scopes      model.ScopeSnapshots
	program     Program
	pool        temporaryPool
	diagnostics model.Diagnostics
	nextLabel   int
	loops       []loopTargets
	registry    *functionRegistry
	current     *functionInfo
	globalID    int
}

// Generate lowers a semantically validated AST using its matching scope inventory.
// It never mutates its inputs and discards the whole program on unsupported input.
func Generate(program ast.Program, scopes model.ScopeSnapshots) (*Program, model.Diagnostics) {
	return generate(program, scopes, false)
}

func generate(program ast.Program, scopes model.ScopeSnapshots, forceClosures bool) (*Program, model.Diagnostics) {
	l := lowerer{scopes: scopes, program: Program{Instructions: []Instruction{}}, diagnostics: model.Diagnostics{}, registry: newFunctionRegistry(scopes)}
	l.registry.closures = l.registry.closures || forceClosures
	global := l.environment(nil, model.ScopeGlobal, program.Span)
	if global == nil {
		return nil, l.diagnostics
	}
	l.globalID = global.id
	if !l.statements(global, program.Statements) || !l.resolveCalls() {
		return nil, l.diagnostics
	}
	// A discarded function expression has no alias symbol in the inventory.
	// Retry once with explicit environments rather than altering legacy programs
	// merely because they declare ordinary direct-call functions.
	if l.registry.needsClosures && !l.registry.closures {
		return generate(program, scopes, true)
	}
	l.program.PeakTemporaries = l.pool.peak
	l.program.Environments = l.registry.environmentLayouts()
	return &l.program, l.diagnostics
}

func (l *lowerer) unsupported(span ast.Span, reason string) bool {
	l.diagnostics = append(l.diagnostics, model.Diagnostic{Code: "TAC_UNSUPPORTED", Phase: model.Phase("codegen"), Message: reason, Span: span})
	return false
}

func (l *lowerer) environment(parent *environment, kind model.ScopeKind, span ast.Span) *environment {
	parentID := 0
	if parent != nil {
		parentID = parent.id
	}
	for _, scope := range l.scopes {
		if scope.Kind == kind && scope.ParentID == parentID && scope.Span == span {
			l.enterEnvironment(scope)
			return &environment{id: scope.ID, parent: parent, active: map[string]binding{}}
		}
	}
	l.unsupported(span, "missing matching semantic scope")
	return nil
}

func (l *lowerer) statements(env *environment, statements ast.Statements) bool {
	if !l.predeclare(env, statements) {
		return false
	}
	for _, statement := range statements {
		if !l.statement(env, statement) {
			return false
		}
	}
	return true
}

func (l *lowerer) statement(env *environment, statement ast.Statement) bool {
	switch n := statement.(type) {
	case ast.FunctionDeclStmt:
		return l.functionBody(env, n)
	case ast.ReturnStmt:
		return l.returnStatement(env, n)
	case ast.VarDeclStmt:
		return l.declaration(env, n.Name, n.Initializer, n.Span)
	case ast.ConstDeclStmt:
		return l.declaration(env, n.Name, n.Value, n.Span)
	case ast.AssignStmt:
		target, ok := n.Target.(ast.IdentifierExpr)
		if !ok {
			return l.unsupported(n.Span, "only identifier assignment is supported")
		}
		value, ok := env.lookup(target.Name)
		if !ok && env.unavailable(target.Name) {
			return l.unsupported(target.Span, "switch binding is not guaranteed initialized on this path")
		}
		if !ok || !value.mutable {
			return l.unsupported(n.Span, "assignment requires an active mutable numeric binding")
		}
		l.reference(value.operand)
		return l.store(env, value.operand, n.Value, n.Span)
	case ast.BlockStmt:
		child := l.environment(env, model.ScopeBlock, n.Span)
		return child != nil && l.statements(child, n.Statements)
	case ast.IfStmt:
		return l.ifStatement(env, n)
	case ast.WhileStmt:
		return l.whileStatement(env, n)
	case ast.DoWhileStmt:
		return l.doWhileStatement(env, n)
	case ast.ForStmt:
		return l.forStatement(env, n)
	case ast.SwitchStmt:
		return l.switchStatement(env, n)
	case ast.BreakStmt:
		return l.loopTransfer(n.Span, false)
	case ast.ContinueStmt:
		return l.loopTransfer(n.Span, true)
	case ast.ExprStmt:
		if call, ok := statementCall(n.Expression); ok {
			value, success := l.call(env, call, true)
			if success {
				l.pool.release(value)
			}
			return success
		}
		value, ok := l.expression(env, n.Expression)
		if ok {
			l.pool.release(value)
		}
		return ok
	default:
		return l.unsupported(statement.SourceSpan(), "unsupported statement in numeric TAC")
	}
}

func (l *lowerer) declaration(env *environment, name string, initializer ast.Expression, span ast.Span) bool {
	if initializer == nil {
		return l.unsupported(span, "numeric declarations require an initializer")
	}
	for _, scope := range l.scopes {
		if scope.ID != env.id {
			continue
		}
		for _, symbol := range scope.Symbols {
			if symbol.Name != name || symbol.Span != span {
				continue
			}
			if !storedValue(symbol.Type.Kind) {
				return l.unsupported(span, "only integer, float and boolean declarations are supported")
			}
			target := symbolOperand(env.id, symbol)
			// Do not activate the new name until its initializer has read outer bindings.
			if !l.store(env, target, initializer, span) {
				return false
			}
			env.active[name] = binding{operand: target, mutable: symbol.Mutable, typeOf: symbol.Type}
			return true
		}
	}
	return l.unsupported(span, "missing matching semantic declaration")
}

func (l *lowerer) store(env *environment, target Operand, expression ast.Expression, span ast.Span) bool {
	value, ok := l.expression(env, expression)
	if !ok {
		return false
	}
	if target.Type == model.TypeFloat {
		value = l.widen(value, expression.SourceSpan())
	}
	if target.Type != value.Type {
		return l.unsupported(span, "numeric store type mismatch")
	}
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: Copy, Target: target, Left: value, Span: span})
	l.pool.release(value)
	return true
}

func numeric(kind model.TypeKind) bool { return kind == model.TypeInteger || kind == model.TypeFloat }

func scalar(kind model.TypeKind) bool { return numeric(kind) || kind == model.TypeBoolean }

// Storage capability is deliberately separate from scalar arithmetic capability.
func storedValue(kind model.TypeKind) bool { return scalar(kind) || kind == model.TypeFunction }

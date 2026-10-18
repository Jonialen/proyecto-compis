package tac

import (
	"sort"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// An entry allocates fresh, initially undefined cells on EVERY execution, not
// merely on function entry. A closure retains that exact lexical parent chain.
// Scope IDs identify layouts; they are not runtime allocation identities.
type EnvironmentLayout struct {
	ScopeID  int       `json:"scopeId"`
	ParentID int       `json:"parentId,omitempty"`
	Bindings []Operand `json:"bindings"`
}

func newFunctionRegistry(scopes model.ScopeSnapshots) *functionRegistry {
	r := &functionRegistry{environments: map[int]EnvironmentLayout{}}
	for _, scope := range scopes {
		for _, symbol := range scope.Symbols {
			if (symbol.Kind == model.SymbolFunction && scope.Kind != model.ScopeGlobal) || (symbol.Type.Kind == model.TypeFunction && symbol.Kind != model.SymbolFunction) {
				r.closures = true
			}
		}
	}
	return r
}

func (l *lowerer) enterEnvironment(scope model.ScopeSnapshot) {
	if !l.registry.closures {
		return
	}
	layout := EnvironmentLayout{ScopeID: scope.ID, ParentID: scope.ParentID, Bindings: []Operand{}}
	for _, symbol := range scope.Symbols {
		if symbol.Kind == model.SymbolVariable || symbol.Kind == model.SymbolConstant || symbol.Kind == model.SymbolParameter {
			layout.Bindings = append(layout.Bindings, symbolOperand(scope.ID, symbol))
		}
	}
	sort.Slice(layout.Bindings, func(i, j int) bool { return layout.Bindings[i].Value < layout.Bindings[j].Value })
	l.registry.environments[scope.ID] = layout
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: EnterEnvironment, Environment: scope.ID, Span: scope.Span})
}

func (r *functionRegistry) environmentLayouts() []EnvironmentLayout {
	var result []EnvironmentLayout
	for _, layout := range r.environments {
		result = append(result, layout)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ScopeID < result[j].ScopeID })
	return result
}

func (l *lowerer) closure(f *functionInfo, span ast.Span) Operand {
	l.registry.needsClosures = true
	target := l.pool.acquire(model.TypeFunction)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: MakeClosure, Target: target, Callee: f.label, Environment: f.definingScope, Span: span})
	return target
}

func operandScope(o Operand) int {
	if o.Kind != Symbol {
		return 0
	}
	id, _ := strconv.Atoi(strings.TrimPrefix(strings.SplitN(o.Value, ".", 2)[0], "s"))
	return id
}

func (l *lowerer) owns(f *functionInfo, o Operand) bool {
	id := operandScope(o)
	for id != 0 {
		found := false
		for _, scope := range l.scopes {
			if scope.ID != id {
				continue
			}
			if scope.Kind == model.ScopeFunction {
				return scope.ID == f.scopeID
			}
			id = scope.ParentID
			found = true
			break
		}
		if !found {
			break
		}
	}
	return false
}

func availableBindings(env *environment) map[string]bool {
	available := map[string]bool{}
	// Binding identities, not spelling: shadowing must not erase an initialized
	// outer cell that a predeclared sibling's declaration-time body references.
	for e := env; e != nil; e = e.parent {
		for name, b := range e.active {
			if b.function == nil && !e.blocked[name] {
				available[b.operand.Value] = true
			}
		}
	}
	return available
}

package tac

import (
	"fmt"
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
		if scope.Kind == model.ScopeClass {
			r.closures = true
		}
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
	if c := l.classScope(scope.ParentID); c != nil {
		layout.ParentID = c.owner
	}
	for _, symbol := range scope.Symbols {
		if symbol.Kind == model.SymbolVariable || symbol.Kind == model.SymbolConstant || symbol.Kind == model.SymbolParameter || symbol.Kind == model.SymbolCatch {
			o := symbolOperand(scope.ID, symbol)
			if symbol.Type.Kind == model.TypeList {
				o = l.listStorageType(scope.ID, o, symbol.Type)
			}
			if symbol.Type.Kind == model.TypeClass {
				o.Class = l.classForType(scope.ID, symbol.Type.Name)
			}
			layout.Bindings = append(layout.Bindings, o)
		}
	}
	// Receivers use the same layout builder and ordering invariant as ordinary
	// parameters. A nested function retains its enclosing receiver instead.
	var receiverClass *classInfo
	if l.current != nil {
		if scope.Kind == model.ScopeClass && scope.ID == l.current.scopeID {
			receiverClass = l.classScope(scope.ID)
		} else if l.current.method != nil && scope.ParentID == l.current.method.layout.ScopeID && scope.Kind == model.ScopeFunction {
			receiverClass = l.current.method
		}
	}
	if receiverClass != nil {
		layout.Bindings = append(layout.Bindings, Operand{Kind: Symbol, Value: fmt.Sprintf("s%d.$this", scope.ID), Type: model.TypeClass, Class: receiverClass.layout.ID})
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
		if id == f.scopeID {
			return true
		}
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

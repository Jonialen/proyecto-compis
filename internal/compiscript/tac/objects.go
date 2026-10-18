package tac

import (
	"fmt"
	"sort"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/model"
)

// Layouts describe cells, not live allocation identities or target byte offsets.
type ClassLayout struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Parent       string         `json:"parent,omitempty"`
	ScopeID      int            `json:"scopeId"`
	LexicalScope int            `json:"lexicalScope"`
	Span         ast.Span       `json:"span"`
	Fields       []FieldLayout  `json:"fields"`
	Methods      []MethodLayout `json:"methods"`
	Initializer  string         `json:"initializer"`
}
type FieldLayout struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Owner   string     `json:"owner"`
	Offset  int        `json:"offset"`
	Type    model.Type `json:"type"`
	Class   string     `json:"class,omitempty"`
	Shape   string     `json:"shape,omitempty"`
	Mutable bool       `json:"mutable"`
	Span    ast.Span   `json:"span"`
}
type MethodLayout struct {
	Name      string     `json:"name"`
	Callee    string     `json:"callee"`
	Owner     string     `json:"owner"`
	Signature model.Type `json:"signature"`
	Span      ast.Span   `json:"span"`
}
type classInfo struct {
	layout      ClassLayout
	declaration ast.ClassDeclStmt
	owner       int
	parent      *classInfo
	env         *environment
	ready       bool
}

func (l *lowerer) classID(id string) *classInfo {
	for _, c := range l.registry.classes {
		if c.layout.ID == id {
			return c
		}
	}
	return nil
}
func (l *lowerer) classScope(id int) *classInfo {
	for _, c := range l.registry.classes {
		if c.layout.ScopeID == id {
			return c
		}
	}
	return nil
}
func (l *lowerer) findClass(env *environment, name string) *classInfo {
	for current := env; current != nil; current = current.parent {
		if b, ok := current.active[name]; ok && b.class != nil {
			return b.class
		}
	}
	return nil
}
func (r *functionRegistry) classLayouts() []ClassLayout {
	var result []ClassLayout
	for _, c := range r.classes {
		result = append(result, c.layout)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ScopeID < result[j].ScopeID })
	return result
}

func (l *lowerer) predeclareClasses(env *environment, statements ast.Statements) bool {
	for _, s := range statements {
		n, ok := s.(ast.ClassDeclStmt)
		if !ok {
			continue
		}
		var scope *model.ScopeSnapshot
		for _, candidate := range l.scopes {
			if candidate.Kind == model.ScopeClass && candidate.ParentID == env.id && candidate.Span == n.Span {
				copy := candidate
				scope = &copy
				break
			}
		}
		if scope == nil {
			return l.unsupported(n.Span, "missing matching class scope")
		}
		c := &classInfo{declaration: n, owner: env.id, layout: ClassLayout{ID: fmt.Sprintf("s%d.%s", env.id, n.Name), Name: n.Name, ScopeID: scope.ID, LexicalScope: env.id, Span: n.Span, Fields: []FieldLayout{}, Methods: []MethodLayout{}, Initializer: fmt.Sprintf("s%d.$initialize", scope.ID)}}
		c.env = &environment{id: scope.ID, parent: env, active: map[string]binding{}}
		l.registry.classes = append(l.registry.classes, c)
		env.active[n.Name] = binding{class: c}
		l.registry.closures = true
	}
	for _, s := range statements {
		if n, ok := s.(ast.ClassDeclStmt); ok {
			if !l.prepareClass(l.findClass(env, n.Name)) {
				return false
			}
		}
	}
	return true
}

func (l *lowerer) prepareClass(c *classInfo) bool {
	if c.ready {
		return true
	}
	c.ready = true
	if c.declaration.Parent != "" {
		c.parent = l.findClass(c.env.parent, c.declaration.Parent)
		if c.parent == nil {
			return l.unsupported(c.layout.Span, "missing base class layout")
		}
		if !l.prepareClass(c.parent) {
			return false
		}
		c.layout.Parent = c.parent.layout.ID
		c.layout.Fields = append(c.layout.Fields, c.parent.layout.Fields...)
		c.layout.Methods = append(c.layout.Methods, c.parent.layout.Methods...)
	}
	if !l.predeclare(c.env, c.declaration.Members) {
		return false
	}
	for _, member := range c.declaration.Members {
		if n, ok := member.(ast.FunctionDeclStmt); ok {
			f := c.env.active[n.Name].function
			m := MethodLayout{Name: n.Name, Callee: f.label, Owner: c.layout.ID, Signature: f.typeOf, Span: n.Span}
			replaced := false
			for i, old := range c.layout.Methods {
				if old.Name == n.Name {
					c.layout.Methods[i] = m
					replaced = true
					break
				}
			}
			if !replaced {
				c.layout.Methods = append(c.layout.Methods, m)
			}
			continue
		}
		for _, scope := range l.scopes {
			if scope.ID != c.layout.ScopeID {
				continue
			}
			for _, symbol := range scope.Symbols {
				if symbol.Span != member.SourceSpan() || symbol.Kind != model.SymbolField {
					continue
				}
				if !storedValue(symbol.Type.Kind) {
					return l.unsupported(symbol.Span, "unsupported field type")
				}
				o := l.typedOperand(c.env, Operand{Type: symbol.Type.Kind}, symbol.Type)
				c.layout.Fields = append(c.layout.Fields, FieldLayout{ID: fmt.Sprintf("s%d.%s", scope.ID, symbol.Name), Name: symbol.Name, Owner: c.layout.ID, Offset: len(c.layout.Fields), Type: symbol.Type, Class: o.Class, Shape: o.Shape, Mutable: symbol.Mutable, Span: symbol.Span})
			}
		}
	}
	for i := range c.layout.Fields {
		field := &c.layout.Fields[i]
		if field.Owner == c.layout.ID {
			c.env.active[field.Name] = binding{field: field, mutable: field.Mutable, typeOf: field.Type}
		}
	}
	return true
}

func (l *lowerer) classBody(env *environment, n ast.ClassDeclStmt) bool {
	c := l.findClass(env, n.Name)
	if c == nil {
		return l.unsupported(n.Span, "missing class identity")
	}
	c.env.parent = env.fork()
	result := model.Type{Kind: model.TypeNull, Name: "null"}
	f := &functionInfo{label: c.layout.Initializer, typeOf: model.Type{Kind: model.TypeFunction, Result: &result}, globals: map[string]Operand{}, captures: map[string]Operand{}, definingScope: c.owner, scopeID: c.layout.ScopeID}
	l.registry.ordered = append(l.registry.ordered, f)
	child := lowerer{scopes: l.scopes, program: Program{Instructions: []Instruction{}}, registry: l.registry, current: f, globalID: l.globalID}
	for _, scope := range l.scopes {
		if scope.ID == c.layout.ScopeID {
			child.enterEnvironment(scope)
			break
		}
	}
	initialEnv := c.env.fork()
	receiver := Operand{Kind: Symbol, Value: fmt.Sprintf("s%d.$this", c.layout.ScopeID), Type: model.TypeClass, Class: c.layout.ID}
	initialEnv.active["$this"] = binding{operand: receiver}
	for _, member := range n.Members {
		var name string
		var value ast.Expression
		switch m := member.(type) {
		case ast.VarDeclStmt:
			name, value = m.Name, m.Initializer
		case ast.ConstDeclStmt:
			name, value = m.Name, m.Value
		}
		if value != nil {
			field := initialEnv.active[name].field
			stored, ok := child.fieldStore(initialEnv, receiver, *field, value, member.SourceSpan(), true)
			if !ok {
				l.diagnostics = append(l.diagnostics, child.diagnostics...)
				return false
			}
			child.pool.release(stored)
		}
	}
	child.program.Instructions = append(child.program.Instructions, Instruction{Op: Return, Span: n.Span})
	p := Procedure{Label: f.label, ScopeID: c.layout.ScopeID, Span: n.Span, ResultKind: model.TypeNull, Parameters: []Operand{}, Receiver: &receiver, LexicalScope: c.owner, Instructions: child.program.Instructions, PeakTemporaries: child.pool.peak}
	if !child.layout(&p) {
		l.diagnostics = append(l.diagnostics, child.diagnostics...)
		return false
	}
	l.program.Procedures = append(l.program.Procedures, p)
	for _, member := range n.Members {
		if method, ok := member.(ast.FunctionDeclStmt); ok {
			if !l.functionBody(c.env, method) {
				return false
			}
		}
	}
	return true
}

func (l *lowerer) thisValue(env *environment, span ast.Span) (Operand, bool) {
	b, ok := env.lookup("$this")
	if !ok {
		return Operand{}, l.unsupported(span, "missing lexical receiver")
	}
	l.reference(b.operand)
	return b.operand, true
}
func (l *lowerer) member(receiver Operand, name string) (*FieldLayout, *MethodLayout) {
	c := l.classID(receiver.Class)
	if c == nil {
		return nil, nil
	}
	for i := range c.layout.Fields {
		if c.layout.Fields[i].Name == name {
			return &c.layout.Fields[i], nil
		}
	}
	for i := range c.layout.Methods {
		if c.layout.Methods[i].Name == name {
			return nil, &c.layout.Methods[i]
		}
	}
	return nil, nil
}

func (l *lowerer) property(env *environment, receiver ast.Expression, name string, span ast.Span) (Operand, bool) {
	o, ok := l.expression(env, receiver)
	if !ok {
		return Operand{}, false
	}
	field, method := l.member(o, name)
	if field != nil {
		target := l.pool.acquire(field.Type.Kind)
		target.Class = field.Class
		target.Shape = field.Shape
		offset := field.Offset
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: LoadField, Target: target, Left: o, Class: o.Class, Offset: &offset, Member: field.ID, Span: span})
		l.pool.release(o)
		return target, true
	}
	if method != nil {
		return l.boundMethod(o, *method, span), true
	}
	return Operand{}, l.unsupported(span, "missing matching member layout")
}
func (l *lowerer) boundMethod(receiver Operand, method MethodLayout, span ast.Span) Operand {
	target := l.pool.acquire(model.TypeFunction)
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: BindMethod, Target: target, Left: receiver, Class: receiver.Class, Member: method.Name, Span: span})
	l.pool.release(receiver)
	return target
}
func (l *lowerer) propertyStore(env *environment, receiver ast.Expression, name string, value ast.Expression, span ast.Span) (Operand, bool) {
	o, ok := l.expression(env, receiver)
	if !ok {
		return Operand{}, false
	}
	field, _ := l.member(o, name)
	if field == nil {
		return Operand{}, l.unsupported(span, "assignment requires a field")
	}
	return l.fieldStore(env, o, *field, value, span, false)
}
func (l *lowerer) fieldStore(env *environment, receiver Operand, field FieldLayout, value ast.Expression, span ast.Span, initialize bool) (Operand, bool) {
	if !initialize && !field.Mutable {
		return Operand{}, l.unsupported(span, "assignment requires a mutable field")
	}
	receiver = l.freeze(receiver, span)
	expected := Operand{Type: field.Type.Kind, Class: field.Class, Shape: field.Shape}
	v, ok := l.expressionExpected(env, value, expected)
	if !ok {
		return Operand{}, false
	}
	if field.Type.Kind == model.TypeFloat {
		v = l.widen(v, value.SourceSpan())
	}
	if !l.assignable(expected, v) {
		return Operand{}, l.unsupported(span, "field store type mismatch")
	}
	offset := field.Offset
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: StoreField, Left: receiver, Right: &v, Class: receiver.Class, Offset: &offset, Member: field.ID, Span: span})
	l.pool.release(receiver)
	return v, true
}

func (l *lowerer) newObject(env *environment, n ast.NewExpr) (Operand, bool) {
	c := l.findClass(env, n.ClassName)
	if c == nil {
		return Operand{}, l.unsupported(n.Span, "missing class layout")
	}
	var constructor *MethodLayout
	for i := range c.layout.Methods {
		if c.layout.Methods[i].Name == "constructor" {
			constructor = &c.layout.Methods[i]
		}
	}
	params := model.Types{}
	if constructor != nil {
		params = constructor.Signature.Params
	}
	var typeClasses []string
	if constructor != nil {
		typeClasses = l.functionTypeClasses(constructor.Callee)
	}
	args, ok := l.arguments(env, n.Arguments, params, typeClasses)
	if !ok {
		return Operand{}, false
	}
	// Arguments are selected first and must survive initializer/constructor calls.
	for i := range args {
		args[i] = l.freeze(args[i], n.Span)
	}
	object := l.pool.acquire(model.TypeClass)
	object.Class = c.layout.ID
	l.program.Instructions = append(l.program.Instructions, Instruction{Op: NewObject, Target: object, Class: c.layout.ID, Span: n.Span})
	var chain []*classInfo
	for base := c; base != nil; base = base.parent {
		chain = append(chain, base)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		base := chain[i]
		selected := l.pool.acquire(model.TypeFunction)
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: BindMethod, Target: selected, Left: object, Class: base.layout.ID, Callee: base.layout.Initializer, Span: n.Span})
		null := model.Type{Kind: model.TypeNull, Name: "null"}
		signature := model.Type{Kind: model.TypeFunction, Result: &null}
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: CallIndirect, Left: selected, Signature: &signature, Span: n.Span})
		l.pool.release(selected)
	}
	if constructor != nil {
		selected := l.pool.acquire(model.TypeFunction)
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: BindMethod, Target: selected, Left: object, Class: c.layout.ID, Member: "constructor", Span: n.Span})
		l.program.Instructions = append(l.program.Instructions, Instruction{Op: CallIndirect, Left: selected, Arguments: args, Signature: &constructor.Signature, TypeClasses: typeClasses, Span: n.Span})
		l.pool.release(selected)
	}
	for _, arg := range args {
		l.pool.release(arg)
	}
	return object, true
}

func (l *lowerer) signatureClasses(env *environment, t model.Type) []string {
	classes := make([]string, len(t.Params)+1)
	used := false
	for i, p := range t.Params {
		classes[i] = l.typedOperand(env, Operand{}, p).Class
		used = used || classes[i] != ""
	}
	if t.Result != nil {
		classes[len(t.Params)] = l.typedOperand(env, Operand{}, *t.Result).Class
		used = used || classes[len(t.Params)] != ""
	}
	if !used {
		return nil
	}
	return classes
}

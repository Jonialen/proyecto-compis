package model

import "genanalex/internal/compiscript/ast"

type TypeKind string

const (
	TypeError     TypeKind = "error"
	TypeInteger   TypeKind = "integer"
	TypeFloat     TypeKind = "float"
	TypeBoolean   TypeKind = "boolean"
	TypeString    TypeKind = "string"
	TypeNull      TypeKind = "null"
	TypeList      TypeKind = "list"
	TypeClass     TypeKind = "class"
	TypeFunction  TypeKind = "function"
	TypeException TypeKind = "exception"
)

type SymbolKind string

const (
	SymbolVariable  SymbolKind = "variable"
	SymbolConstant  SymbolKind = "constant"
	SymbolParameter SymbolKind = "parameter"
	SymbolFunction  SymbolKind = "function"
	SymbolClass     SymbolKind = "class"
	SymbolField     SymbolKind = "field"
	SymbolMethod    SymbolKind = "method"
	SymbolCatch     SymbolKind = "catch"
)

type ScopeKind string

const (
	ScopeGlobal   ScopeKind = "global"
	ScopeClass    ScopeKind = "class"
	ScopeFunction ScopeKind = "function"
	ScopeBlock    ScopeKind = "block"
	ScopeCatch    ScopeKind = "catch"
)

type Phase string

const (
	PhaseLexical  Phase = "lexical"
	PhaseSyntax   Phase = "syntax"
	PhaseSemantic Phase = "semantic"
)

type Types []Type
type Symbols []Symbol
type ScopeSnapshots []ScopeSnapshot
type Diagnostics []Diagnostic
type ASTViews []ASTView

type Type struct {
	Kind    TypeKind `json:"kind"`
	Name    string   `json:"name"`
	Element *Type    `json:"element"`
	Params  Types    `json:"params"`
	Result  *Type    `json:"result"`
}
type Symbol struct {
	Name     string     `json:"name"`
	Kind     SymbolKind `json:"kind"`
	Type     Type       `json:"type"`
	Mutable  bool       `json:"mutable"`
	Captured bool       `json:"captured"`
	Span     ast.Span   `json:"span"`
}
type ScopeSnapshot struct {
	ID       int       `json:"id"`
	ParentID int       `json:"parentId"`
	Kind     ScopeKind `json:"kind"`
	Span     ast.Span  `json:"span"`
	Symbols  Symbols   `json:"symbols"`
}
type Diagnostic struct {
	Code    string   `json:"code"`
	Phase   Phase    `json:"phase"`
	Message string   `json:"message"`
	Span    ast.Span `json:"span"`
}
type ASTView struct {
	Kind     string   `json:"kind"`
	Label    string   `json:"label"`
	Span     ast.Span `json:"span"`
	Children ASTViews `json:"children"`
}
type AnalysisReport struct {
	AST         ASTView        `json:"ast"`
	Diagnostics Diagnostics    `json:"diagnostics"`
	Scopes      ScopeSnapshots `json:"scopes"`
}

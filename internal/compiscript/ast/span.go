package ast

type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Node interface{ SourceSpan() Span }
type Statement interface {
	Node
	isStatement()
}
type Expression interface {
	Node
	isExpression()
}

package ast

import "encoding/json"

type Statements []Statement
type Expressions []Expression
type Parameters []Parameter
type SwitchCases []SwitchCase

func (s Statements) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = Statements{}
	}
	return json.Marshal([]Statement(s))
}
func (s Expressions) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = Expressions{}
	}
	return json.Marshal([]Expression(s))
}
func (s Parameters) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = Parameters{}
	}
	return json.Marshal([]Parameter(s))
}
func (s SwitchCases) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = SwitchCases{}
	}
	return json.Marshal([]SwitchCase(s))
}

type Program struct {
	Span       Span       `json:"span"`
	Statements Statements `json:"statements"`
}
type TypeRef struct {
	Span       Span   `json:"span"`
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
}
type Parameter struct {
	Span Span     `json:"span"`
	Name string   `json:"name"`
	Type *TypeRef `json:"type"`
}
type SwitchCase struct {
	Span       Span       `json:"span"`
	Default    bool       `json:"default"`
	Value      Expression `json:"value"`
	Statements Statements `json:"statements"`
}

func (n Program) SourceSpan() Span    { return n.Span }
func (n TypeRef) SourceSpan() Span    { return n.Span }
func (n Parameter) SourceSpan() Span  { return n.Span }
func (n SwitchCase) SourceSpan() Span { return n.Span }

type BlockStmt struct {
	Span       Span       `json:"span"`
	Statements Statements `json:"statements"`
}
type VarDeclStmt struct {
	Span        Span       `json:"span"`
	Name        string     `json:"name"`
	Type        *TypeRef   `json:"type"`
	Initializer Expression `json:"initializer"`
}
type ConstDeclStmt struct {
	Span  Span       `json:"span"`
	Name  string     `json:"name"`
	Type  *TypeRef   `json:"type"`
	Value Expression `json:"value"`
}
type AssignStmt struct {
	Span   Span       `json:"span"`
	Target Expression `json:"target"`
	Value  Expression `json:"value"`
}
type FunctionDeclStmt struct {
	Span       Span       `json:"span"`
	Name       string     `json:"name"`
	Parameters Parameters `json:"parameters"`
	Result     *TypeRef   `json:"result"`
	Body       *BlockStmt `json:"body"`
}
type ClassDeclStmt struct {
	Span    Span       `json:"span"`
	Name    string     `json:"name"`
	Parent  string     `json:"parent"`
	Members Statements `json:"members"`
}
type ExprStmt struct {
	Span       Span       `json:"span"`
	Expression Expression `json:"expression"`
}
type PrintStmt struct {
	Span  Span       `json:"span"`
	Value Expression `json:"value"`
}
type IfStmt struct {
	Span      Span       `json:"span"`
	Condition Expression `json:"condition"`
	Then      *BlockStmt `json:"then"`
	Else      *BlockStmt `json:"else"`
}
type WhileStmt struct {
	Span      Span       `json:"span"`
	Condition Expression `json:"condition"`
	Body      *BlockStmt `json:"body"`
}
type DoWhileStmt struct {
	Span      Span       `json:"span"`
	Condition Expression `json:"condition"`
	Body      *BlockStmt `json:"body"`
}
type ForStmt struct {
	Span      Span       `json:"span"`
	Init      Statement  `json:"init"`
	Condition Expression `json:"condition"`
	Post      Expression `json:"post"`
	Body      *BlockStmt `json:"body"`
}
type ForeachStmt struct {
	Span     Span       `json:"span"`
	Name     string     `json:"name"`
	Iterable Expression `json:"iterable"`
	Body     *BlockStmt `json:"body"`
}
type TryCatchStmt struct {
	Span     Span       `json:"span"`
	Try      *BlockStmt `json:"try"`
	Name     string     `json:"name"`
	NameSpan Span       `json:"nameSpan"`
	Catch    *BlockStmt `json:"catch"`
}
type SwitchStmt struct {
	Span  Span        `json:"span"`
	Value Expression  `json:"value"`
	Cases SwitchCases `json:"cases"`
}
type BreakStmt struct {
	Span Span `json:"span"`
}
type ContinueStmt struct {
	Span Span `json:"span"`
}
type ReturnStmt struct {
	Span  Span       `json:"span"`
	Value Expression `json:"value"`
}
type BadStmt struct {
	Span Span `json:"span"`
}

func (n BlockStmt) SourceSpan() Span        { return n.Span }
func (n VarDeclStmt) SourceSpan() Span      { return n.Span }
func (n ConstDeclStmt) SourceSpan() Span    { return n.Span }
func (n AssignStmt) SourceSpan() Span       { return n.Span }
func (n FunctionDeclStmt) SourceSpan() Span { return n.Span }
func (n ClassDeclStmt) SourceSpan() Span    { return n.Span }
func (n ExprStmt) SourceSpan() Span         { return n.Span }
func (n PrintStmt) SourceSpan() Span        { return n.Span }
func (n IfStmt) SourceSpan() Span           { return n.Span }
func (n WhileStmt) SourceSpan() Span        { return n.Span }
func (n DoWhileStmt) SourceSpan() Span      { return n.Span }
func (n ForStmt) SourceSpan() Span          { return n.Span }
func (n ForeachStmt) SourceSpan() Span      { return n.Span }
func (n TryCatchStmt) SourceSpan() Span     { return n.Span }
func (n SwitchStmt) SourceSpan() Span       { return n.Span }
func (n BreakStmt) SourceSpan() Span        { return n.Span }
func (n ContinueStmt) SourceSpan() Span     { return n.Span }
func (n ReturnStmt) SourceSpan() Span       { return n.Span }
func (n BadStmt) SourceSpan() Span          { return n.Span }
func (BlockStmt) isStatement()              {}
func (VarDeclStmt) isStatement()            {}
func (ConstDeclStmt) isStatement()          {}
func (AssignStmt) isStatement()             {}
func (FunctionDeclStmt) isStatement()       {}
func (ClassDeclStmt) isStatement()          {}
func (ExprStmt) isStatement()               {}
func (PrintStmt) isStatement()              {}
func (IfStmt) isStatement()                 {}
func (WhileStmt) isStatement()              {}
func (DoWhileStmt) isStatement()            {}
func (ForStmt) isStatement()                {}
func (ForeachStmt) isStatement()            {}
func (TryCatchStmt) isStatement()           {}
func (SwitchStmt) isStatement()             {}
func (BreakStmt) isStatement()              {}
func (ContinueStmt) isStatement()           {}
func (ReturnStmt) isStatement()             {}
func (BadStmt) isStatement()                {}

type IdentifierExpr struct {
	Span Span   `json:"span"`
	Name string `json:"name"`
}
type LiteralExpr struct {
	Span   Span   `json:"span"`
	Lexeme string `json:"lexeme"`
}
type ArrayExpr struct {
	Span     Span        `json:"span"`
	Elements Expressions `json:"elements"`
}
type ThisExpr struct {
	Span Span `json:"span"`
}
type NewExpr struct {
	Span      Span        `json:"span"`
	ClassName string      `json:"className"`
	Arguments Expressions `json:"arguments"`
}
type GroupExpr struct {
	Span       Span       `json:"span"`
	Expression Expression `json:"expression"`
}
type UnaryExpr struct {
	Span     Span       `json:"span"`
	Operator string     `json:"operator"`
	Operand  Expression `json:"operand"`
}
type BinaryExpr struct {
	Span     Span       `json:"span"`
	Left     Expression `json:"left"`
	Operator string     `json:"operator"`
	Right    Expression `json:"right"`
}
type TernaryExpr struct {
	Span      Span       `json:"span"`
	Condition Expression `json:"condition"`
	Then      Expression `json:"then"`
	Else      Expression `json:"else"`
}
type AssignExpr struct {
	Span   Span       `json:"span"`
	Target Expression `json:"target"`
	Value  Expression `json:"value"`
}
type PropertyAssignExpr struct {
	Span     Span       `json:"span"`
	Receiver Expression `json:"receiver"`
	Name     string     `json:"name"`
	Value    Expression `json:"value"`
}
type CallExpr struct {
	Span      Span        `json:"span"`
	Callee    Expression  `json:"callee"`
	Arguments Expressions `json:"arguments"`
}
type IndexExpr struct {
	Span       Span       `json:"span"`
	Collection Expression `json:"collection"`
	Index      Expression `json:"index"`
}
type PropertyAccessExpr struct {
	Span     Span       `json:"span"`
	Receiver Expression `json:"receiver"`
	Name     string     `json:"name"`
}
type BadExpr struct {
	Span Span `json:"span"`
}

func (n IdentifierExpr) SourceSpan() Span     { return n.Span }
func (n LiteralExpr) SourceSpan() Span        { return n.Span }
func (n ArrayExpr) SourceSpan() Span          { return n.Span }
func (n ThisExpr) SourceSpan() Span           { return n.Span }
func (n NewExpr) SourceSpan() Span            { return n.Span }
func (n GroupExpr) SourceSpan() Span          { return n.Span }
func (n UnaryExpr) SourceSpan() Span          { return n.Span }
func (n BinaryExpr) SourceSpan() Span         { return n.Span }
func (n TernaryExpr) SourceSpan() Span        { return n.Span }
func (n AssignExpr) SourceSpan() Span         { return n.Span }
func (n PropertyAssignExpr) SourceSpan() Span { return n.Span }
func (n CallExpr) SourceSpan() Span           { return n.Span }
func (n IndexExpr) SourceSpan() Span          { return n.Span }
func (n PropertyAccessExpr) SourceSpan() Span { return n.Span }
func (n BadExpr) SourceSpan() Span            { return n.Span }
func (IdentifierExpr) isExpression()          {}
func (LiteralExpr) isExpression()             {}
func (ArrayExpr) isExpression()               {}
func (ThisExpr) isExpression()                {}
func (NewExpr) isExpression()                 {}
func (GroupExpr) isExpression()               {}
func (UnaryExpr) isExpression()               {}
func (BinaryExpr) isExpression()              {}
func (TernaryExpr) isExpression()             {}
func (AssignExpr) isExpression()              {}
func (PropertyAssignExpr) isExpression()      {}
func (CallExpr) isExpression()                {}
func (IndexExpr) isExpression()               {}
func (PropertyAccessExpr) isExpression()      {}
func (BadExpr) isExpression()                 {}

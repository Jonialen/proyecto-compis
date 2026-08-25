package ast

import (
	"reflect"
	"testing"
)

func TestSourceSpanPreservesBytePositions(t *testing.T) {
	tests := []struct {
		name string
		span Span
	}{
		{"multibyte half-open", Span{Start: Position{Offset: 1, Line: 1, Column: 2}, End: Position{Offset: 3, Line: 1, Column: 3}}},
		{"empty", Span{Start: Position{Offset: 4, Line: 2, Column: 1}, End: Position{Offset: 4, Line: 2, Column: 1}}},
		{"producer-owned coordinates", Span{Start: Position{Offset: -1}, End: Position{Offset: -1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Program{Span: tt.span}).SourceSpan(); !reflect.DeepEqual(got, tt.span) {
				t.Fatalf("SourceSpan() = %#v, want %#v", got, tt.span)
			}
		})
	}
}

func TestEveryNodeStoresItsSpan(t *testing.T) {
	span := Span{Start: Position{Line: 1, Column: 1}, End: Position{Offset: 1, Line: 1, Column: 2}}
	identifier := IdentifierExpr{Span: span, Name: "x"}
	block := BlockStmt{Span: span, Statements: Statements{BreakStmt{Span: span}}}
	nodes := []Node{
		Program{Span: span}, TypeRef{Span: span}, Parameter{Span: span}, SwitchCase{Span: span},
		block, VarDeclStmt{Span: span}, ConstDeclStmt{Span: span}, AssignStmt{Span: span}, FunctionDeclStmt{Span: span}, ClassDeclStmt{Span: span}, ExprStmt{Span: span}, PrintStmt{Span: span}, IfStmt{Span: span}, WhileStmt{Span: span}, DoWhileStmt{Span: span}, ForStmt{Span: span}, ForeachStmt{Span: span}, TryCatchStmt{Span: span}, SwitchStmt{Span: span}, BreakStmt{Span: span}, ContinueStmt{Span: span}, ReturnStmt{Span: span}, BadStmt{Span: span},
		identifier, LiteralExpr{Span: span}, ArrayExpr{Span: span}, ThisExpr{Span: span}, NewExpr{Span: span}, GroupExpr{Span: span}, UnaryExpr{Span: span}, BinaryExpr{Span: span}, TernaryExpr{Span: span}, AssignExpr{Span: span}, PropertyAssignExpr{Span: span}, CallExpr{Span: span}, IndexExpr{Span: span}, PropertyAccessExpr{Span: span}, BadExpr{Span: span},
	}
	for _, node := range nodes {
		if got := node.SourceSpan(); got != span {
			t.Fatalf("%T SourceSpan() = %#v, want %#v", node, got, span)
		}
	}
}

func TestTypedChildrenUseCategories(t *testing.T) {
	span := Span{}
	statement := Statement(BlockStmt{Span: span})
	expression := Expression(IdentifierExpr{Span: span, Name: "value"})
	function := FunctionDeclStmt{
		Span:       span,
		Parameters: Parameters{{Span: span, Name: "arg", Type: &TypeRef{Span: span, Name: "integer"}}},
		Body:       &BlockStmt{Span: span, Statements: Statements{statement}},
	}
	call := CallExpr{Span: span, Callee: expression, Arguments: Expressions{expression}}
	if function.Body.Statements[0].SourceSpan() != span || call.Arguments[0].SourceSpan() != span {
		t.Fatal("typed children did not retain category values")
	}
}

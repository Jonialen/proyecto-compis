package frontend

import (
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend/generated"
	"github.com/antlr4-go/antlr/v4"
)

func TestMapExpressionAlternatives(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"assignment", "target = 3"}, {"property assignment", "object.field = 3"}, {"ternary", "ready ? 1 : 2"},
		{"logical or", "left || right"}, {"logical and", "left && right"}, {"equal", "left == right"}, {"not equal", "left != right"},
		{"less than", "left < right"}, {"less or equal", "left <= right"}, {"greater than", "left > right"}, {"greater or equal", "left >= right"},
		{"addition", "left + right"}, {"subtraction", "left - right"}, {"multiplication", "left * right"}, {"division", "left / right"}, {"remainder", "left % right"},
		{"negative", "-value"}, {"not", "!value"}, {"group", "(value)"}, {"identifier", "value"}, {"new", "new Widget(1, 2)"},
		{"this", "this"}, {"call suffix", "object.method(1)"}, {"index suffix", "items[0]"}, {"property suffix", "object.field"}, {"array", "[1, 2]"},
		{"null", "null"}, {"true", "true"}, {"false", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapExpressionSource(t, tt.source)
			if want := expectedExpression(tt.source); !reflect.DeepEqual(got, want) {
				t.Fatalf("expression = %#v, want %#v", got, want)
			}
		})
	}
}

func TestMapTypesAndFloatLiterals(t *testing.T) {
	for _, source := range []string{"0.0", "3.14", "0", "42", `"text"`} {
		t.Run(source, func(t *testing.T) {
			literal, ok := mapExpressionSource(t, source).(ast.LiteralExpr)
			if !ok || literal.Lexeme != source {
				t.Fatalf("literal = %#v, want lexeme %q", literal, source)
			}
		})
	}

	for _, tt := range []struct {
		source     string
		wantName   string
		wantDimens int
	}{
		{"boolean", "boolean", 0},
		{"integer", "integer", 0},
		{"float", "float", 0},
		{"string", "string", 0},
		{"Widget[][]", "Widget", 2},
	} {
		t.Run(tt.source, func(t *testing.T) {
			got := mapTypeSource(t, tt.source)
			want := ast.TypeRef{Span: testSpan(tt.source, 0, len(tt.source)), Name: tt.wantName, Dimensions: tt.wantDimens}
			if got != want {
				t.Fatalf("type = %#v, want %#v", got, want)
			}
		})
	}
}

func mapExpressionSource(t *testing.T, source string) ast.Expression {
	t.Helper()
	parser := generated.NewCompiscriptParser(antlr.NewCommonTokenStream(
		generated.NewCompiscriptLexer(antlr.NewInputStream(source)), antlr.TokenDefaultChannel,
	))
	return newExpressionMapper([]byte(source)).mapExpression(parser.Expression())
}

func mapTypeSource(t *testing.T, source string) ast.TypeRef {
	t.Helper()
	parser := generated.NewCompiscriptParser(antlr.NewCommonTokenStream(
		generated.NewCompiscriptLexer(antlr.NewInputStream(source)), antlr.TokenDefaultChannel,
	))
	return newExpressionMapper([]byte(source)).mapType(parser.Type_())
}

func expressionOperator(expression ast.Expression) string {
	if unary, ok := expression.(ast.UnaryExpr); ok {
		return unary.Operator
	}
	if binary, ok := expression.(ast.BinaryExpr); ok {
		return binary.Operator
	}
	return ""
}

func expectedExpression(source string) ast.Expression {
	whole := testSpan(source, 0, len(source))
	id := func(name string) ast.IdentifierExpr {
		return ast.IdentifierExpr{Span: testSpan(source, strings.Index(source, name), strings.Index(source, name)+len(name)), Name: name}
	}
	literal := func(text string) ast.LiteralExpr {
		return ast.LiteralExpr{Span: testSpan(source, strings.LastIndex(source, text), strings.LastIndex(source, text)+len(text)), Lexeme: text}
	}
	if fields := strings.Fields(source); len(fields) == 3 && isBinaryOperator(fields[1]) {
		return ast.BinaryExpr{Span: whole, Left: id(fields[0]), Operator: fields[1], Right: id(fields[2])}
	}
	switch source {
	case "target = 3":
		return ast.AssignExpr{Span: whole, Target: id("target"), Value: literal("3")}
	case "object.field = 3":
		return ast.PropertyAssignExpr{Span: whole, Receiver: id("object"), Name: "field", Value: literal("3")}
	case "ready ? 1 : 2":
		return ast.TernaryExpr{Span: whole, Condition: id("ready"), Then: literal("1"), Else: literal("2")}
	case "-value", "!value":
		return ast.UnaryExpr{Span: whole, Operator: source[:1], Operand: id("value")}
	case "(value)":
		return ast.GroupExpr{Span: whole, Expression: id("value")}
	case "value":
		return id("value")
	case "new Widget(1, 2)":
		return ast.NewExpr{Span: whole, ClassName: "Widget", Arguments: ast.Expressions{literal("1"), literal("2")}}
	case "this":
		return ast.ThisExpr{Span: whole}
	case "object.method(1)":
		return ast.CallExpr{Span: whole, Callee: ast.PropertyAccessExpr{Span: testSpan(source, 0, 13), Receiver: id("object"), Name: "method"}, Arguments: ast.Expressions{literal("1")}}
	case "items[0]":
		return ast.IndexExpr{Span: whole, Collection: id("items"), Index: literal("0")}
	case "object.field":
		return ast.PropertyAccessExpr{Span: whole, Receiver: id("object"), Name: "field"}
	case "[1, 2]":
		return ast.ArrayExpr{Span: whole, Elements: ast.Expressions{literal("1"), literal("2")}}
	default:
		return literal(source)
	}
}

func isBinaryOperator(operator string) bool {
	switch operator {
	case "||", "&&", "==", "!=", "<", "<=", ">", ">=", "+", "-", "*", "/", "%":
		return true
	default:
		return false
	}
}

func testSpan(source string, start, end int) ast.Span {
	return ast.Span{Start: ast.Position{Offset: start, Line: 1, Column: start + 1}, End: ast.Position{Offset: end, Line: 1, Column: end + 1}}
}

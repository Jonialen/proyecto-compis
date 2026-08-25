package model

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"genanalex/internal/compiscript/ast"
)

func TestAnalysisReportJSONIsOrderedAndDeterministic(t *testing.T) {
	span := ast.Span{Start: ast.Position{Line: 1, Column: 1}, End: ast.Position{Offset: 1, Line: 1, Column: 2}}
	typeValue := Type{Kind: TypeFunction, Name: "fn", Element: &Type{Kind: TypeList, Name: "items"}, Params: Types{{Kind: TypeString, Name: "string"}, {Kind: TypeBoolean, Name: "boolean"}}, Result: &Type{Kind: TypeInteger, Name: "integer"}}
	report := AnalysisReport{
		AST:         ASTView{Kind: "program", Label: "Program", Span: span, Children: ASTViews{{Kind: "literal", Label: "one", Span: span, Children: ASTViews{}}, {Kind: "literal", Label: "two", Span: span, Children: ASTViews{{Kind: "literal", Label: "nested", Span: span}}}}},
		Diagnostics: Diagnostics{{Code: "E2", Phase: PhaseSemantic, Message: "second", Span: span}, {Code: "E1", Phase: PhaseSyntax, Message: "first", Span: span}},
		Scopes:      ScopeSnapshots{{ID: 2, ParentID: 1, Kind: ScopeFunction, Span: span, Symbols: Symbols{{Name: "z", Kind: SymbolVariable, Type: typeValue, Span: span}, {Name: "a", Kind: SymbolConstant, Type: Type{Kind: TypeFloat, Name: "float"}, Mutable: true, Captured: true, Span: span}}}},
	}
	first, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("JSON changed between encodes: %s != %s", first, second)
	}
	want := `{"ast":{"kind":"program","label":"Program","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}},"children":[{"kind":"literal","label":"one","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}},"children":[]},{"kind":"literal","label":"two","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}},"children":[{"kind":"literal","label":"nested","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}},"children":[]}]}]},"diagnostics":[{"code":"E2","phase":"semantic","message":"second","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}}},{"code":"E1","phase":"syntax","message":"first","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}}}],"scopes":[{"id":2,"parentId":1,"kind":"function","span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}},"symbols":[{"name":"z","kind":"variable","type":{"kind":"function","name":"fn","element":{"kind":"list","name":"items","element":null,"params":[],"result":null},"params":[{"kind":"string","name":"string","element":null,"params":[],"result":null},{"kind":"boolean","name":"boolean","element":null,"params":[],"result":null}],"result":{"kind":"integer","name":"integer","element":null,"params":[],"result":null}},"mutable":false,"captured":false,"span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}}},{"name":"a","kind":"constant","type":{"kind":"float","name":"float","element":null,"params":[],"result":null},"mutable":true,"captured":true,"span":{"start":{"offset":0,"line":1,"column":1},"end":{"offset":1,"line":1,"column":2}}}]}]}`
	if string(first) != want {
		t.Errorf("JSON tree = %s, want %s", first, want)
	}
}

func TestAllEnumValuesEncodeExactly(t *testing.T) {
	values := []any{
		[]TypeKind{TypeError, TypeInteger, TypeFloat, TypeBoolean, TypeString, TypeNull, TypeList, TypeClass, TypeFunction, TypeException},
		[]SymbolKind{SymbolVariable, SymbolConstant, SymbolParameter, SymbolFunction, SymbolClass, SymbolField, SymbolMethod, SymbolCatch},
		[]ScopeKind{ScopeGlobal, ScopeClass, ScopeFunction, ScopeBlock, ScopeCatch},
		[]Phase{PhaseLexical, PhaseSyntax, PhaseSemantic},
	}
	want := []string{`["error","integer","float","boolean","string","null","list","class","function","exception"]`, `["variable","constant","parameter","function","class","field","method","catch"]`, `["global","class","function","block","catch"]`, `["lexical","syntax","semantic"]`}
	for i, value := range values {
		got, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want[i] {
			t.Errorf("enum JSON = %s, want %s", got, want[i])
		}
	}
}

func TestNamedCollectionsEncodeArraysAndPreserveOrder(t *testing.T) {
	span := func(offset int) ast.Span {
		return ast.Span{Start: ast.Position{Offset: offset, Line: 1, Column: 1}, End: ast.Position{Offset: offset + 1, Line: 1, Column: 2}}
	}
	cases := []struct {
		name                          string
		nilValue, emptyValue, ordered any
		first, second                 string
	}{
		{"statements", ast.Statements(nil), ast.Statements{}, ast.Statements{ast.BreakStmt{Span: span(1)}, ast.ContinueStmt{Span: span(2)}}, `"offset":1`, `"offset":2`},
		{"expressions", ast.Expressions(nil), ast.Expressions{}, ast.Expressions{ast.IdentifierExpr{Name: "first"}, ast.IdentifierExpr{Name: "second"}}, `"name":"first"`, `"name":"second"`},
		{"parameters", ast.Parameters(nil), ast.Parameters{}, ast.Parameters{{Name: "first"}, {Name: "second"}}, `"name":"first"`, `"name":"second"`},
		{"switch cases", ast.SwitchCases(nil), ast.SwitchCases{}, ast.SwitchCases{{Value: ast.LiteralExpr{Lexeme: "first"}}, {Value: ast.LiteralExpr{Lexeme: "second"}}}, `"lexeme":"first"`, `"lexeme":"second"`},
		{"type parameters", Types(nil), Types{}, Types{{Name: "first"}, {Name: "second"}}, `"name":"first"`, `"name":"second"`},
		{"symbols", Symbols(nil), Symbols{}, Symbols{{Name: "first"}, {Name: "second"}}, `"name":"first"`, `"name":"second"`},
		{"scopes", ScopeSnapshots(nil), ScopeSnapshots{}, ScopeSnapshots{{ID: 1}, {ID: 2}}, `"id":1`, `"id":2`},
		{"diagnostics", Diagnostics(nil), Diagnostics{}, Diagnostics{{Code: "first"}, {Code: "second"}}, `"code":"first"`, `"code":"second"`},
		{"AST views", ASTViews(nil), ASTViews{}, ASTViews{{Label: "first"}, {Label: "second", Children: ASTViews{{Label: "nested", Children: ASTViews{}}}}}, `"label":"first"`, `"label":"second"`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, value := range []any{tt.nilValue, tt.emptyValue} {
				data, err := json.Marshal(value)
				if err != nil || string(data) != "[]" {
					t.Fatalf("empty JSON = %s, %v; want []", data, err)
				}
			}
			data, err := json.Marshal(tt.ordered)
			if err != nil || strings.Index(string(data), tt.first) >= strings.Index(string(data), tt.second) {
				t.Fatalf("ordered JSON = %s, %v", data, err)
			}
		})
	}
}

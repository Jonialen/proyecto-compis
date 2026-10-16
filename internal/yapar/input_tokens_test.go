package yapar

import (
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/shared"
)

func TestParsersRejectSuppliedEndMarker(t *testing.T) {
	g := mustBuildGrammar(t, "%token INT PLUS\n%%\nstart : INT | ;")
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []Method{MethodLL1, MethodSLR, MethodLALR} {
		t.Run(string(method), func(t *testing.T) {
			parser, err := BuildParser(g, ff, method)
			if err != nil {
				t.Fatal(err)
			}
			for _, tt := range []struct {
				name, location string
				tokens         []shared.Token
			}{
				{name: "marker only", tokens: []shared.Token{{Type: EndMarker, Line: 3}}},
				{name: "leading marker", tokens: []shared.Token{{Type: EndMarker, Line: 3}, {Type: "INT", Line: 7}}},
				{name: "trailing marker", tokens: []shared.Token{{Type: "INT"}, {Type: EndMarker, Line: 3}}},
				{name: "premature marker", tokens: []shared.Token{{Type: "INT"}, {Type: EndMarker, Line: 3}, {Type: "PLUS", Line: 7}}},
				{name: "marker line fallback", location: "line 9", tokens: []shared.Token{{Type: EndMarker}, {Type: "INT", Line: 9}}},
			} {
				t.Run(tt.name, func(t *testing.T) {
					before := append([]shared.Token(nil), tt.tokens...)
					result, err := parser.Parse(tt.tokens)
					location := tt.location
					if location == "" {
						location = "line 3"
					}
					if result == nil || result.Accepted || err == nil || !strings.Contains(err.Error(), "reserved for end of input") || !strings.Contains(err.Error(), location) {
						t.Fatalf("Parse() = (%#v, %v), want rejection of external EOF at %s", result, err, location)
					}
					if !reflect.DeepEqual(tt.tokens, before) {
						t.Fatal("Parse() modified the input token stream")
					}
				})
			}
			for _, tokens := range [][]shared.Token{nil, {{Type: "INT", Lexeme: "$", Line: 1}}} {
				result, err := parser.Parse(tokens)
				if err != nil || result == nil || !result.Accepted {
					t.Fatalf("Parse(%v) = (%#v, %v), want implicit EOF and dollar lexemes preserved", tokens, result, err)
				}
			}
		})
	}
}

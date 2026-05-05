package yapar

import (
	"errors"
	"reflect"
	"testing"

	"genanalex/internal/shared"
)

func TestParseTokensAcceptsValidInput(t *testing.T) {
	g, table := mustBuildParsingTable(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)

	result, err := ParseTokens(g, table, []shared.Token{
		{Type: "C", Lexeme: "C", Line: 1},
		{Type: "D", Lexeme: "D", Line: 1},
		{Type: "D", Lexeme: "D", Line: 1},
	})
	if err != nil {
		t.Fatalf("ParseTokens() error = %v", err)
	}
	if result == nil || !result.Accepted {
		t.Fatalf("ParseTokens() = %#v, want Accepted=true", result)
	}
}

func TestParseTokensRejectsInvalidInput(t *testing.T) {
	g, table := mustBuildParsingTable(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	result, err := ParseTokens(g, table, []shared.Token{
		{Type: "ID", Lexeme: "x", Line: 7},
		{Type: "ID", Lexeme: "y", Line: 7},
	})
	if result == nil || result.Accepted {
		t.Fatalf("ParseTokens() result = %#v, want Accepted=false", result)
	}
	if err == nil {
		t.Fatal("ParseTokens() error = nil, want syntax error")
	}

	var syntaxErr *SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("error type = %T, want *SyntaxError", err)
	}
	if syntaxErr.Line != 7 {
		t.Fatalf("syntaxErr.Line = %d, want 7", syntaxErr.Line)
	}
	if syntaxErr.GotType != "ID" {
		t.Fatalf("syntaxErr.GotType = %q, want ID", syntaxErr.GotType)
	}
	if syntaxErr.Lexeme != "y" {
		t.Fatalf("syntaxErr.Lexeme = %q, want y", syntaxErr.Lexeme)
	}
	if got, want := syntaxErr.Expected, []string{"PLUS"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("syntaxErr.Expected = %v, want %v", got, want)
	}
}

func TestParseTokensIgnoresConfiguredTokens(t *testing.T) {
	g, table := mustBuildParsingTable(t, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)

	result, err := ParseTokens(g, table, []shared.Token{
		{Type: "WS", Lexeme: " ", Line: 3},
		{Type: "ID", Lexeme: "x", Line: 3},
		{Type: "WS", Lexeme: " ", Line: 3},
		{Type: "PLUS", Lexeme: "+", Line: 3},
		{Type: "WS", Lexeme: " ", Line: 3},
		{Type: "ID", Lexeme: "y", Line: 3},
		{Type: "WS", Lexeme: " ", Line: 3},
	})
	if err != nil {
		t.Fatalf("ParseTokens() error = %v", err)
	}
	if result == nil || !result.Accepted {
		t.Fatalf("ParseTokens() = %#v, want Accepted=true", result)
	}
}

func mustBuildParsingTable(t *testing.T, input string) (*Grammar, *ParsingTable) {
	t.Helper()

	g := mustBuildGrammar(t, input)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	states, transitions, err := BuildCanonicalCollection(g)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}
	table, err := BuildSLRTable(g, ff, states, transitions)
	if err != nil {
		t.Fatalf("BuildSLRTable() error = %v", err)
	}

	return g, table
}

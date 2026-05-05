package yapar

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseYalpSpec(t *testing.T) {
	input := `/* comentario de encabezado */
%token ID PLUS TIMES LPAREN RPAREN WS
IGNORE WS

%%

/* comentario
multilinea */
expr : expr PLUS term
	     | term
	     ;
term : term TIMES factor | factor ;
factor : ID | LPAREN expr RPAREN ;
`

	spec, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if spec.StartSymbol != "expr" {
		t.Fatalf("StartSymbol = %q, want %q", spec.StartSymbol, "expr")
	}

	wantTokens := []string{"ID", "PLUS", "TIMES", "LPAREN", "RPAREN", "WS"}
	if !reflect.DeepEqual(spec.Tokens, wantTokens) {
		t.Fatalf("Tokens = %v, want %v", spec.Tokens, wantTokens)
	}

	if !spec.IgnoreTokens["WS"] {
		t.Fatalf("IgnoreTokens = %v, want WS ignored", spec.IgnoreTokens)
	}

	wantProductions := []RawProduction{
		{Head: "expr", Bodies: [][]string{{"expr", "PLUS", "term"}, {"term"}}},
		{Head: "term", Bodies: [][]string{{"term", "TIMES", "factor"}, {"factor"}}},
		{Head: "factor", Bodies: [][]string{{"ID"}, {"LPAREN", "expr", "RPAREN"}}},
	}
	if !reflect.DeepEqual(spec.Productions, wantProductions) {
		t.Fatalf("Productions = %#v, want %#v", spec.Productions, wantProductions)
	}
}

func TestParseAllowsEmptyAlternative(t *testing.T) {
	input := `%token ID
%%
optional : ID |
         ;
`

	spec, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := RawProduction{Head: "optional", Bodies: [][]string{{"ID"}, {}}}
	if !reflect.DeepEqual(spec.Productions[0], want) {
		t.Fatalf("production = %#v, want %#v", spec.Productions[0], want)
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "parser.yalp")
	content := `%token ID
%%
start : ID ;
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	spec, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if spec.StartSymbol != "start" {
		t.Fatalf("StartSymbol = %q, want start", spec.StartSymbol)
	}
}

func TestParseSpecErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMessage string
		wantLine    int
	}{
		{
			name:        "missing separator",
			input:       `%token ID
start : ID ;`,
			wantMessage: "missing %% section separator",
		},
		{
			name: "unterminated comment",
			input: `/* comentario
sin cierre
%token ID
%%
start : ID ;`,
			wantMessage: "unterminated block comment",
			wantLine:    1,
		},
		{
			name: "duplicate token",
			input: `%token ID ID
%%
start : ID ;`,
			wantMessage: `token "ID" declared more than once`,
			wantLine:    1,
		},
		{
			name: "unknown header directive",
			input: `%start expr
%%
expr : ID ;`,
			wantMessage: `unexpected header directive "%start"`,
			wantLine:    1,
		},
		{
			name: "missing colon",
			input: `%token ID
%%
start ID ;`,
			wantMessage: `expected ':' after production head "start", got "ID"`,
			wantLine:    3,
		},
		{
			name: "unexpected colon in body",
			input: `%token ID
%%
start : ID : ID ;`,
			wantMessage: `unexpected ':' inside production "start"`,
			wantLine:    3,
		},
		{
			name: "missing semicolon",
			input: `%token ID
%%
start : ID`,
			wantMessage: `production "start" must end with ';'`,
			wantLine:    3,
		},
		{
			name: "multiple separators",
			input: `%token ID
%%
start : ID ;
%%`,
			wantMessage: "multiple %% section separators are not allowed",
			wantLine:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Fatal("Parse() error = nil, want error")
			}

			specErr, ok := err.(*SpecError)
			if !ok {
				t.Fatalf("error type = %T, want *SpecError", err)
			}
			if !strings.Contains(specErr.Error(), tt.wantMessage) {
				t.Fatalf("error = %q, want message containing %q", specErr.Error(), tt.wantMessage)
			}
			if specErr.Line != tt.wantLine {
				t.Fatalf("error line = %d, want %d", specErr.Line, tt.wantLine)
			}
		})
	}
}

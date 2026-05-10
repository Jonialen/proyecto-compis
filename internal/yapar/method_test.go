package yapar

import (
	"errors"
	"reflect"
	"testing"

	"genanalex/internal/shared"
)

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Method
		wantErr bool
	}{
		{name: "slr", input: "slr", want: MethodSLR},
		{name: "ll1", input: "ll1", want: MethodLL1},
		{name: "lalr", input: "lalr", want: MethodLALR},
		{name: "invalid", input: "foo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMethod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("ParseMethod() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildParser(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	parser, err := BuildParser(g, ff, MethodSLR)
	if err != nil {
		t.Fatalf("BuildParser() error = %v", err)
	}
	if parser == nil {
		t.Fatal("BuildParser() = nil, want executable parser")
	}
	if parser.Table() == nil {
		t.Fatal("parser.Table() = nil, want table view")
	}

	for _, method := range []Method{MethodLL1, MethodLALR} {
		t.Run(string(method), func(t *testing.T) {
			parser, err := BuildParser(g, ff, method)
			if parser != nil {
				t.Fatalf("BuildParser() parser = %#v, want nil", parser)
			}
			if !errors.Is(err, ErrNotImplemented) {
				t.Fatalf("BuildParser() error = %v, want ErrNotImplemented", err)
			}
		})
	}
}

func TestExecutableParserParseMatchesParseTokens(t *testing.T) {
	g, table := mustBuildParsingTable(t, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	parser, err := BuildParser(g, ff, MethodSLR)
	if err != nil {
		t.Fatalf("BuildParser() error = %v", err)
	}

	tokens := []shared.Token{
		{Type: "WS", Lexeme: " ", Line: 3},
		{Type: "ID", Lexeme: "x", Line: 3},
		{Type: "PLUS", Lexeme: "+", Line: 3},
		{Type: "ID", Lexeme: "y", Line: 3},
	}

	gotResult, gotErr := parser.Parse(tokens)
	wantResult, wantErr := ParseTokens(g, table, tokens)
	if !reflect.DeepEqual(gotResult, wantResult) {
		t.Fatalf("parser.Parse() result = %#v, want %#v", gotResult, wantResult)
	}
	if !reflect.DeepEqual(gotErr, wantErr) {
		t.Fatalf("parser.Parse() error = %#v, want %#v", gotErr, wantErr)
	}
}

func TestSLRTableViewExposesActionGoto(t *testing.T) {
	g, table := mustBuildParsingTable(t, `%token C D
%%
s : c c ;
c : C c | D ;
`)

	view := (&slrTableView{grammar: g, table: table})
	if kind, target, ok := view.ActionAt(0, "C"); !ok || kind != ActionShift || target == 0 {
		t.Fatalf("ActionAt(0, C) = (%v, %d, %v), want shift action with target state", kind, target, ok)
	}
	if !assertHasReduceAction(t, view, table) {
		t.Fatal("ActionAt() did not expose any reduce action from the SLR table")
	}
	if got, ok := view.GotoAt(0, "c"); !ok || got == 0 {
		t.Fatalf("GotoAt(0, c) = (%d, %v), want target state", got, ok)
	}
	if got, want := view.Terminals(), []string{"$", "C", "D"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Terminals() = %v, want %v", got, want)
	}
	if got, want := view.NonTerminals(), []string{"c", "s"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NonTerminals() = %v, want %v", got, want)
	}
	if len(view.States()) == 0 {
		t.Fatal("States() = empty, want parser states")
	}
}

func assertHasReduceAction(t *testing.T, view TableView, table *ParsingTable) bool {
	t.Helper()
	for state, row := range table.Action {
		for symbol, action := range row {
			if action.Kind != ActionReduce {
				continue
			}
			kind, target, ok := view.ActionAt(state, symbol)
			if !ok {
				t.Fatalf("ActionAt(%d, %q) = missing, want reduce action", state, symbol)
			}
			if kind != ActionReduce || target != action.ProductionID {
				t.Fatalf("ActionAt(%d, %q) = (%v, %d, %v), want reduce production %d", state, symbol, kind, target, ok, action.ProductionID)
			}
			return true
		}
	}
	return false
}

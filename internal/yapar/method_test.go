package yapar

import (
	"reflect"
	"strings"
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

	ll1Grammar := mustBuildGrammar(t, `%token A B
%%
s : A opt ;
opt : B | ;
`)
	ll1FF, err := ComputeFirstFollow(ll1Grammar)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() ll1 error = %v", err)
	}
	ll1Parser, err := BuildParser(ll1Grammar, ll1FF, MethodLL1)
	if err != nil {
		t.Fatalf("BuildParser() ll1 error = %v", err)
	}
	if ll1Parser == nil || ll1Parser.Table() == nil {
		t.Fatal("BuildParser() ll1 returned nil parser/table")
	}

	parser, err = BuildParser(g, ff, MethodLALR)
	if err != nil {
		t.Fatalf("BuildParser() lalr error = %v", err)
	}
	if parser == nil || parser.Table() == nil {
		t.Fatal("BuildParser() lalr returned nil parser/table")
	}

	_, err = BuildParser(g, ff, Method("lr2"))
	if err == nil {
		t.Fatal("BuildParser() unsupported method error = nil, want error")
	}
	if got := err.Error(); !strings.Contains(got, "not implemented yet") {
		t.Fatalf("BuildParser() unsupported method error = %q, want message containing %q", got, "not implemented yet")
	}
}

func TestValidMethodsReturnsDefensiveCopy(t *testing.T) {
	got := ValidMethods()
	want := []Method{MethodSLR, MethodLL1, MethodLALR}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidMethods() = %v, want %v", got, want)
	}

	got[0] = Method("mutated")
	if fresh := ValidMethods(); !reflect.DeepEqual(fresh, want) {
		t.Fatalf("ValidMethods() defensive copy = %v, want %v", fresh, want)
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

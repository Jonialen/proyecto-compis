package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/yapar"
)

type stubTableView struct {
	actions      map[int]map[string]actionCell
	gotos        map[int]map[string]int
	states       []int
	terminals    []string
	nonTerminals []string
}

type actionCell struct {
	kind  yapar.ActionKind
	value int
	ok    bool
}

func (s stubTableView) ActionAt(state int, symbol string) (yapar.ActionKind, int, bool) {
	if row := s.actions[state]; row != nil {
		if cell, ok := row[symbol]; ok {
			return cell.kind, cell.value, cell.ok
		}
	}
	return yapar.ActionError, 0, false
}

func (s stubTableView) GotoAt(state int, symbol string) (int, bool) {
	if row := s.gotos[state]; row != nil {
		target, ok := row[symbol]
		return target, ok
	}
	return 0, false
}

func (s stubTableView) States() []int          { return append([]int(nil), s.states...) }
func (s stubTableView) Terminals() []string    { return append([]string(nil), s.terminals...) }
func (s stubTableView) NonTerminals() []string { return append([]string(nil), s.nonTerminals...) }

func TestRunHandlesNilWriters(t *testing.T) {
	dir := t.TempDir()
	yalpPath := writeParserSpec(t, dir, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)

	if err := run([]string{"-yalp", yalpPath}, nil, nil); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestParseFlagsValidationBranches(t *testing.T) {
	t.Run("missing yalp", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags(nil, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want missing -yalp")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if !strings.Contains(stderr.String(), "Usage: yapar") {
			t.Fatalf("stderr = %q, want usage output", stderr.String())
		}
		if got, want := err.Error(), "missing required -yalp flag"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("yal and src must be paired", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-yal", "lexer.yal"}, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want pairing error")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if got, want := err.Error(), "-yal and -src must be provided together"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("unexpected positional args are rejected", func(t *testing.T) {
		var stderr bytes.Buffer
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "extra"}, &stderr)
		if err == nil {
			t.Fatal("parseFlags() error = nil, want positional-args error")
		}
		if cfg != nil {
			t.Fatalf("parseFlags() cfg = %#v, want nil", cfg)
		}
		if got, want := err.Error(), "unexpected positional arguments: extra"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})

	t.Run("valid ll1 config keeps parsed method", func(t *testing.T) {
		cfg, err := parseFlags([]string{"-yalp", "parser.yalp", "-method", "ll1", "-table"}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("parseFlags() error = %v", err)
		}
		if cfg.method != yapar.MethodLL1 {
			t.Fatalf("cfg.method = %q, want %q", cfg.method, yapar.MethodLL1)
		}
		if !cfg.printTable {
			t.Fatal("cfg.printTable = false, want true")
		}
	})
}

func TestFormatParsingHelpersDefensiveBranches(t *testing.T) {
	if got, want := formatParsingTable(nil), "<empty table>\n"; got != want {
		t.Fatalf("formatParsingTable(nil) = %q, want %q", got, want)
	}

	table := stubTableView{
		actions: map[int]map[string]actionCell{
			0: {
				"A":   {kind: yapar.ActionShift, value: 2, ok: true},
				"B":   {kind: yapar.ActionReduce, value: 4, ok: true},
				"$":   {kind: yapar.ActionAccept, ok: true},
				"ERR": {kind: yapar.ActionError, ok: true},
			},
		},
		gotos:        map[int]map[string]int{0: {"expr": 7}},
		states:       []int{0},
		terminals:    []string{"A", "B", "$", "ERR"},
		nonTerminals: []string{"expr"},
	}

	if got, want := formatActionRow(table, 0, []string{"A", "B", "$", "ERR"}), "A=s2, B=r4, $=acc"; got != want {
		t.Fatalf("formatActionRow() = %q, want %q", got, want)
	}
	if got, want := formatActionRow(table, 1, []string{"A"}), "-"; got != want {
		t.Fatalf("formatActionRow(empty) = %q, want %q", got, want)
	}
	if got, want := formatGotoRow(table, 0, []string{"expr"}), "expr=7"; got != want {
		t.Fatalf("formatGotoRow() = %q, want %q", got, want)
	}
	if got, want := formatGotoRow(table, 1, []string{"expr"}), "-"; got != want {
		t.Fatalf("formatGotoRow(empty) = %q, want %q", got, want)
	}
	if got, want := lookupAction(table, 0, "ERR"), ""; got != want {
		t.Fatalf("lookupAction(default) = %q, want empty string", got)
	}
	if got, ok := lookupGoto(table, 1, "expr"); ok || got != 0 {
		t.Fatalf("lookupGoto(missing) = (%d, %v), want (0, false)", got, ok)
	}

	formatted := formatParsingTable(table)
	if !strings.Contains(formatted, "State\tACTION\tGOTO") {
		t.Fatalf("formatParsingTable() = %q, want header", formatted)
	}
	if !strings.Contains(formatted, "0\tA=s2, B=r4, $=acc\texpr=7") {
		t.Fatalf("formatParsingTable() = %q, want formatted row", formatted)
	}
}

func TestStubTableViewCopiesSlices(t *testing.T) {
	table := stubTableView{states: []int{1, 2}, terminals: []string{"A"}, nonTerminals: []string{"s"}}

	states := table.States()
	terms := table.Terminals()
	nonTerms := table.NonTerminals()
	states[0] = 99
	terms[0] = "X"
	nonTerms[0] = "expr"

	if got, want := table.States(), []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("States() copy = %v, want %v", got, want)
	}
	if got, want := table.Terminals(), []string{"A"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Terminals() copy = %v, want %v", got, want)
	}
	if got, want := table.NonTerminals(), []string{"s"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NonTerminals() copy = %v, want %v", got, want)
	}
}

func writeParserSpec(t *testing.T, dir, content string) string {
	t.Helper()
	path := dir + "/parser.yalp"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

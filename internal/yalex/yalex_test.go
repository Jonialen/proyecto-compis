package yalex

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireError(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got: %v", substr, err)
	}
}

// ---------------------------------------------------------------------------
// parser.go tests
// ---------------------------------------------------------------------------

func TestParse_BasicMacroAndRule(t *testing.T) {
	input := "let DIGIT = [0-9]\n\nrule tokens =\n| [0-9]+ { INT }"

	res, err := Parse(input)
	requireNoError(t, err)

	// Macro
	if v, ok := res.Macros["DIGIT"]; !ok || v != "[0-9]" {
		t.Fatalf("expected macro DIGIT=[0-9], got %q (ok=%v)", v, ok)
	}

	// Rules
	if len(res.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(res.Rules))
	}
	r := res.Rules[0]
	if r.Pattern != "[0-9]+" {
		t.Errorf("pattern: got %q, want %q", r.Pattern, "[0-9]+")
	}
	if r.Action != " INT " {
		// Action includes surrounding whitespace from { INT }
		if strings.TrimSpace(r.Action) != "INT" {
			t.Errorf("action: got %q, want %q", r.Action, "INT")
		}
	}
	if r.Priority != 0 {
		t.Errorf("priority: got %d, want 0", r.Priority)
	}
}

func TestParse_CommentsRemoved(t *testing.T) {
	input := "(* comment *) let X = a\n\nrule t =\n| 'a' { A }"

	res, err := Parse(input)
	requireNoError(t, err)

	if _, ok := res.Macros["X"]; !ok {
		t.Error("macro X should be present after comment removal")
	}
	if len(res.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(res.Rules))
	}
	if strings.TrimSpace(res.Rules[0].Action) != "A" {
		t.Errorf("action: got %q, want %q", res.Rules[0].Action, "A")
	}
}

func TestParse_MultilineComment(t *testing.T) {
	input := "(* multi\nline\ncomment *) rule t =\n| 'x' { X }"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(res.Rules))
	}
	if strings.TrimSpace(res.Rules[0].Action) != "X" {
		t.Errorf("action: got %q, want %q", res.Rules[0].Action, "X")
	}
}

func TestParse_UnclosedComment(t *testing.T) {
	// Unclosed comment should strip everything from (* to end of input
	input := "(* never closed rule t =\n| 'x' { X }"

	res, err := Parse(input)
	requireNoError(t, err)

	// Everything after (* is swallowed — no rule section remains
	if len(res.Rules) != 0 {
		t.Errorf("expected 0 rules (content consumed by unclosed comment), got %d", len(res.Rules))
	}
}

func TestParse_NoRuleSection(t *testing.T) {
	input := "let A = [a-z]\nlet B = [0-9]"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Macros) != 2 {
		t.Errorf("expected 2 macros, got %d", len(res.Macros))
	}
	if len(res.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(res.Rules))
	}
}

func TestParse_HeaderAndTrailer(t *testing.T) {
	input := "{ header code }\nlet X = a\n\nrule t =\n| 'a' { A }\n{ trailer code }"

	res, err := Parse(input)
	requireNoError(t, err)

	// Header and trailer should be stripped; macro and rule should survive.
	if _, ok := res.Macros["X"]; !ok {
		t.Error("macro X should be present after header/trailer removal")
	}
	if len(res.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(res.Rules))
	}
	if strings.TrimSpace(res.Rules[0].Action) != "A" {
		t.Errorf("action: got %q, want %q", strings.TrimSpace(res.Rules[0].Action), "A")
	}
}

func TestParse_MultiPipeAlternatives(t *testing.T) {
	// Two pipe alternatives sharing a single action block
	input := "rule t =\n| 'T' | 'F' { BOOLEAN }"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d: %+v", len(res.Rules), res.Rules)
	}

	for i, r := range res.Rules {
		action := strings.TrimSpace(r.Action)
		if action != "BOOLEAN" {
			t.Errorf("rule[%d].Action = %q, want %q", i, action, "BOOLEAN")
		}
	}

	// First pattern should be 'T', second 'F'
	p0 := strings.TrimSpace(res.Rules[0].Pattern)
	p1 := strings.TrimSpace(res.Rules[1].Pattern)
	if p0 != "'T'" {
		t.Errorf("rule[0].Pattern = %q, want %q", p0, "'T'")
	}
	if p1 != "'F'" {
		t.Errorf("rule[1].Pattern = %q, want %q", p1, "'F'")
	}
}

func TestParse_EscapedSingleQuote(t *testing.T) {
	// Test that an escaped single quote inside a quoted pattern is handled
	input := "rule t =\n| '\\'' { QUOTE }"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(res.Rules))
	}
	if strings.TrimSpace(res.Rules[0].Action) != "QUOTE" {
		t.Errorf("action: got %q, want %q", strings.TrimSpace(res.Rules[0].Action), "QUOTE")
	}
	// Pattern should contain the escaped quote
	if !strings.Contains(res.Rules[0].Pattern, "\\'") {
		t.Errorf("pattern should contain escaped quote, got %q", res.Rules[0].Pattern)
	}
}

func TestParse_PriorityOrder(t *testing.T) {
	input := "rule t =\n| 'a' { A }\n| 'b' { B }\n| 'c' { C }"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(res.Rules))
	}
	for i, r := range res.Rules {
		if r.Priority != i {
			t.Errorf("rule[%d].Priority = %d, want %d", i, r.Priority, i)
		}
	}

	// Also verify order matches actions
	expected := []string{"A", "B", "C"}
	for i, r := range res.Rules {
		if strings.TrimSpace(r.Action) != expected[i] {
			t.Errorf("rule[%d].Action = %q, want %q", i, strings.TrimSpace(r.Action), expected[i])
		}
	}
}

// ---------------------------------------------------------------------------
// splitByPipe tests (unexported)
// ---------------------------------------------------------------------------

func TestSplitByPipe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of parts
	}{
		{
			name:  "simple two parts",
			input: "'a' | 'b'",
			want:  2,
		},
		{
			name:  "nested parens",
			input: "('a' | 'b') | 'c'",
			want:  2,
		},
		{
			name:  "pipe inside double quotes",
			input: `"a|b" | 'c'`,
			want:  2,
		},
		{
			name:  "pipe inside brackets",
			input: "[a|b] | 'c'",
			want:  2,
		},
		{
			name:  "pipe inside braces",
			input: "{ a | b } | 'c'",
			want:  2,
		},
		{
			name:  "no pipe",
			input: "'a'",
			want:  1,
		},
		{
			name:  "multiple pipes",
			input: "'a' | 'b' | 'c' | 'd'",
			want:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitByPipe(tt.input)
			if len(parts) != tt.want {
				t.Errorf("splitByPipe(%q) returned %d parts, want %d: %v",
					tt.input, len(parts), tt.want, parts)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// expander.go tests
// ---------------------------------------------------------------------------

func TestExpand_SimpleMacro(t *testing.T) {
	macros := map[string]string{
		"DIGIT": "[0-9]",
	}
	rules := []TokenRule{
		{Pattern: "DIGIT+", Action: "INT", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	if len(expanded) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(expanded))
	}
	// DIGIT should be expanded and wrapped in parens: ([0-9])+
	got := expanded[0].Pattern
	if got != "([0-9])+" {
		t.Errorf("expanded pattern = %q, want %q", got, "([0-9])+")
	}
}

func TestExpand_TransitiveMacro(t *testing.T) {
	macros := map[string]string{
		"DIGIT":  "[0-9]",
		"NUMBER": "DIGIT+",
	}
	rules := []TokenRule{
		{Pattern: "NUMBER", Action: "NUM", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	// DIGIT → [0-9], NUMBER → (DIGIT)+ → ([0-9])+
	// Rule NUMBER → (([0-9])+)
	got := expanded[0].Pattern
	if got != "(([0-9])+)" {
		t.Errorf("expanded pattern = %q, want %q", got, "(([0-9])+)")
	}
}

func TestExpand_CyclicMacro(t *testing.T) {
	// A self-referencing macro (A=A) should be detected as a cycle
	// and produce an error.
	macros := map[string]string{
		"A": "A",
	}
	rules := []TokenRule{
		{Pattern: "A", Action: "X", Priority: 0},
	}

	_, err := Expand(macros, rules)
	if err == nil {
		t.Fatal("expected error for cyclic macro, got nil")
	}
	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("expected error containing 'cyclic', got %q", err.Error())
	}
}

func TestExpand_CyclicMacro_MutualReference(t *testing.T) {
	// Mutual references A=B, B=A should be detected as a cycle
	// and produce an error.
	macros := map[string]string{
		"A": "B",
		"B": "A",
	}
	rules := []TokenRule{
		{Pattern: "A", Action: "X", Priority: 0},
	}

	_, err := Expand(macros, rules)
	if err == nil {
		t.Fatal("expected error for cyclic macro, got nil")
	}
	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("expected error containing 'cyclic', got %q", err.Error())
	}
}

func TestExpand_UndefinedMacro(t *testing.T) {
	// The expander only calls the resolver for identifiers that exist in
	// the expanded map. An identifier not in the expanded map is passed
	// through as a literal (see expandPatternWithResolver line 191).
	// So expanding a macro whose body references an unknown name does NOT
	// produce an error — the unknown name stays as-is.
	//
	// However, a rule that directly references a macro name which does exist
	// in the macros map WILL call the resolver. To trigger "undefined" we
	// need the resolver itself to fail, which happens inside expandMacros
	// when expandOne is called for a name not in the original macros map.
	// But expandOne is only invoked for identifiers found in the expanded map.
	//
	// Test what actually happens: UNDEF is treated as a literal.
	macros := map[string]string{
		"A": "UNDEF",
	}
	rules := []TokenRule{
		{Pattern: "A", Action: "X", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	// A's body "UNDEF" is not a known macro, so it stays as-is.
	// Then the rule "A" → "(UNDEF)"
	got := expanded[0].Pattern
	if got != "(UNDEF)" {
		t.Errorf("expanded pattern = %q, want %q", got, "(UNDEF)")
	}
}

func TestExpand_NoMacros(t *testing.T) {
	macros := map[string]string{}
	rules := []TokenRule{
		{Pattern: "'a'|'b'", Action: "AB", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	got := expanded[0].Pattern
	if got != "'a'|'b'" {
		t.Errorf("expanded pattern = %q, want %q", got, "'a'|'b'")
	}
}

func TestExpand_MacroInCharClass(t *testing.T) {
	// Macro names inside [...] should NOT be expanded
	macros := map[string]string{
		"X": "expanded",
	}
	rules := []TokenRule{
		{Pattern: "[X]", Action: "T", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	got := expanded[0].Pattern
	if got != "[X]" {
		t.Errorf("pattern inside brackets should not expand: got %q, want %q", got, "[X]")
	}
}

func TestExpand_MacroInSingleQuotes(t *testing.T) {
	// Macro names inside '...' should NOT be expanded
	macros := map[string]string{
		"X": "expanded",
	}
	rules := []TokenRule{
		{Pattern: "'X'", Action: "T", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	got := expanded[0].Pattern
	if got != "'X'" {
		t.Errorf("pattern inside quotes should not expand: got %q, want %q", got, "'X'")
	}
}

func TestExpand_MacroInDoubleQuotes(t *testing.T) {
	// Macro names inside "..." should NOT be expanded
	macros := map[string]string{
		"X": "expanded",
	}
	rules := []TokenRule{
		{Pattern: `"X"`, Action: "T", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	got := expanded[0].Pattern
	if got != `"X"` {
		t.Errorf("pattern inside double quotes should not expand: got %q, want %q", got, `"X"`)
	}
}

func TestExpand_UnderscorePassthrough(t *testing.T) {
	// '_' is listed in the operator passthrough set (|, *, +, ?, ., _).
	// However, '_' is also returned true by isIdentStart, and the operator
	// check only fires for standalone characters. When '_' is contiguous
	// with an alpha identifier (e.g. "A_"), isIdentContinue('_') is true,
	// so "A_" is consumed as a single identifier.
	//
	// Test the standalone case: a bare '_' in a pattern is treated as an
	// operator and passed through without triggering identifier expansion.
	macros := map[string]string{
		"A": "[a-z]",
	}
	rules := []TokenRule{
		{Pattern: "_", Action: "T", Priority: 0},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	// Standalone '_' hits the operator branch, NOT the identifier branch.
	got := expanded[0].Pattern
	if got != "_" {
		t.Errorf("expanded pattern = %q, want %q — '_' should pass through as operator", got, "_")
	}
}

// ---------------------------------------------------------------------------
// Additional edge-case tests
// ---------------------------------------------------------------------------

func TestExpand_PreservesActionAndPriority(t *testing.T) {
	macros := map[string]string{
		"DIGIT": "[0-9]",
	}
	rules := []TokenRule{
		{Pattern: "DIGIT", Action: "NUM", Priority: 5},
	}

	expanded, err := Expand(macros, rules)
	requireNoError(t, err)

	if expanded[0].Action != "NUM" {
		t.Errorf("action should be preserved: got %q, want %q", expanded[0].Action, "NUM")
	}
	if expanded[0].Priority != 5 {
		t.Errorf("priority should be preserved: got %d, want 5", expanded[0].Priority)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	res, err := Parse("")
	requireNoError(t, err)

	if len(res.Macros) != 0 {
		t.Errorf("expected 0 macros, got %d", len(res.Macros))
	}
	if len(res.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(res.Rules))
	}
}

func TestParse_OnlyComments(t *testing.T) {
	input := "(* this is all comments *)"
	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Macros) != 0 {
		t.Errorf("expected 0 macros, got %d", len(res.Macros))
	}
	if len(res.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(res.Rules))
	}
}

func TestParse_MultipleMacros(t *testing.T) {
	input := "let DIGIT = [0-9]\nlet LETTER = [a-zA-Z]\nlet IDENT = LETTER(LETTER|DIGIT)*\n\nrule tokens =\n| IDENT { ID }"

	res, err := Parse(input)
	requireNoError(t, err)

	if len(res.Macros) != 3 {
		t.Errorf("expected 3 macros, got %d", len(res.Macros))
	}

	expected := map[string]string{
		"DIGIT":  "[0-9]",
		"LETTER": "[a-zA-Z]",
		"IDENT":  "LETTER(LETTER|DIGIT)*",
	}
	for name, want := range expected {
		got, ok := res.Macros[name]
		if !ok {
			t.Errorf("macro %q not found", name)
			continue
		}
		if got != want {
			t.Errorf("macro %q = %q, want %q", name, got, want)
		}
	}
}

func TestRemoveComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no comments",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "single comment",
			input: "before (* comment *) after",
			want:  "before  after",
		},
		{
			name:  "multiline comment",
			input: "before (* multi\nline *) after",
			want:  "before  after",
		},
		{
			name:  "unclosed comment",
			input: "before (* never closed",
			want:  "before ",
		},
		{
			name:  "multiple comments",
			input: "(* c1 *) mid (* c2 *) end",
			want:  " mid  end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeComments(tt.input)
			if got != tt.want {
				t.Errorf("removeComments(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractPatternAction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPat    string
		wantAction string
		wantErr    bool
	}{
		{
			name:       "simple",
			input:      "'a' { A }",
			wantPat:    "'a' ",
			wantAction: " A ",
			wantErr:    false,
		},
		{
			name:    "no action",
			input:   "'a'",
			wantErr: true,
		},
		{
			name:       "brace inside quotes",
			input:      "'{' { LBRACE }",
			wantPat:    "'{' ",
			wantAction: " LBRACE ",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, action, err := extractPatternAction(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			requireNoError(t, err)
			if pat != tt.wantPat {
				t.Errorf("pattern = %q, want %q", pat, tt.wantPat)
			}
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: Parse + Expand
// ---------------------------------------------------------------------------

func TestParseAndExpand_Integration(t *testing.T) {
	input := `let DIGIT = [0-9]
let NUMBER = DIGIT+

rule tokens =
| NUMBER { NUM }
| 'a' { A }`

	res, err := Parse(input)
	requireNoError(t, err)

	expanded, err := Expand(res.Macros, res.Rules)
	requireNoError(t, err)

	if len(expanded) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(expanded))
	}

	// Rule 0: NUMBER → (([0-9])+)
	got0 := expanded[0].Pattern
	if got0 != "(([0-9])+)" {
		t.Errorf("rule[0].Pattern = %q, want %q", got0, "(([0-9])+)")
	}

	// Rule 1: 'a' should be unchanged
	got1 := expanded[1].Pattern
	if got1 != "'a'" {
		t.Errorf("rule[1].Pattern = %q, want %q", got1, "'a'")
	}
}

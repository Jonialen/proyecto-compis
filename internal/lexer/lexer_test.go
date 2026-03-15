package lexer

import (
	"os"
	"strings"
	"testing"

	"genanalex/internal/dfa"
	"genanalex/internal/regex"
)

// buildDFA constructs a minimized DFA from a .yal-style pattern string.
func buildDFA(t *testing.T, pattern string, tokenName string) *dfa.DFA {
	t.Helper()

	normalized, err := regex.Normalize(pattern)
	if err != nil {
		t.Fatalf("regex.Normalize(%q): %v", pattern, err)
	}

	postfix, err := regex.BuildPostfix(normalized)
	if err != nil {
		t.Fatalf("regex.BuildPostfix(%q): %v", pattern, err)
	}

	root, posToSymbol, err := dfa.BuildTree(postfix)
	if err != nil {
		t.Fatalf("dfa.BuildTree(%q): %v", pattern, err)
	}

	d := dfa.BuildDFA(root, posToSymbol, tokenName)
	return dfa.Minimize(d)
}

// writeTempFile creates a temporary file with the given content and returns its path.
// The caller must call os.Remove on the returned path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "lexer_test_*.txt")
	if err != nil {
		t.Fatalf("os.CreateTemp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("Close: %v", err)
	}
	return f.Name()
}

// ─── reader.go tests ────────────────────────────────────────────────────────

func TestReadSource_NormalFile(t *testing.T) {
	path := writeTempFile(t, "hello\nworld\n")
	defer os.Remove(path)

	sf, err := ReadSource(path)
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}

	if sf.Content != "hello\nworld\n" {
		t.Errorf("Content = %q, want %q", sf.Content, "hello\nworld\n")
	}
	// "hello\nworld\n" split by \n → ["hello", "world", ""]
	wantLines := []string{"hello", "world", ""}
	if len(sf.Lines) != len(wantLines) {
		t.Fatalf("Lines count = %d, want %d; Lines = %v", len(sf.Lines), len(wantLines), sf.Lines)
	}
	for i, line := range wantLines {
		if sf.Lines[i] != line {
			t.Errorf("Lines[%d] = %q, want %q", i, sf.Lines[i], line)
		}
	}
}

func TestReadSource_CRLFNormalization(t *testing.T) {
	path := writeTempFile(t, "line1\r\nline2\r\n")
	defer os.Remove(path)

	sf, err := ReadSource(path)
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}

	if strings.Contains(sf.Content, "\r") {
		t.Errorf("Content still contains \\r after normalization: %q", sf.Content)
	}
	if sf.Content != "line1\nline2\n" {
		t.Errorf("Content = %q, want %q", sf.Content, "line1\nline2\n")
	}
}

func TestReadSource_LoneCR(t *testing.T) {
	path := writeTempFile(t, "a\rb\rc")
	defer os.Remove(path)

	sf, err := ReadSource(path)
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}

	if strings.Contains(sf.Content, "\r") {
		t.Errorf("Content still contains \\r: %q", sf.Content)
	}
	if sf.Content != "a\nb\nc" {
		t.Errorf("Content = %q, want %q", sf.Content, "a\nb\nc")
	}
	wantLines := []string{"a", "b", "c"}
	if len(sf.Lines) != len(wantLines) {
		t.Fatalf("Lines count = %d, want %d", len(sf.Lines), len(wantLines))
	}
	for i, line := range wantLines {
		if sf.Lines[i] != line {
			t.Errorf("Lines[%d] = %q, want %q", i, sf.Lines[i], line)
		}
	}
}

func TestReadSource_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")
	defer os.Remove(path)

	sf, err := ReadSource(path)
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}

	if sf.Content != "" {
		t.Errorf("Content = %q, want empty", sf.Content)
	}
	// strings.Split("", "\n") returns [""]
	if len(sf.Lines) != 1 || sf.Lines[0] != "" {
		t.Errorf("Lines = %v, want [\"\"]", sf.Lines)
	}
}

func TestReadSource_NonexistentFile(t *testing.T) {
	_, err := ReadSource("/tmp/this_file_does_not_exist_12345.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// ─── simulator.go tests (Tokenize) ─────────────────────────────────────────

// Standard DFA entries used across multiple tests.
func intEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "[0-9]+", "INT"), TokenName: "INT", Priority: priority}
}

func idEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "[a-zA-Z][a-zA-Z0-9_]*", "ID"), TokenName: "ID", Priority: priority}
}

func kwLetEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, `"let"`, "KEYWORD_let"), TokenName: "KEYWORD_let", Priority: priority}
}

func wsEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "[' ' '\\t' '\\n']+", "skip"), TokenName: "skip", Priority: priority}
}

func plusOpEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "'+'", "PLUS"), TokenName: "PLUS", Priority: priority}
}

func lparenEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "'('", "LPAREN"), TokenName: "LPAREN", Priority: priority}
}

func rparenEntry(t *testing.T, priority int) DFAEntry {
	t.Helper()
	return DFAEntry{DFA: buildDFA(t, "')'", "RPAREN"), TokenName: "RPAREN", Priority: priority}
}

func makeSourceFile(content string) *SourceFile {
	return &SourceFile{
		Path:    "<test>",
		Content: content,
		Lines:   strings.Split(content, "\n"),
	}
}

// Test 6: Single token
func TestTokenize_SingleToken(t *testing.T) {
	dfas := []DFAEntry{intEntry(t, 0)}
	tokens, errs := Tokenize(dfas, makeSourceFile("42"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1; tokens = %v", len(tokens), tokens)
	}
	tok := tokens[0]
	if tok.Type != "INT" || tok.Lexeme != "42" || tok.Line != 1 {
		t.Errorf("got Token{%q, %q, %d}, want {INT, 42, 1}", tok.Type, tok.Lexeme, tok.Line)
	}
}

// Test 7: Multiple tokens
func TestTokenize_MultipleTokens(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		idEntry(t, 1),
		wsEntry(t, 2),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("42 abc"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "INT" || tokens[0].Lexeme != "42" {
		t.Errorf("tokens[0] = {%q, %q}, want {INT, 42}", tokens[0].Type, tokens[0].Lexeme)
	}
	if tokens[1].Type != "ID" || tokens[1].Lexeme != "abc" {
		t.Errorf("tokens[1] = {%q, %q}, want {ID, abc}", tokens[1].Type, tokens[1].Lexeme)
	}
}

// Test 8: Skip whitespace
func TestTokenize_SkipWhitespace(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		wsEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("  42  "))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "INT" || tokens[0].Lexeme != "42" {
		t.Errorf("got {%q, %q}, want {INT, 42}", tokens[0].Type, tokens[0].Lexeme)
	}
}

// Test 9: Maximal munch — INT is greedy, then ID picks up the rest
func TestTokenize_MaximalMunch(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		idEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("123abc"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "INT" || tokens[0].Lexeme != "123" {
		t.Errorf("tokens[0] = {%q, %q}, want {INT, 123}", tokens[0].Type, tokens[0].Lexeme)
	}
	if tokens[1].Type != "ID" || tokens[1].Lexeme != "abc" {
		t.Errorf("tokens[1] = {%q, %q}, want {ID, abc}", tokens[1].Type, tokens[1].Lexeme)
	}
}

// Test 10: Priority disambiguation — keyword beats identifier when same length
func TestTokenize_PriorityDisambiguation(t *testing.T) {
	dfas := []DFAEntry{
		kwLetEntry(t, 0),
		idEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("let"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "KEYWORD_let" || tokens[0].Lexeme != "let" {
		t.Errorf("got {%q, %q}, want {KEYWORD_let, let}", tokens[0].Type, tokens[0].Lexeme)
	}
}

// Test 11: Longer match wins over higher priority
func TestTokenize_LongerMatchWins(t *testing.T) {
	dfas := []DFAEntry{
		kwLetEntry(t, 0),
		idEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("letter"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "ID" || tokens[0].Lexeme != "letter" {
		t.Errorf("got {%q, %q}, want {ID, letter}", tokens[0].Type, tokens[0].Lexeme)
	}
}

// Test 12: Unrecognized character
func TestTokenize_UnrecognizedChar(t *testing.T) {
	dfas := []DFAEntry{intEntry(t, 0)}
	tokens, errs := Tokenize(dfas, makeSourceFile("@"))

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0; tokens = %+v", len(tokens), tokens)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1; errs = %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "@") {
		t.Errorf("error %q does not mention '@'", errs[0])
	}
	if !strings.Contains(errs[0], "unrecognized") {
		t.Errorf("error %q does not say 'unrecognized'", errs[0])
	}
}

// Test 13: Error recovery — tokenizer continues past bad characters
func TestTokenize_ErrorRecovery(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		idEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("42@abc"))

	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1; errs = %v", len(errs), errs)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "INT" || tokens[0].Lexeme != "42" {
		t.Errorf("tokens[0] = {%q, %q}, want {INT, 42}", tokens[0].Type, tokens[0].Lexeme)
	}
	if tokens[1].Type != "ID" || tokens[1].Lexeme != "abc" {
		t.Errorf("tokens[1] = {%q, %q}, want {ID, abc}", tokens[1].Type, tokens[1].Lexeme)
	}
}

// Test 14: Line tracking across newlines
func TestTokenize_LineTracking(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		idEntry(t, 1),
		wsEntry(t, 2),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("42\nabc"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Line != 1 {
		t.Errorf("tokens[0].Line = %d, want 1", tokens[0].Line)
	}
	if tokens[1].Line != 2 {
		t.Errorf("tokens[1].Line = %d, want 2", tokens[1].Line)
	}
}

// Test 15: Empty source
func TestTokenize_EmptySource(t *testing.T) {
	dfas := []DFAEntry{intEntry(t, 0)}
	tokens, errs := Tokenize(dfas, makeSourceFile(""))

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0", len(errs))
	}
}

// Test 16: Full expression
func TestTokenize_Expression(t *testing.T) {
	dfas := []DFAEntry{
		lparenEntry(t, 0),
		rparenEntry(t, 1),
		intEntry(t, 2),
		plusOpEntry(t, 3),
		wsEntry(t, 4),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("(1 + 2)"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	expected := []struct {
		typ    string
		lexeme string
	}{
		{"LPAREN", "("},
		{"INT", "1"},
		{"PLUS", "+"},
		{"INT", "2"},
		{"RPAREN", ")"},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d; tokens = %+v", len(tokens), len(expected), tokens)
	}
	for i, want := range expected {
		if tokens[i].Type != want.typ || tokens[i].Lexeme != want.lexeme {
			t.Errorf("tokens[%d] = {%q, %q}, want {%q, %q}",
				i, tokens[i].Type, tokens[i].Lexeme, want.typ, want.lexeme)
		}
	}
}

// ─── Edge case tests ────────────────────────────────────────────────────────

// Test 17: Single char tokens adjacent
func TestTokenize_SingleCharTokens(t *testing.T) {
	dfas := []DFAEntry{
		lparenEntry(t, 0),
		rparenEntry(t, 1),
	}
	tokens, errs := Tokenize(dfas, makeSourceFile("()"))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2; tokens = %+v", len(tokens), tokens)
	}
	if tokens[0].Type != "LPAREN" || tokens[0].Lexeme != "(" {
		t.Errorf("tokens[0] = {%q, %q}, want {LPAREN, (}", tokens[0].Type, tokens[0].Lexeme)
	}
	if tokens[1].Type != "RPAREN" || tokens[1].Lexeme != ")" {
		t.Errorf("tokens[1] = {%q, %q}, want {RPAREN, )}", tokens[1].Type, tokens[1].Lexeme)
	}
}

// Test 18: All whitespace
func TestTokenize_AllWhitespace(t *testing.T) {
	dfas := []DFAEntry{wsEntry(t, 0)}
	tokens, errs := Tokenize(dfas, makeSourceFile("   \t\n  "))

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0 (all skipped); tokens = %+v", len(tokens), tokens)
	}
}

// Test 19: Consecutive errors
func TestTokenize_ConsecutiveErrors(t *testing.T) {
	dfas := []DFAEntry{intEntry(t, 0)}
	tokens, errs := Tokenize(dfas, makeSourceFile("@#$"))

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
	if len(errs) != 3 {
		t.Fatalf("got %d errors, want 3; errs = %v", len(errs), errs)
	}
	// Each error should mention the specific character
	for i, ch := range []string{"@", "#", "$"} {
		if !strings.Contains(errs[i], ch) {
			t.Errorf("errs[%d] = %q, does not mention %q", i, errs[i], ch)
		}
	}
}

// Test 20: Multi-line errors with correct line numbers
func TestTokenize_MultiLineError(t *testing.T) {
	dfas := []DFAEntry{
		intEntry(t, 0),
		wsEntry(t, 1),
	}
	// @ on line 1, # on line 2
	tokens, errs := Tokenize(dfas, makeSourceFile("@\n#"))

	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2; errs = %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "line 1") {
		t.Errorf("errs[0] = %q, should mention 'line 1'", errs[0])
	}
	if !strings.Contains(errs[1], "line 2") {
		t.Errorf("errs[1] = %q, should mention 'line 2'", errs[1])
	}
}

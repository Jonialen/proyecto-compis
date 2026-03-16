package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/dfa"
	"genanalex/internal/lexer"
	"genanalex/internal/regex"
	"genanalex/internal/yalex"
)

// buildDFAEntry builds a complete DFAEntry from a raw regex pattern and token name.
// Goes through: Normalize -> BuildPostfix -> BuildTree -> BuildDFA -> Minimize.
func buildDFAEntry(t *testing.T, pattern, tokenName string, priority int) lexer.DFAEntry {
	t.Helper()

	normalized, err := regex.Normalize(pattern)
	if err != nil {
		t.Fatalf("normalize %q: %v", pattern, err)
	}

	postfix, err := regex.BuildPostfix(normalized)
	if err != nil {
		t.Fatalf("postfix %q: %v", pattern, err)
	}

	root, posToSymbol, err := dfa.BuildTree(postfix)
	if err != nil {
		t.Fatalf("tree %q: %v", pattern, err)
	}

	builtDFA := dfa.BuildDFA(root, posToSymbol, tokenName)
	minimized := dfa.Minimize(builtDFA)

	return lexer.DFAEntry{
		DFA:       minimized,
		TokenName: tokenName,
		Priority:  priority,
	}
}

// requireGoCompiler skips the test if the Go compiler is not available.
func requireGoCompiler(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("requires go compiler")
	}
}

// setupCompilableDir creates a temp directory with a go.mod so generated code can compile.
// Returns the temp dir path (caller should NOT RemoveAll; t.TempDir handles cleanup).
func setupCompilableDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	goMod := "module testlexer\n\ngo 1.26.1\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	return tmpDir
}

// generateAndCompile generates the source file and compiles it, returning the binary path.
func generateAndCompile(t *testing.T, entries []lexer.DFAEntry) (binaryPath, tmpDir string) {
	t.Helper()
	requireGoCompiler(t)

	tmpDir = setupCompilableDir(t)
	outputFile := filepath.Join(tmpDir, "main.go")

	if err := GenerateSource(outputFile, entries); err != nil {
		t.Fatalf("GenerateSource: %v", err)
	}

	binaryPath = filepath.Join(tmpDir, "lexer_bin")
	cmd := exec.Command("go", "build", "-o", binaryPath, outputFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, string(out))
	}

	return binaryPath, tmpDir
}

// ---------- Tests ----------

func TestGenerateSource_CreatesFile(t *testing.T) {
	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "[0-9]+", "INT", 0),
		buildDFAEntry(t, "[a-zA-Z]+", "ID", 1),
	}

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "generated_lexer.go")

	if err := GenerateSource(outputFile, entries); err != nil {
		t.Fatalf("GenerateSource: %v", err)
	}

	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateSource_ValidGoCode(t *testing.T) {
	requireGoCompiler(t)

	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "[0-9]+", "INT", 0),
		buildDFAEntry(t, "[a-zA-Z]+", "ID", 1),
	}

	tmpDir := setupCompilableDir(t)
	outputFile := filepath.Join(tmpDir, "main.go")

	if err := GenerateSource(outputFile, entries); err != nil {
		t.Fatalf("GenerateSource: %v", err)
	}

	binaryPath := filepath.Join(tmpDir, "lexer_bin")
	cmd := exec.Command("go", "build", "-o", binaryPath, outputFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compilation failed: %v\n%s", err, string(out))
	}

	// Verify the binary was created
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not created: %v", err)
	}
}

func TestGenerateSource_ProducesCorrectOutput(t *testing.T) {
	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "[0-9]+", "INT", 0),
		buildDFAEntry(t, "[a-zA-Z]+", "ID", 1),
		buildDFAEntry(t, "[' ' '\\t']+", "skip", 2),
	}

	binaryPath, tmpDir := generateAndCompile(t, entries)

	// Write a source file to tokenize
	srcPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(srcPath, []byte("42 abc 123"), 0644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	cmd := exec.Command(binaryPath, "-src", srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated lexer: %v\n%s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "INT") {
		t.Errorf("output should contain INT tokens:\n%s", output)
	}
	if !strings.Contains(output, "ID") {
		t.Errorf("output should contain ID tokens:\n%s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("output should contain lexeme '42':\n%s", output)
	}
	if !strings.Contains(output, "abc") {
		t.Errorf("output should contain lexeme 'abc':\n%s", output)
	}
	if !strings.Contains(output, "123") {
		t.Errorf("output should contain lexeme '123':\n%s", output)
	}
}

func TestGenerateSource_WithRealYal(t *testing.T) {
	requireGoCompiler(t)

	// Use the project's real testdata files
	projectRoot := findProjectRoot(t)
	yalPath := filepath.Join(projectRoot, "testdata", "lexer.yal")
	lispPath := filepath.Join(projectRoot, "testdata", "test.lisp")

	// Parse the real .yal file
	parseResult, err := yalex.ParseFile(yalPath)
	if err != nil {
		t.Fatalf("parsing yal: %v", err)
	}

	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		t.Fatalf("expanding macros: %v", err)
	}

	// Build all DFAs
	var entries []lexer.DFAEntry
	for _, rule := range expandedRules {
		entries = append(entries, buildDFAEntry(t, rule.Pattern, rule.Action, rule.Priority))
	}

	binaryPath, tmpDir := generateAndCompile(t, entries)

	// Copy the lisp file to the temp dir
	lispData, err := os.ReadFile(lispPath)
	if err != nil {
		t.Fatalf("reading test.lisp: %v", err)
	}
	srcPath := filepath.Join(tmpDir, "test.lisp")
	if err := os.WriteFile(srcPath, lispData, 0644); err != nil {
		t.Fatalf("writing test.lisp copy: %v", err)
	}

	cmd := exec.Command(binaryPath, "-src", srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated lexer: %v\n%s", err, string(out))
	}

	output := string(out)

	// Verify expected token types are present
	expectedTypes := []string{"KEYWORD", "INT", "FLOAT", "STRING", "BOOLEAN", "DELIMITER", "OPERATOR", "ID", "COMMENT", "NIL"}
	for _, typ := range expectedTypes {
		if !strings.Contains(output, typ) {
			t.Errorf("expected token type %q in output, but not found:\n%s", typ, output)
		}
	}

	// Verify no errors section
	if strings.Contains(output, "--- Errors ---") {
		t.Errorf("generated lexer produced errors:\n%s", output)
	}
}

func TestGenerateSource_EmptyEntries(t *testing.T) {
	requireGoCompiler(t)

	var entries []lexer.DFAEntry

	tmpDir := setupCompilableDir(t)
	outputFile := filepath.Join(tmpDir, "main.go")

	if err := GenerateSource(outputFile, entries); err != nil {
		t.Fatalf("GenerateSource with empty entries: %v", err)
	}

	// Should still produce valid (compilable) Go code
	binaryPath := filepath.Join(tmpDir, "lexer_bin")
	cmd := exec.Command("go", "build", "-o", binaryPath, outputFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compilation of empty-entries lexer failed: %v\n%s", err, string(out))
	}
}

func TestGenerateSource_SpecialCharsInTransitions(t *testing.T) {
	// Build DFAs with patterns that contain special characters (newline, tab)
	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "'\\n'", "NEWLINE", 0),
		buildDFAEntry(t, "'\\t'+", "TABS", 1),
		buildDFAEntry(t, "[a-z]+", "ID", 2),
	}

	binaryPath, tmpDir := generateAndCompile(t, entries)

	// Write input with tabs and newlines
	srcPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(srcPath, []byte("abc\t\tdef\n"), 0644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	cmd := exec.Command(binaryPath, "-src", srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated lexer: %v\n%s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "ID") {
		t.Errorf("expected ID tokens in output:\n%s", output)
	}
	if !strings.Contains(output, "TABS") {
		t.Errorf("expected TABS token in output:\n%s", output)
	}
	if !strings.Contains(output, "NEWLINE") {
		t.Errorf("expected NEWLINE token in output:\n%s", output)
	}
}

func TestGenerateSource_OutputPathCreation(t *testing.T) {
	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "[a-z]+", "ID", 0),
	}

	tmpDir := t.TempDir()
	// Attempt to write to a non-existent subdirectory
	outputFile := filepath.Join(tmpDir, "nonexistent", "subdir", "lexer.go")

	err := GenerateSource(outputFile, entries)
	// The function uses os.WriteFile which does NOT create directories.
	// We expect an error because the directory doesn't exist.
	if err == nil {
		t.Error("expected error when writing to non-existent directory, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "writing generated file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateSource_SkipTokensNotEmitted(t *testing.T) {
	entries := []lexer.DFAEntry{
		buildDFAEntry(t, "[' ' '\\t']+", "skip", 0),
		buildDFAEntry(t, "[a-zA-Z]+", "ID", 1),
	}

	binaryPath, tmpDir := generateAndCompile(t, entries)

	srcPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(srcPath, []byte("  abc  "), 0644); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	cmd := exec.Command(binaryPath, "-src", srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated lexer: %v\n%s", err, string(out))
	}

	output := string(out)

	// Should contain the ID token
	if !strings.Contains(output, "ID") {
		t.Errorf("expected ID token in output:\n%s", output)
	}
	if !strings.Contains(output, "abc") {
		t.Errorf("expected 'abc' lexeme in output:\n%s", output)
	}

	// Count the token lines in the "--- Tokens ---" section.
	// "skip" tokens should not appear in the output.
	lines := strings.Split(output, "\n")
	tokenCount := 0
	for _, line := range lines {
		// Token lines have format: [1] ID           abc
		if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.Contains(line, "ID") {
			tokenCount++
		}
	}
	if tokenCount != 1 {
		t.Errorf("expected exactly 1 ID token, found %d token lines:\n%s", tokenCount, output)
	}

	// "skip" should NOT appear as a token type
	if strings.Contains(output, "skip") {
		t.Errorf("'skip' tokens should not be emitted in output:\n%s", output)
	}
}

// findProjectRoot walks up from the current test's directory to find the go.mod file.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

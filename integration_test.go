package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/dfa"
	"genanalex/internal/generator"
	"genanalex/internal/lexer"
	"genanalex/internal/regex"
	"genanalex/internal/yalex"
)

// buildPipeline runs the full lexer-generator pipeline:
//
//	parse .yal → expand macros → for each rule: normalize → postfix → tree → buildDFA → minimize → tokenize
//
// Returns the tokens and any lexical errors.
func buildPipeline(t *testing.T, yalContent, sourceContent string) ([]lexer.Token, []string) {
	t.Helper()

	// 1. Write yalContent to a temp file and parse it
	tmpDir := t.TempDir()
	yalPath := filepath.Join(tmpDir, "test.yal")
	if err := os.WriteFile(yalPath, []byte(yalContent), 0644); err != nil {
		t.Fatalf("writing temp yal file: %v", err)
	}

	parseResult, err := yalex.ParseFile(yalPath)
	if err != nil {
		t.Fatalf("parsing yal file: %v", err)
	}

	// 2. Expand macros
	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		t.Fatalf("expanding macros: %v", err)
	}

	// 3. For each expanded rule, build the DFA pipeline
	var dfaEntries []lexer.DFAEntry
	for i, rule := range expandedRules {
		normalized, err := regex.Normalize(rule.Pattern)
		if err != nil {
			t.Fatalf("normalizing rule %d (%q): %v", i, rule.Pattern, err)
		}

		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			t.Fatalf("building postfix for rule %d (%q): %v", i, rule.Pattern, err)
		}

		root, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			t.Fatalf("building syntax tree for rule %d (%q): %v", i, rule.Pattern, err)
		}

		builtDFA := dfa.BuildDFA(root, posToSymbol, rule.Action)
		minimizedDFA := dfa.Minimize(builtDFA)

		dfaEntries = append(dfaEntries, lexer.DFAEntry{
			DFA:       minimizedDFA,
			TokenName: rule.Action,
			Priority:  rule.Priority,
		})
	}

	// 4. Create SourceFile from sourceContent (write to temp file)
	srcPath := filepath.Join(tmpDir, "test.src")
	if err := os.WriteFile(srcPath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("writing temp source file: %v", err)
	}

	src, err := lexer.ReadSource(srcPath)
	if err != nil {
		t.Fatalf("reading source file: %v", err)
	}

	// 5. Tokenize
	return lexer.Tokenize(dfaEntries, src)
}

// ---------- Test helpers ----------

type expectedToken struct {
	Type   string
	Lexeme string
	Line   int // 0 means don't check
}

func assertTokens(t *testing.T, got []lexer.Token, want []expectedToken) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("token count: got %d, want %d", len(got), len(want))
		for i, tok := range got {
			t.Logf("  got[%d]: %s %q (line %d)", i, tok.Type, tok.Lexeme, tok.Line)
		}
		return
	}
	for i := range want {
		if got[i].Type != want[i].Type {
			t.Errorf("token[%d].Type: got %q, want %q", i, got[i].Type, want[i].Type)
		}
		if want[i].Lexeme != "" && got[i].Lexeme != want[i].Lexeme {
			t.Errorf("token[%d].Lexeme: got %q, want %q", i, got[i].Lexeme, want[i].Lexeme)
		}
		if want[i].Line != 0 && got[i].Line != want[i].Line {
			t.Errorf("token[%d].Line: got %d, want %d", i, got[i].Line, want[i].Line)
		}
	}
}

// ---------- Integration tests ----------

func TestIntegration_RealYalFile(t *testing.T) {
	yalData, err := os.ReadFile("testdata/lexer.yal")
	if err != nil {
		t.Fatalf("reading testdata/lexer.yal: %v", err)
	}
	srcData, err := os.ReadFile("testdata/test.lisp")
	if err != nil {
		t.Fatalf("reading testdata/test.lisp: %v", err)
	}

	tokens, errors := buildPipeline(t, string(yalData), string(srcData))

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Verify total token count (73 from actual run output)
	if len(tokens) != 73 {
		t.Errorf("token count: got %d, want 73", len(tokens))
		for i, tok := range tokens {
			t.Logf("  [%d] %s %q (line %d)", i, tok.Type, tok.Lexeme, tok.Line)
		}
	}

	// Verify first 5 tokens
	if len(tokens) >= 5 {
		first5 := []expectedToken{
			{Type: "DELIMITER", Lexeme: "(", Line: 1},
			{Type: "KEYWORD", Lexeme: "let", Line: 1},
			{Type: "DELIMITER", Lexeme: "(", Line: 1},
			{Type: "DELIMITER", Lexeme: "(", Line: 1},
			{Type: "ID", Lexeme: "x", Line: 1},
		}
		assertTokens(t, tokens[:5], first5)
	}

	// Verify last 5 tokens
	if len(tokens) >= 5 {
		n := len(tokens)
		last5 := []expectedToken{
			{Type: "BOOLEAN", Lexeme: "F", Line: 8},
			{Type: "STRING", Lexeme: `"negativo"`, Line: 8},
			{Type: "DELIMITER", Lexeme: ")", Line: 8},
			{Type: "DELIMITER", Lexeme: ")", Line: 8},
		}
		assertTokens(t, tokens[n-4:], last5)
	}

	// Verify all expected token types appear at least once
	expectedTypes := []string{
		"DELIMITER", "KEYWORD", "ID", "INT", "FLOAT",
		"STRING", "BOOLEAN", "NIL", "COMMENT", "OPERATOR",
	}
	typeSet := make(map[string]bool)
	for _, tok := range tokens {
		typeSet[tok.Type] = true
	}
	for _, typ := range expectedTypes {
		if !typeSet[typ] {
			t.Errorf("token type %q never appeared in output", typ)
		}
	}
}

func TestIntegration_SimpleCalculator(t *testing.T) {
	yal := `
let DIGIT = [0-9]

rule tokens =
  | [' ' '\t' '\n']+ { skip }
  | DIGIT+ { INT }
  | '+' { PLUS }
  | '-' { MINUS }
  | '*' { TIMES }
  | '/' { DIV }
  | '(' { LPAREN }
  | ')' { RPAREN }
`
	tokens, errors := buildPipeline(t, yal, "(1 + 2) * 3")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	want := []expectedToken{
		{Type: "LPAREN", Lexeme: "("},
		{Type: "INT", Lexeme: "1"},
		{Type: "PLUS", Lexeme: "+"},
		{Type: "INT", Lexeme: "2"},
		{Type: "RPAREN", Lexeme: ")"},
		{Type: "TIMES", Lexeme: "*"},
		{Type: "INT", Lexeme: "3"},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_KeywordVsIdentifier(t *testing.T) {
	yal := `
rule tokens =
  | [' ']+ { skip }
  | "if" { KEYWORD }
  | "else" { KEYWORD }
  | [a-zA-Z]+ { ID }
`
	tokens, errors := buildPipeline(t, yal, "if else iffy elsewhere")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	want := []expectedToken{
		{Type: "KEYWORD", Lexeme: "if"},
		{Type: "KEYWORD", Lexeme: "else"},
		{Type: "ID", Lexeme: "iffy"},
		{Type: "ID", Lexeme: "elsewhere"},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_FloatVsIntVsDot(t *testing.T) {
	yal := `
let DIGIT = [0-9]

rule tokens =
  | [' ']+ { skip }
  | DIGIT+ '.' DIGIT+ { FLOAT }
  | DIGIT+ { INT }
  | '.' { DOT }
`
	tokens, errors := buildPipeline(t, yal, "42 3.14 .5 0.")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// "42" → INT, "3.14" → FLOAT, ".5" → DOT then INT, "0." → INT then DOT
	want := []expectedToken{
		{Type: "INT", Lexeme: "42"},
		{Type: "FLOAT", Lexeme: "3.14"},
		{Type: "DOT", Lexeme: "."},
		{Type: "INT", Lexeme: "5"},
		{Type: "INT", Lexeme: "0"},
		{Type: "DOT", Lexeme: "."},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_StringWithEscapes(t *testing.T) {
	yal := `
rule tokens =
  | [' ' '\t' '\n']+ { skip }
  | '"' ([^ '"' '\\'] | '\\' _)* '"' { STRING }
  | [a-zA-Z]+ { ID }
`
	source := `hello "world" "hello\"there" done`

	tokens, errors := buildPipeline(t, yal, source)

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	want := []expectedToken{
		{Type: "ID", Lexeme: "hello"},
		{Type: "STRING", Lexeme: `"world"`},
		{Type: "STRING", Lexeme: `"hello\"there"`},
		{Type: "ID", Lexeme: "done"},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	yal := `
rule tokens =
  | [' ']+ { skip }
  | [a-z]+ { ID }
  | [0-9]+ { INT }
`
	tokens, errors := buildPipeline(t, yal, "abc @#$ 123")

	// Should still get the valid tokens
	want := []expectedToken{
		{Type: "ID", Lexeme: "abc"},
		{Type: "INT", Lexeme: "123"},
	}
	assertTokens(t, tokens, want)

	// Should get 3 errors (one per unrecognized char: @, #, $)
	if len(errors) != 3 {
		t.Errorf("error count: got %d, want 3", len(errors))
		for i, e := range errors {
			t.Logf("  error[%d]: %s", i, e)
		}
	}
	// Each error should mention "unrecognized"
	for i, e := range errors {
		if !strings.Contains(e, "unrecognized") {
			t.Errorf("error[%d] should mention 'unrecognized': %s", i, e)
		}
	}
}

func TestIntegration_EmptySource(t *testing.T) {
	yal := `
rule tokens =
  | [a-z]+ { ID }
`
	tokens, errors := buildPipeline(t, yal, "")

	if len(tokens) != 0 {
		t.Errorf("expected no tokens, got %d", len(tokens))
	}
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d", len(errors))
	}
}

func TestIntegration_MultilineLineTracking(t *testing.T) {
	yal := `
rule tokens =
  | [' ' '\t']+ { skip }
  | '\n' { skip }
  | [a-z]+ { ID }
`
	tokens, errors := buildPipeline(t, yal, "foo\nbar\nbaz")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	want := []expectedToken{
		{Type: "ID", Lexeme: "foo", Line: 1},
		{Type: "ID", Lexeme: "bar", Line: 2},
		{Type: "ID", Lexeme: "baz", Line: 3},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_CommentsInYal(t *testing.T) {
	yal := `
(* This is a comment *)
let DIGIT = [0-9]
(* Another comment *)

rule tokens =
  | DIGIT+ { INT }
`
	tokens, errors := buildPipeline(t, yal, "42")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	want := []expectedToken{
		{Type: "INT", Lexeme: "42"},
	}
	assertTokens(t, tokens, want)
}

func TestIntegration_BooleanAndNil(t *testing.T) {
	yal := `
rule tokens =
  | [' ']+ { skip }
  | ('T' | 'F') { BOOLEAN }
  | "Nil" { NIL }
  | [a-zA-Z]+ { ID }
`
	tokens, errors := buildPipeline(t, yal, "T F Nil Nile True")

	if len(errors) > 0 {
		t.Errorf("unexpected errors: %v", errors)
	}

	// Maximal munch: "Nile" > "Nil" so it matches ID, "True" > "T" so ID
	want := []expectedToken{
		{Type: "BOOLEAN", Lexeme: "T"},
		{Type: "BOOLEAN", Lexeme: "F"},
		{Type: "NIL", Lexeme: "Nil"},
		{Type: "ID", Lexeme: "Nile"},
		{Type: "ID", Lexeme: "True"},
	}
	assertTokens(t, tokens, want)
}

// ---------- CLI / Generator Integration tests ----------

// requireGoCompilerInteg skips if the Go toolchain is not available.
func requireGoCompilerInteg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("requires go compiler")
	}
}

// projectRoot returns the project root directory (where go.mod lives).
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// The integration tests run from the project root already, but be safe.
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

func TestIntegration_OutFlag(t *testing.T) {
	requireGoCompilerInteg(t)
	root := projectRoot(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "generated_lexer.go")

	cmd := exec.Command("go", "run", "main.go", "-yal", "testdata/lexer.yal", "-out", outFile)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run main.go -yal ... -out ... failed: %v\n%s", err, string(out))
	}

	// Verify output file was created and is non-empty
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}

	// Verify the generated file can compile
	goMod := "module testlexer\n\ngo 1.26.1\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	// Rename to main.go for compilation
	mainFile := filepath.Join(tmpDir, "main.go")
	if outFile != mainFile {
		data, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("reading generated file: %v", err)
		}
		if err := os.WriteFile(mainFile, data, 0644); err != nil {
			t.Fatalf("writing main.go: %v", err)
		}
	}

	binaryPath := filepath.Join(tmpDir, "lexer_bin")
	compileCmd := exec.Command("go", "build", "-o", binaryPath, mainFile)
	compileCmd.Dir = tmpDir
	compileOut, err := compileCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiling generated lexer: %v\n%s", err, string(compileOut))
	}
}

func TestIntegration_TreeFlagStandalone(t *testing.T) {
	requireGoCompilerInteg(t)
	root := projectRoot(t)

	// tree.dot is written to the process working directory (project root)
	dotPath := filepath.Join(root, "tree.dot")

	// Clean up any pre-existing tree.dot, and ensure cleanup after the test
	os.Remove(dotPath)
	t.Cleanup(func() { os.Remove(dotPath) })

	cmd := exec.Command("go", "run", "main.go", "-yal", "testdata/lexer.yal", "-tree")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run main.go -yal ... -tree failed: %v\n%s", err, string(out))
	}

	// Verify tree.dot was created
	info, err := os.Stat(dotPath)
	if err != nil {
		t.Fatalf("tree.dot not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("tree.dot is empty")
	}
}

func TestIntegration_OutAndSrcTogether(t *testing.T) {
	requireGoCompilerInteg(t)
	root := projectRoot(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "generated_lexer.go")

	cmd := exec.Command("go", "run", "main.go",
		"-yal", "testdata/lexer.yal",
		"-src", "testdata/test.lisp",
		"-out", outFile,
	)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run main.go -yal -src -out failed: %v\n%s", err, string(out))
	}

	output := string(out)

	// Verify the output file was created
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// Verify tokens were printed to stdout (from the -src flag)
	if !strings.Contains(output, "--- Tokenization Results ---") {
		t.Errorf("expected tokenization results in stdout:\n%s", output)
	}
	if !strings.Contains(output, "KEYWORD") {
		t.Errorf("expected KEYWORD token in stdout:\n%s", output)
	}

	// Verify the generated lexer was also written
	if !strings.Contains(output, "Lexer generation successful") {
		t.Errorf("expected generation success message:\n%s", output)
	}
}

func TestIntegration_GeneratedLexerMatchesSimulator(t *testing.T) {
	requireGoCompilerInteg(t)
	root := projectRoot(t)

	// --- Step 1: Run the simulator (in-memory) ---
	yalData, err := os.ReadFile(filepath.Join(root, "testdata", "lexer.yal"))
	if err != nil {
		t.Fatalf("reading lexer.yal: %v", err)
	}
	srcData, err := os.ReadFile(filepath.Join(root, "testdata", "test.lisp"))
	if err != nil {
		t.Fatalf("reading test.lisp: %v", err)
	}

	// Build DFA entries via the full pipeline
	tmpYalDir := t.TempDir()
	yalPath := filepath.Join(tmpYalDir, "lexer.yal")
	if err := os.WriteFile(yalPath, yalData, 0644); err != nil {
		t.Fatalf("writing yal: %v", err)
	}

	parseResult, err := yalex.ParseFile(yalPath)
	if err != nil {
		t.Fatalf("parsing yal: %v", err)
	}

	expandedRules, err := yalex.Expand(parseResult.Macros, parseResult.Rules)
	if err != nil {
		t.Fatalf("expanding macros: %v", err)
	}

	var dfaEntries []lexer.DFAEntry
	for _, rule := range expandedRules {
		normalized, err := regex.Normalize(rule.Pattern)
		if err != nil {
			t.Fatalf("normalizing: %v", err)
		}
		postfix, err := regex.BuildPostfix(normalized)
		if err != nil {
			t.Fatalf("postfix: %v", err)
		}
		treeRoot, posToSymbol, err := dfa.BuildTree(postfix)
		if err != nil {
			t.Fatalf("tree: %v", err)
		}
		builtDFA := dfa.BuildDFA(treeRoot, posToSymbol, rule.Action)
		minimized := dfa.Minimize(builtDFA)
		dfaEntries = append(dfaEntries, lexer.DFAEntry{
			DFA:       minimized,
			TokenName: rule.Action,
			Priority:  rule.Priority,
		})
	}

	// Run simulator
	srcPath := filepath.Join(tmpYalDir, "test.lisp")
	if err := os.WriteFile(srcPath, srcData, 0644); err != nil {
		t.Fatalf("writing src: %v", err)
	}
	src, err := lexer.ReadSource(srcPath)
	if err != nil {
		t.Fatalf("reading source: %v", err)
	}
	simTokens, simErrors := lexer.Tokenize(dfaEntries, src)
	if len(simErrors) > 0 {
		t.Fatalf("simulator errors: %v", simErrors)
	}

	// --- Step 2: Generate, compile, and run the standalone lexer ---
	genDir := t.TempDir()
	goMod := "module testlexer\n\ngo 1.26.1\n"
	if err := os.WriteFile(filepath.Join(genDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	genFile := filepath.Join(genDir, "main.go")
	if err := generator.GenerateSource(genFile, dfaEntries); err != nil {
		t.Fatalf("GenerateSource: %v", err)
	}

	binaryPath := filepath.Join(genDir, "lexer_bin")
	compileCmd := exec.Command("go", "build", "-o", binaryPath, genFile)
	compileCmd.Dir = genDir
	compileOut, err := compileCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compilation: %v\n%s", err, string(compileOut))
	}

	// Copy test.lisp to genDir
	genSrcPath := filepath.Join(genDir, "test.lisp")
	if err := os.WriteFile(genSrcPath, srcData, 0644); err != nil {
		t.Fatalf("writing test.lisp: %v", err)
	}

	runCmd := exec.Command(binaryPath, "-src", genSrcPath)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated lexer: %v\n%s", err, string(runOut))
	}

	// --- Step 3: Parse the generated lexer's output and compare ---
	genOutput := string(runOut)

	// Parse tokens from the generated lexer's output
	// Format: [line] TYPE         lexeme
	genTokens := parseGeneratedOutput(t, genOutput)

	// Compare token counts
	if len(genTokens) != len(simTokens) {
		t.Errorf("token count mismatch: simulator=%d, generated=%d", len(simTokens), len(genTokens))
		maxLen := len(simTokens)
		if len(genTokens) > maxLen {
			maxLen = len(genTokens)
		}
		for i := 0; i < maxLen; i++ {
			simStr := "<missing>"
			genStr := "<missing>"
			if i < len(simTokens) {
				simStr = fmt.Sprintf("[%d] %s %q", simTokens[i].Line, simTokens[i].Type, simTokens[i].Lexeme)
			}
			if i < len(genTokens) {
				genStr = fmt.Sprintf("[%d] %s %q", genTokens[i].line, genTokens[i].typ, genTokens[i].lexeme)
			}
			if simStr != genStr {
				t.Logf("  diff[%d]: sim=%s | gen=%s", i, simStr, genStr)
			}
		}
		return
	}

	// Compare each token
	for i := range simTokens {
		sim := simTokens[i]
		gen := genTokens[i]

		if sim.Type != gen.typ {
			t.Errorf("token[%d] type mismatch: simulator=%q, generated=%q", i, sim.Type, gen.typ)
		}
		if sim.Lexeme != gen.lexeme {
			t.Errorf("token[%d] lexeme mismatch: simulator=%q, generated=%q", i, sim.Lexeme, gen.lexeme)
		}
		if sim.Line != gen.line {
			t.Errorf("token[%d] line mismatch: simulator=%d, generated=%d", i, sim.Line, gen.line)
		}
	}
}

// parsedToken represents a token parsed from the generated lexer's stdout.
type parsedToken struct {
	line   int
	typ    string
	lexeme string
}

// parseGeneratedOutput parses the generated lexer's stdout into tokens.
// Expected format per line: [line] TYPE         lexeme
func parseGeneratedOutput(t *testing.T, output string) []parsedToken {
	t.Helper()
	var tokens []parsedToken

	lines := strings.Split(output, "\n")
	inTokens := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "--- Tokens ---" {
			inTokens = true
			continue
		}
		if strings.HasPrefix(trimmed, "--- ") && trimmed != "--- Tokens ---" {
			inTokens = false
			continue
		}
		if !inTokens || trimmed == "" {
			continue
		}
		// Parse: [line] TYPE         lexeme
		if !strings.HasPrefix(trimmed, "[") {
			continue
		}

		var lineNum int
		var typ, lexeme string
		// Find the closing bracket
		closeBracket := strings.Index(trimmed, "]")
		if closeBracket < 0 {
			continue
		}
		_, err := fmt.Sscanf(trimmed[1:closeBracket], "%d", &lineNum)
		if err != nil {
			continue
		}
		rest := strings.TrimSpace(trimmed[closeBracket+1:])
		// Split into type and lexeme (type is the first whitespace-delimited field)
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		typ = fields[0]
		if len(fields) >= 2 {
			lexeme = strings.Join(fields[1:], " ")
		}

		tokens = append(tokens, parsedToken{
			line:   lineNum,
			typ:    typ,
			lexeme: lexeme,
		})
	}

	return tokens
}

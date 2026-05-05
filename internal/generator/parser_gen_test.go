package generator

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/shared"
	"genanalex/internal/yapar"
)

func TestGenerateParserSource_ValidStandaloneParser(t *testing.T) {
	requireGoCompiler(t)

	grammar, table := mustBuildParserArtifacts(t, `%token INT PLUS WS
IGNORE WS
%%
expr : INT PLUS INT ;
`)

	tmpDir := setupCompilableDir(t)
	outputFile := filepath.Join(tmpDir, "main.go")

	if err := GenerateParserSource(outputFile, grammar, table); err != nil {
		t.Fatalf("GenerateParserSource() error = %v", err)
	}

	binaryPath := filepath.Join(tmpDir, "parser_bin")
	cmd := exec.Command("go", "build", "-o", binaryPath, outputFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated parser compilation failed: %v\n%s", err, string(out))
	}

	tokensPath := filepath.Join(tmpDir, "tokens.json")
	tokens := []shared.Token{
		{Type: "INT", Lexeme: "12", Line: 1},
		{Type: "WS", Lexeme: " ", Line: 1},
		{Type: "PLUS", Lexeme: "+", Line: 1},
		{Type: "WS", Lexeme: " ", Line: 1},
		{Type: "INT", Lexeme: "34", Line: 1},
	}
	encoded, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(tokensPath, encoded, 0o644); err != nil {
		t.Fatalf("WriteFile(tokens) error = %v", err)
	}

	runCmd := exec.Command(binaryPath, "-tokens", tokensPath)
	runCmd.Dir = tmpDir
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated parser execution failed: %v\n%s", err, string(runOut))
	}
	if !strings.Contains(string(runOut), "Input accepted") {
		t.Fatalf("generated parser output = %q, want acceptance message", string(runOut))
	}
}

func TestGenerateParserSource_RejectsInvalidInput(t *testing.T) {
	requireGoCompiler(t)

	grammar, table := mustBuildParserArtifacts(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

	tmpDir := setupCompilableDir(t)
	outputFile := filepath.Join(tmpDir, "main.go")
	if err := GenerateParserSource(outputFile, grammar, table); err != nil {
		t.Fatalf("GenerateParserSource() error = %v", err)
	}

	binaryPath := filepath.Join(tmpDir, "parser_bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, outputFile)
	buildCmd.Dir = tmpDir
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated parser compilation failed: %v\n%s", err, string(buildOut))
	}

	tokensPath := filepath.Join(tmpDir, "tokens.json")
	tokens := []shared.Token{
		{Type: "ID", Lexeme: "x", Line: 7},
		{Type: "ID", Lexeme: "y", Line: 7},
	}
	encoded, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(tokensPath, encoded, 0o644); err != nil {
		t.Fatalf("WriteFile(tokens) error = %v", err)
	}

	runCmd := exec.Command(binaryPath, "-tokens", tokensPath)
	runCmd.Dir = tmpDir
	runOut, err := runCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("generated parser run error = nil, want syntax failure; output=%s", string(runOut))
	}
	if !strings.Contains(string(runOut), "expected [PLUS]") {
		t.Fatalf("generated parser output = %q, want expected tokens diagnostics", string(runOut))
	}
}

func mustBuildParserArtifacts(t *testing.T, spec string) (*yapar.Grammar, *yapar.ParsingTable) {
	t.Helper()

	parsed, err := yapar.Parse(spec)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	grammar, err := yapar.BuildGrammar(parsed)
	if err != nil {
		t.Fatalf("BuildGrammar() error = %v", err)
	}
	ff, err := yapar.ComputeFirstFollow(grammar)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	states, transitions, err := yapar.BuildCanonicalCollection(grammar)
	if err != nil {
		t.Fatalf("BuildCanonicalCollection() error = %v", err)
	}
	table, err := yapar.BuildSLRTable(grammar, ff, states, transitions)
	if err != nil {
		t.Fatalf("BuildSLRTable() error = %v", err)
	}
	return grammar, table
}

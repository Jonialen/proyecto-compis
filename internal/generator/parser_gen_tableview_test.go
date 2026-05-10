package generator

import (
	"os"
	"path/filepath"
	"testing"

	"genanalex/internal/yapar"
)

func TestGenerateParserSourceFromTableView_MatchesLegacyOutput(t *testing.T) {
	grammar, table := mustBuildParserArtifacts(t, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)
	ff, err := yapar.ComputeFirstFollow(grammar)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}
	parser, err := yapar.BuildParser(grammar, ff, yapar.MethodSLR)
	if err != nil {
		t.Fatalf("BuildParser() error = %v", err)
	}

	tmpDir := t.TempDir()
	legacyPath := filepath.Join(tmpDir, "legacy.go")
	viewPath := filepath.Join(tmpDir, "view.go")

	if err := GenerateParserSource(legacyPath, grammar, table); err != nil {
		t.Fatalf("GenerateParserSource() error = %v", err)
	}
	if err := GenerateParserSourceFromTableView(viewPath, grammar, parser.Table()); err != nil {
		t.Fatalf("GenerateParserSourceFromTableView() error = %v", err)
	}

	legacySource, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("ReadFile(legacy) error = %v", err)
	}
	viewSource, err := os.ReadFile(viewPath)
	if err != nil {
		t.Fatalf("ReadFile(view) error = %v", err)
	}
	if string(viewSource) != string(legacySource) {
		t.Fatal("generated source from TableView differs from legacy output")
	}
}

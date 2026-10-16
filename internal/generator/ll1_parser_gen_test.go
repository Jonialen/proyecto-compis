package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/yapar"
)

func TestGenerateLL1ParserSourceRejectsMissingArtifacts(t *testing.T) {
	for _, tt := range []struct {
		name, message string
		grammar       *yapar.Grammar
	}{
		{name: "grammar", message: "grammar is required"},
		{name: "table", grammar: &yapar.Grammar{}, message: "parsing table is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "parser.go")
			const sentinel = "existing file\n"
			if err := os.WriteFile(path, []byte(sentinel), 0o600); err != nil {
				t.Fatal(err)
			}
			err := GenerateLL1ParserSource(path, tt.grammar, nil)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error=%v, want %q", err, tt.message)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != sentinel {
				t.Fatalf("rejected generation changed destination: %q, %v", data, err)
			}
		})
	}
}

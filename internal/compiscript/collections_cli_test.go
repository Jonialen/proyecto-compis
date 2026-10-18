package compiscript

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectionCLIExecution(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range collectionCases {
		t.Run(tt.name, func(t *testing.T) { assertClosureCLI(t, cli, tt.source, tt.want) })
	}
	example, err := os.ReadFile("../../testdata/compiscript/tac/collections.cps")
	if err != nil {
		t.Fatal(err)
	}
	assertClosureCLI(t, cli, string(example), map[string]any{"shared": float64(9), "sum": float64(12), "first": float64(3), "second": float64(6), "continued": float64(4)})
}

func TestCollectionCLIRejection(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, source := range collectionRejections {
		t.Run(source, func(t *testing.T) { assertClosureCLIRejection(t, cli, source) })
	}
}

func TestCollectionCLIRuntimeGuards(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range collectionFaults {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "guard.cps")
			if err := os.WriteFile(path, []byte(tt.source), 0600); err != nil {
				t.Fatal(err)
			}
			want := Compile([]byte(tt.source))
			if want.TAC == nil || len(want.Diagnostics) != 0 {
				t.Fatalf("Compile: %v", want.Diagnostics)
			}
			cmd := exec.Command(cli, "-format=json", path)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("exit=%v stderr=%s", err, &stderr)
			}
			encoded, _ := json.MarshalIndent(want, "", "  ")
			if stdout.String() != string(encoded)+"\n" {
				t.Fatal("JSON mismatch")
			}
			var report CompilationReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			values, err := runFunctionFrames(report.TAC)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("values=%v err=%v", values, err)
			}
			for name, want := range tt.effects {
				if values["s1."+name] != want {
					t.Fatalf("effects %s=%v want %v", name, values["s1."+name], want)
				}
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != tt.source {
				t.Fatal("source changed")
			}
		})
	}
}

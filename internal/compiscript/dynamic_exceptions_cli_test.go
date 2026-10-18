package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDynamicExceptionsCLI(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range dynamicExceptionsCases {
		t.Run(tt.name, func(t *testing.T) {
			assertDynamicExceptionsAccepted(t, tt.source)
			path := filepath.Join(t.TempDir(), "input.cps")
			source := []byte(tt.source)
			if err := os.WriteFile(path, source, 0600); err != nil {
				t.Fatal(err)
			}
			// Even a failed CLI assertion must check that input was preserved.
			t.Cleanup(func() {
				after, err := os.ReadFile(path)
				if err != nil || sha256.Sum256(after) != sha256.Sum256(source) {
					t.Error("CLI changed source input")
				}
			})
			want := Compile(source)
			for _, format := range []string{"json", "tac"} {
				cmd := exec.Command(cli, "-format="+format, path)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				if err := cmd.Run(); err != nil || stderr.Len() != 0 {
					var rejected CompilationReport
					decodeErr := json.Unmarshal(stdout.Bytes(), &rejected)
					t.Fatalf("%s: want exit=0 stderr=empty; got exit=%v stderr=%q JSON decode=%v diagnostics=%+v", format, err, &stderr, decodeErr, rejected.Diagnostics)
				}
				if format == "tac" {
					if want.TAC == nil || stdout.String() != want.TAC.String() {
						t.Fatal("TAC text differs from Compile")
					}
					continue
				}
				encoded, err := json.MarshalIndent(want, "", "  ")
				if err != nil || stdout.String() != string(encoded)+"\n" {
					t.Fatal("whole JSON differs from Compile")
				}
				var report CompilationReport
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
					t.Fatal(err)
				}
				assertValuesEffects(t, report, tt)
			}
		})
	}
}

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

func TestValuesEffectsCLI(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range valuesEffectsCases {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.cps")
			if err := os.WriteFile(path, []byte(tt.source), 0600); err != nil {
				t.Fatal(err)
			}
			want := Compile([]byte(tt.source))
			for _, format := range []string{"json", "tac"} {
				cmd := exec.Command(cli, "-format="+format, path)
				var out, stderr bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil || stderr.Len() != 0 {
					t.Fatalf("%s exit=%v stdout=%s stderr=%s", format, err, &out, &stderr)
				}
				if format == "json" {
					encoded, _ := json.MarshalIndent(want, "", "  ")
					if out.String() != string(encoded)+"\n" {
						t.Fatal("JSON differs from Compile")
					}
					var report CompilationReport
					if err := json.Unmarshal(out.Bytes(), &report); err != nil {
						t.Fatal(err)
					}
					assertValuesEffects(t, report, tt)
				} else if want.TAC == nil || out.String() != want.TAC.String() {
					t.Fatal("TAC text differs")
				}
			}
			after, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(after) != sha256.Sum256([]byte(tt.source)) {
				t.Fatal("input changed")
			}
		})
	}
}

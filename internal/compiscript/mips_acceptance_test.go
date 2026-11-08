package compiscript

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Reuse the previous phase's accepted-source matrices, asserting values through
// public print statements rather than inspecting MIPS memory or interpreting IR.
func TestMIPSPhase3Acceptance(t *testing.T) {
	if testing.Short() {
		t.Skip("real MARS acceptance omitted in short mode")
	}
	cases := append(append([]valuesEffectsCase{}, valuesEffectsCases...), dynamicExceptionsCases...)
	for _, tt := range objectCases {
		cases = append(cases, valuesEffectsCase{name: "object/" + tt.name, source: tt.source, want: tt.want})
	}
	for _, tt := range collectionCases {
		cases = append(cases, valuesEffectsCase{name: "list/" + tt.name, source: tt.source, want: tt.want})
	}
	cases = append(cases, valuesEffectsCase{name: "exception in a list", source: `try{let bad=1/0;}catch(e){print([e]);}`, output: "[division by zero]\n"})
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			source, want := tt.source, tt.output
			if tt.fault == "" {
				names := make([]string, 0, len(tt.want))
				for name := range tt.want {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					source += "\nprint(" + name + ");"
					switch value := tt.want[name].(type) {
					case nil:
						want += "null\n"
					case float64:
						want += strconv.FormatFloat(value, 'g', -1, 64) + "\n"
					case bool:
						want += strconv.FormatBool(value) + "\n"
					case string:
						want += value + "\n"
					default:
						t.Fatalf("unsupported expected scalar type %T", value)
					}
				}
			}
			input := []byte(source)
			before := sha256.Sum256(input)
			report := CompileMIPS(input)
			if len(report.Diagnostics) != 0 || report.MIPSText == "" {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			if sha256.Sum256(input) != before {
				t.Fatal("compilation modified source")
			}
			assembly := filepath.Join(t.TempDir(), "program.asm")
			if err := os.WriteFile(assembly, []byte(report.MIPSText), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "scripts/mars.sh"), "1000000", assembly)
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			code := 0
			if err := cmd.Run(); err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					code = e.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			if ctx.Err() != nil {
				t.Fatal("simulation budget exceeded", ctx.Err())
			}
			if stderr.String() != "\n" {
				t.Fatalf("simulator stderr=%q", stderr.String())
			}
			if tt.fault == "" {
				if code != 0 || stdout.String() != want {
					t.Fatalf("exit=%d stdout=%q want=%q", code, stdout.String(), want)
				}
			} else {
				// Existing matrices name fault classes (e.g. "undefined"), not a target's
				// exact diagnostic spelling. Preserve the exact pre-fault output prefix.
				output := stdout.String()
				if code != 1 || !strings.HasPrefix(output, want) || !strings.Contains(strings.TrimPrefix(output, want), tt.fault) {
					t.Fatalf("exit=%d stdout=%q want prefix=%q fault=%q", code, output, want, tt.fault)
				}
			}
		})
	}
	t.Logf("MARS acceptance matrix contains %d scenarios", len(cases))
}

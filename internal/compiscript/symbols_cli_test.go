package compiscript

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCompilationSymbolsCLI(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct {
		source string
		values map[string]any
	}{
		{`const x=2; {let x=3; x=x+1;} let y=x+4;`, map[string]any{"x": float64(2), "y": float64(6)}},
		{escapingCounter, map[string]any{"first": float64(11), "second": float64(12)}},
		{`class A {let x:integer=1; function constructor(){} function get():integer{return this.x;}} class B : A {let y:integer=2;} let b=new B(); let n=b.get();`, map[string]any{"n": float64(1)}},
		{`let xs=[1,2]; let total=0; foreach (x in xs){total=total+x;}`, map[string]any{"total": float64(3)}},
	} {
		source := tt.source
		t.Run(source, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "symbols.cps")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(cli, "-format=json", path)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("exit=%v stderr=%s", err, &stderr)
			}
			var report CompilationReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			// Assert raw CLI JSON as well: old decoders intentionally ignore additions.
			if !bytes.Contains(stdout.Bytes(), []byte(`"codegen"`)) {
				t.Fatal("CLI omitted codegen")
			}
			assertSymbolReport(t, report)
			want, _ := json.MarshalIndent(Compile([]byte(source)), "", "  ")
			if stdout.String() != string(want)+"\n" {
				t.Fatal("CLI/core JSON mismatch")
			}
			after, _ := os.ReadFile(path)
			if string(after) != source {
				t.Fatal("source changed")
			}
			assertClosureCLI(t, cli, source, tt.values)
		})
	}
	assertClosureCLI(t, cli, `let x=1+2;`, map[string]any{"x": float64(3)})
	for _, source := range []string{`let x=;`, `let x=missing;`, `let x=1; try{missing;}catch(e){}`} {
		assertClosureCLIRejection(t, cli, source)
	}
}

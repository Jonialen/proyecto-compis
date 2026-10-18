package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

func TestIndependentT3B1Recheck(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct {
		name, source string
		want         float64
	}{
		{"shadow nested foreach extraction", `class A{let n:integer=7;}function make():A{return new A();}let fs=[[make]];let x=0;{class A{let n:integer=50;}foreach(row in fs){foreach(fn in row){let alias=fn;let a=alias();x=x+a.n;}}}`, 7},
		{"index assignment expression alias", `class A{let n:integer=7;}function make():A{return new A();}let fs=[make];let g=(fs[0]=make);let alias=g;let a=alias();let x=a.n;`, 7},
		{"leading empty nested function lists", `class A{let n:integer=7;}function make():A{return new A();}let fs=[[],[make]];let g=fs[1][0];let a=g();let x=a.n;`, 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			hash := sha256.Sum256(input)
			analysis, report := Analyze(input), Compile(input)
			if len(analysis.Diagnostics) != 0 || len(report.Diagnostics) != 0 || report.TAC == nil {
				t.Fatalf("Analyze=%v Compile=%v", analysis.Diagnostics, report.Diagnostics)
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil || values["s1.x"] != tt.want {
				t.Fatalf("values=%v err=%v", values, err)
			}
			if hash != sha256.Sum256(input) || !reflect.DeepEqual(analysis.Scopes, report.Scopes) || !reflect.DeepEqual(report, Compile(input)) {
				t.Fatal("mutation/nondeterminism")
			}
			tree, _ := frontend.Parse(input)
			scopes, _ := semantic.Analyze(tree)
			before, _ := json.Marshal([]any{tree, scopes})
			program, diagnostics := tac.Generate(tree, scopes)
			after, _ := json.Marshal([]any{tree, scopes})
			if len(diagnostics) != 0 || !bytes.Equal(before, after) || !reflect.DeepEqual(program, report.TAC) {
				t.Fatal("Generate mutation/output")
			}
			assertClosureCLI(t, cli, tt.source, map[string]any{"x": tt.want})
		})
	}
	for _, tt := range []struct{ name, source string }{
		{"nominal replacement before effects", `let effects=0;class A{}function make():A{return new A();}let fs=[make];{class A{}function other():A{effects=effects+1;return new A();}fs[0]=other;}let f=fs[0];let a=f();`},
		{"nested rank replacement before effects", `let effects=0;class A{}function make(a:A[][]):A[][]{return a;}function other(a:A[]):A[][]{effects=effects+1;return [a];}let fs=[make];fs[0]=other;let f=fs[0];let a=f([[new A()]]);`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			hash := sha256.Sum256(input)
			analysis, report := Analyze(input), Compile(input)
			if len(analysis.Diagnostics) != 0 || len(report.Diagnostics) != 0 || report.TAC == nil {
				t.Fatalf("Analyze=%v Compile=%v", analysis.Diagnostics, report.Diagnostics)
			}
			check := func(program *tac.Program) {
				t.Helper()
				values, err := runFunctionFrames(program)
				if err == nil || err.Error() != "indirect call signature mismatch" || values["s1.effects"] != float64(0) {
					t.Fatalf("values=%v fault=%v", values, err)
				}
			}
			check(report.TAC)
			path := filepath.Join(t.TempDir(), "guard.cps")
			if err := os.WriteFile(path, input, 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(cli, "-format=json", path)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("exit=%v stderr=%q", err, stderr.String())
			}
			encoded, _ := json.MarshalIndent(report, "", "  ")
			if stdout.String() != string(encoded)+"\n" {
				t.Fatal("JSON mismatch")
			}
			var decoded CompilationReport
			if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			check(decoded.TAC)
			after, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(after) != hash || sha256.Sum256(input) != hash || !reflect.DeepEqual(analysis.Scopes, report.Scopes) {
				t.Fatal("source/scopes mutation")
			}
		})
	}
}

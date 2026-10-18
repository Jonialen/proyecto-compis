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
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

var acceptanceRejected = []struct{ name, source, phase string }{
	{"malformed for", "for(let i=0; i<;i=i+1){", "syntax"},
	{"missing while body", "while(true)", "syntax"},
	{"missing do body", "do", "syntax"},
	{"missing foreach body", "foreach(x in [1])", "syntax"},
	{"missing for body", "for(let i=0; i<2; i=i+1)", "syntax"},
	{"const root", "const rows:integer[][]=[[1]];rows[0][0]=2;", "semantic"},
	{"nested shape", "let rows:integer[][]=[[1]];rows[0]=true;", "semantic"},
	{"unsupported dead code", "let a=[1];if(false){print(a[0]);}", "codegen"},
}

func TestAcceptanceAtomicRecovery(t *testing.T) {
	for _, tt := range acceptanceRejected {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			before := sha256.Sum256(source)
			program, pd := frontend.Parse(source)
			analysis := Analyze(source)
			r := Compile(source)
			if r.TAC != nil || r.Codegen != nil || len(r.Diagnostics) == 0 {
				t.Fatalf("non-atomic rejection: %+v", r)
			}
			found := false
			for _, d := range r.Diagnostics {
				found = found || string(d.Phase) == tt.phase
				if d.Span.Start.Line < 1 || d.Span.Start.Column < 1 || d.Span.End.Offset < d.Span.Start.Offset {
					t.Fatalf("unlocated diagnostic: %+v", d)
				}
			}
			if !found || tt.phase == "syntax" && len(pd) == 0 {
				t.Fatalf("wrong diagnostics: %+v, %+v", pd, r.Diagnostics)
			}
			if !reflect.DeepEqual(analysis.Scopes, r.Scopes) || !reflect.DeepEqual(r, Compile(source)) {
				t.Fatal("changed analysis scopes or nondeterministic rejection")
			}
			again, _ := frontend.Parse(source)
			if !reflect.DeepEqual(program, again) || sha256.Sum256(source) != before {
				t.Fatal("source or recovered AST changed")
			}
		})
	}
}

func TestAcceptanceMixedExecution(t *testing.T) {
	source, err := os.ReadFile("../../testdata/compiscript/tac/acceptance.cps")
	if err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(source)
	r := Compile(source)
	if r.TAC == nil || len(r.Diagnostics) != 0 {
		t.Fatalf("Compile: %+v", r.Diagnostics)
	}
	assertSymbolReport(t, r)
	want := map[string]any{
		"first": float64(30), "second": float64(43), "independent": float64(7),
		"continued": float64(57), "shared": float64(37), "total": float64(10), "result": float64(147),
	}
	values, err := runFunctionFrames(r.TAC)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range want {
		if values["s1."+name] != value {
			t.Errorf("%s=%v, want %v", name, values["s1."+name], value)
		}
	}
	program, pd := frontend.Parse(source)
	scopes, sd := semantic.Analyze(program)
	snapshot, _ := json.Marshal([]any{program, scopes})
	generated, ds := tac.Generate(program, scopes)
	after, _ := json.Marshal([]any{program, scopes})
	if len(pd)+len(sd)+len(ds) != 0 || !reflect.DeepEqual(generated, r.TAC) || !bytes.Equal(snapshot, after) {
		t.Fatal("Generate changed AST/scopes or differs from Compile")
	}
	if !reflect.DeepEqual(Analyze(source).Scopes, r.Scopes) || !reflect.DeepEqual(r, Compile(source)) || sha256.Sum256(source) != before {
		t.Fatal("source/scopes changed or compilation is nondeterministic")
	}
	assertClosureCLI(t, buildClosureCLI(t), string(source), want)
}

func TestAcceptanceRecoveryCLI(t *testing.T) {
	cli := buildClosureCLI(t)
	analyzer := filepath.Join(t.TempDir(), "compiscript")
	if out, err := exec.Command("go", "build", "-o", analyzer, "../../cmd/compiscript").CombinedOutput(); err != nil {
		t.Fatalf("build Analyze CLI: %v\n%s", err, out)
	}
	for _, tt := range acceptanceRejected {
		t.Run(tt.name, func(t *testing.T) {
			assertClosureCLIRejection(t, cli, tt.source)
			path := filepath.Join(t.TempDir(), "rejected.cps")
			if err := os.WriteFile(path, []byte(tt.source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(analyzer, path)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			want, _ := json.MarshalIndent(Analyze([]byte(tt.source)), "", "  ")
			if err := cmd.Run(); err != nil || stderr.Len() != 0 || stdout.String() != string(want)+"\n" {
				t.Fatalf("legacy Analyze exit/stdout/stderr: %v %q %q", err, &stdout, &stderr)
			}
			var report model.AnalysisReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(after) != sha256.Sum256([]byte(tt.source)) {
				t.Fatal("Analyze changed source")
			}
			cmd = exec.Command(cli, "-format=invalid", path)
			stdout.Reset()
			stderr.Reset()
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 || stdout.Len() != 0 || stderr.String() != "compiscript-tac: format must be tac or json\n" {
				t.Fatalf("usage policy: %v %q %q", err, &stdout, &stderr)
			}
		})
	}
}

func TestAcceptanceValidLoops(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, source := range []string{
		"let n=0;while(n<2){n=n+1;}", "let n=0;do{n=n+1;}while(n<2);",
		"let n=0;for(let i=0;i<2;i=i+1){n=n+1;}", "let n=0;foreach(x in [1,1]){n=n+x;}",
	} {
		t.Run(source, func(t *testing.T) {
			r := Compile([]byte(source))
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("valid loop rejected: %+v", r.Diagnostics)
			}
			values, err := runFunctionFrames(r.TAC)
			if err != nil || values["s1.n"] != float64(2) {
				t.Fatalf("loop execution: %v, %v", values, err)
			}
			assertClosureCLI(t, cli, source, map[string]any{"n": float64(2)})
		})
	}
}

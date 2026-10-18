package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Build in scratch space and execute the public CLI's JSON, not an in-process substitute.
func buildClosureCLI(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the public TAC CLI")
	}
	path := filepath.Join(t.TempDir(), "compiscript-tac")
	cmd := exec.Command("go", "build", "-o", path, "../../cmd/compiscript-tac")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return path
}

func TestClosureCLIAtomicRejection(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, source := range []string{
		`function outer():integer{let x=read(); let n=8; function read():integer{return n;} return x;} let x=outer();`,
		`function outer(){function bad(){try{}catch(e){}}}`,
		`function f():integer{return 1;} let alias=f; function bad(){try{}catch(e){}} let x=alias();`,
	} {
		t.Run(source, func(t *testing.T) {
			assertClosureCLIRejection(t, cli, source)
		})
	}
}

func assertClosureCLIRejection(t *testing.T, cli, source string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.cps")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	want := Compile([]byte(source))
	if want.TAC != nil || len(want.Diagnostics) == 0 {
		t.Fatalf("fixture: %+v", want)
	}
	for _, format := range []string{"tac", "json"} {
		cmd := exec.Command(cli, "-format="+format, path)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 1 {
			t.Fatalf("exit=%v", err)
		}
		if format == "tac" {
			var diagnostics bytes.Buffer
			for _, d := range want.Diagnostics {
				fmt.Fprintf(&diagnostics, "%s %s at %d:%d: %s\n", d.Phase, d.Code, d.Span.Start.Line, d.Span.Start.Column, d.Message)
			}
			if stdout.Len() != 0 || stderr.String() != diagnostics.String() {
				t.Fatalf("stdout=%q stderr=%q", &stdout, &stderr)
			}
		} else {
			encoded, _ := json.MarshalIndent(want, "", "  ")
			if stdout.String() != string(encoded)+"\n" || stderr.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q", &stdout, &stderr)
			}
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(after) != sha256.Sum256([]byte(source)) {
		t.Fatal("rejected source changed")
	}
}

func TestClosureCLIExecution(t *testing.T) {
	cli := buildClosureCLI(t)
	assertClosureCLI(t, cli, escapingCounter, map[string]any{"first": float64(11), "second": float64(12)})
	example, err := os.ReadFile("../../testdata/compiscript/tac/closures.cps")
	if err != nil {
		t.Fatal(err)
	}
	assertClosureCLI(t, cli, string(example), map[string]any{"first": float64(11), "second": float64(12), "independent": float64(21), "continued": float64(13)})
}

func assertClosureCLI(t *testing.T, cli, source string, wantValues map[string]any) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "counter.cps")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	want := Compile([]byte(source))
	for _, format := range []string{"tac", "json"} {
		cmd := exec.Command(cli, "-format="+format, path)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil || stderr.Len() != 0 {
			t.Fatalf("%s: exit=%v stdout=%s stderr=%s", format, err, &stdout, &stderr)
		}
		if want.TAC == nil {
			t.Fatalf("compile: %+v", want.Diagnostics)
		}
		if format == "tac" {
			if stdout.String() != want.TAC.String() {
				t.Fatal("text output mismatch")
			}
		} else {
			encoded, _ := json.MarshalIndent(want, "", "  ")
			if stdout.String() != string(encoded)+"\n" {
				t.Fatal("JSON output mismatch")
			}
			var report CompilationReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			got, err := runFunctionFrames(report.TAC)
			if err != nil {
				t.Fatalf("CLI execution: %v %v", got, err)
			}
			for name, value := range wantValues {
				if got["s1."+name] != value {
					t.Fatalf("CLI execution %s=%v want %v", name, got["s1."+name], value)
				}
			}
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(after) != sha256.Sum256([]byte(source)) {
		t.Fatal("source changed")
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"genanalex/internal/compiscript"
	"genanalex/internal/compiscript/mips"
)

var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "compiscript-mips-cli-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binary = filepath.Join(dir, "compiscript-mips")
	out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	code := 1
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n%s", err, out)
	} else {
		code = m.Run()
	}
	os.RemoveAll(dir)
	os.Exit(code)
}
func invoke(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.String(), stderr.String()
}
func TestMIPSCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.cps")
	source := `print(1+2);`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	r := compiscript.Compile([]byte(source))
	want, err := mips.Generate(r.TAC)
	if err != nil {
		t.Fatal(err)
	}
	code, out, stderr := invoke(t, path)
	if code != 0 || out != want || stderr != "" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, stderr)
	}
	code, out, stderr = invoke(t, "-format=json", path)
	var report compiscript.MIPSCompilationReport
	if err := json.Unmarshal([]byte(out), &report); err != nil || code != 0 || report.MIPSText != want || stderr != "" {
		t.Fatalf("JSON exit=%d stderr=%q err=%v", code, stderr, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != source {
		t.Fatal("source modified", err)
	}
	code, out, stderr = invoke(t, "-h")
	if code != 0 || out != "" || !strings.Contains(stderr, "mips|json") {
		t.Fatalf("help exit=%d out=%q stderr=%q", code, out, stderr)
	}
	code, out, stderr = invoke(t, "-format=bad", path)
	if code != 2 || out != "" || stderr != "compiscript-mips: format must be mips or json\n" {
		t.Fatalf("format exit=%d out=%q stderr=%q", code, out, stderr)
	}
	invalid := `let n=missing;`
	os.WriteFile(path, []byte(invalid), 0600)
	old := compiscript.Compile([]byte(invalid))
	var expected strings.Builder
	for _, d := range old.Diagnostics {
		fmt.Fprintf(&expected, "%s %s at %d:%d: %s\n", d.Phase, d.Code, d.Span.Start.Line, d.Span.Start.Column, d.Message)
	}
	code, out, stderr = invoke(t, path)
	if code != 1 || out != "" || stderr != expected.String() {
		t.Fatalf("rejection exit=%d out=%q stderr=%q", code, out, stderr)
	}
	after, err = os.ReadFile(path)
	if err != nil || string(after) != invalid {
		t.Fatal("rejected source changed", err)
	}
	code, out, stderr = invoke(t, "-format=json", path)
	if err := json.Unmarshal([]byte(out), &report); err != nil || code != 1 || report.MIPSText != "" || stderr != "" || len(report.Diagnostics) == 0 {
		t.Fatalf("rejected JSON exit=%d out=%q stderr=%q err=%v", code, out, stderr, err)
	}
}

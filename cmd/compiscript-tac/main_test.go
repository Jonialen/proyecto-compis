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

	compiscript "genanalex/internal/compiscript"
)

var cliPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "compiscript-tac-cli-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cliPath = filepath.Join(dir, "compiscript-tac")
	output, buildErr := exec.Command("go", "build", "-o", cliPath, ".").CombinedOutput()
	code := 1
	if buildErr != nil {
		fmt.Fprintf(os.Stderr, "build TAC CLI: %v\n%s", buildErr, output)
	} else {
		code = m.Run()
	}
	os.RemoveAll(dir)
	os.Exit(code)
}

func execute(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(cliPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, stdout.String(), stderr.String()
}

func fixture(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.cps")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLICompilesNumericSources(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"precedence", "let x=1+2*3;", "t0:integer = 2:integer * 3:integer\nt0:integer = 1:integer + t0:integer\ns1.x:integer = t0:integer\n"},
		{"empty", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(t, tt.source)
			for i := 0; i < 2; i++ {
				code, out, errOut := execute(t, path)
				if code != 0 || out != tt.want || errOut != "" {
					t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
				}
			}
			code, out, errOut := execute(t, "-format", "json", path)
			want, err := json.MarshalIndent(compiscript.Compile([]byte(tt.source)), "", "  ")
			if err != nil || code != 0 || out != string(want)+"\n" || errOut != "" {
				t.Fatalf("JSON exit=%d stdout=%q stderr=%q encode=%v", code, out, errOut, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != tt.source {
				t.Fatalf("source changed: %q, %v", after, err)
			}
		})
	}
}

func TestCLICompilationErrorsHaveNoPartialTAC(t *testing.T) {
	for _, source := range []string{"let x=1; print(x);", "let x=missing;", "function f()", "let x:integer;"} {
		t.Run(source, func(t *testing.T) {
			path := fixture(t, source)
			code, out, errOut := execute(t, path)
			if code != 1 || out != "" || !strings.Contains(errOut, " at 1:") {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
			}
			code, out, errOut = execute(t, "-format=json", path)
			var report compiscript.CompilationReport
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatalf("invalid report: %q, %v", out, err)
			}
			if code != 1 || errOut != "" || report.TAC != nil || len(report.Diagnostics) == 0 {
				t.Fatalf("exit=%d stderr=%q report=%+v", code, errOut, report)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != source {
				t.Fatalf("rejected input changed: %q, %v", after, err)
			}
		})
	}
}

func TestCLIConditionalSources(t *testing.T) {
	source := "let x=false; if(true) x=!x; else x=false;"
	path := fixture(t, source)
	report := compiscript.Compile([]byte(source))
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		t.Fatalf("conditional compilation failed: %+v", report.Diagnostics)
	}
	code, out, errOut := execute(t, path)
	wantText := "s1.x:boolean = false:boolean\nif true:boolean goto L0\ngoto L1\nL0:\nt0:boolean = ! s1.x:boolean\ns1.x:boolean = t0:boolean\ngoto L2\nL1:\ns1.x:boolean = false:boolean\nL2:\n"
	if code != 0 || out != wantText || errOut != "" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
	}
	code, out, errOut = execute(t, "-format=json", path)
	want, err := json.MarshalIndent(report, "", "  ")
	if err != nil || code != 0 || out != string(want)+"\n" || errOut != "" {
		t.Fatalf("JSON exit=%d stdout=%q stderr=%q encode=%v", code, out, errOut, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != source {
		t.Fatalf("source changed: %q %v", after, err)
	}
}

func TestCLILoopSources(t *testing.T) {
	for _, source := range []string{
		"let x=0; while(x<3){x=x+1; if(x==2)continue;}",
		"let x=0; do{x=x+1; continue;}while(x<3);",
		"let x=0; for(;;x=x+1){if(x>=3)break;}",
	} {
		t.Run(source, func(t *testing.T) {
			path := fixture(t, source)
			report := compiscript.Compile([]byte(source))
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			for repeat := 0; repeat < 2; repeat++ {
				code, out, errOut := execute(t, path)
				if code != 0 || out != report.TAC.String() || errOut != "" {
					t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
				}
				code, out, errOut = execute(t, "-format=json", path)
				want, err := json.MarshalIndent(report, "", "  ")
				if err != nil || code != 0 || out != string(want)+"\n" || errOut != "" {
					t.Fatalf("exit=%d stdout=%q stderr=%q encode=%v", code, out, errOut, err)
				}
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != source {
				t.Fatalf("source changed: %q %v", after, err)
			}
		})
	}
}

func TestCLIRejectsSkippedUnsupportedCode(t *testing.T) {
	for _, tt := range []struct{ source, wantError string }{
		{"if(false){print(1);}", "codegen TAC_UNSUPPORTED at 1:11: unsupported statement in numeric TAC\n"},
		{"let x=true||((5%2)>0);", "codegen TAC_UNSUPPORTED at 1:15: unsupported expression in numeric TAC\n"},
		{"while(false){print(1);}", "codegen TAC_UNSUPPORTED at 1:14: unsupported statement in numeric TAC\n"},
		{"for(;false;5%2){}", "codegen TAC_UNSUPPORTED at 1:12: unsupported expression in numeric TAC\n"},
		{"break;", "semantic SEM_TRANSFER at 1:1: break outside loop or switch\n"},
		{"continue;", "semantic SEM_TRANSFER at 1:1: continue outside loop\n"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			path := fixture(t, tt.source)
			code, out, errOut := execute(t, path)
			if code != 1 || out != "" || errOut != tt.wantError {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
			}
			code, out, errOut = execute(t, "-format=json", path)
			want, err := json.MarshalIndent(compiscript.Compile([]byte(tt.source)), "", "  ")
			if err != nil || code != 1 || out != string(want)+"\n" || errOut != "" {
				t.Fatalf("exit=%d stdout=%q stderr=%q encode=%v", code, out, errOut, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != tt.source {
				t.Fatalf("source changed: %q %v", after, err)
			}
		})
	}
}

func TestCLIUsageAndInputErrors(t *testing.T) {
	valid := fixture(t, "let x=1;")
	invalidUTF8 := fixture(t, string([]byte{0xff}))
	missing := filepath.Join(t.TempDir(), "missing.cps")
	for _, tt := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"missing argument", nil, 2, "usage: compiscript-tac"},
		{"extra argument", []string{valid, valid}, 2, "usage: compiscript-tac"},
		{"extension", []string{"input.txt"}, 2, "expected a .cps file"},
		{"missing file", []string{missing}, 1, "read"},
		{"UTF-8", []string{invalidUTF8}, 1, "source is not valid UTF-8"},
		{"format", []string{"-format=xml", valid}, 2, "format must be tac or json"},
		{"unknown flag", []string{"-unknown", valid}, 2, "flag provided but not defined"},
		{"help", []string{"-h"}, 0, "usage: compiscript-tac"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := execute(t, tt.args...)
			if code != tt.code || out != "" || !strings.Contains(errOut, tt.want) {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, out, errOut)
			}
		})
	}
}

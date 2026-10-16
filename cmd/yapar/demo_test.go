package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("demo acceptance builds and executes the CLI")
	}
	const demos = "testdata/yapar/demos/"
	for _, file := range []string{
		"required.yalp", "nullable.yalp", "indirect.yalp", "assignment.yal", "assignment.yalp",
		"valid.txt", "invalid.txt", "empty.txt", "assignment_valid.txt", "assignment_invalid.txt",
	} {
		if _, err := os.ReadFile(filepath.Join("../..", demos, file)); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(t.TempDir(), "yapar")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	type demoCase struct {
		name, method, grammar, lexer, input, message string
		code                                         int
	}
	tests := []demoCase{
		{name: "indirect left recursion", method: "ll1", grammar: "indirect.yalp", message: "left recursion", code: 1},
		{name: "SLR assignment conflict", method: "slr", grammar: "assignment.yalp", message: "shift/reduce", code: 1},
		{name: "LALR assignment accepted", method: "lalr", grammar: "assignment.yalp", lexer: demos + "assignment.yal", input: "assignment_valid.txt", message: "Input accepted."},
		{name: "LALR assignment rejected", method: "lalr", grammar: "assignment.yalp", lexer: demos + "assignment.yal", input: "assignment_invalid.txt", message: "syntax error", code: 1},
	}
	for _, method := range []string{"ll1", "slr", "lalr"} {
		for _, scenario := range []struct {
			name, grammar, input, message string
			code                          int
		}{
			{name: "valid", grammar: "required.yalp", input: "valid.txt", message: "Input accepted."},
			{name: "invalid", grammar: "required.yalp", input: "invalid.txt", message: "syntax error", code: 1},
			{name: "required empty", grammar: "required.yalp", input: "empty.txt", message: "syntax error", code: 1},
			{name: "nullable empty", grammar: "nullable.yalp", input: "empty.txt", message: "Input accepted."},
		} {
			tests = append(tests, demoCase{
				name: method + " " + scenario.name, method: method, grammar: scenario.grammar,
				lexer: "testdata/yapar/arithmetic.yal", input: scenario.input, message: scenario.message, code: scenario.code,
			})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"-yalp", demos + tt.grammar, "-method", tt.method}
			if tt.input != "" {
				args = append(args, "-yal", tt.lexer, "-src", demos+tt.input)
			}
			cmd := exec.Command(binary, args...)
			cmd.Dir = "../.."
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
			if code != tt.code {
				t.Fatalf("exit = %d, want %d; stdout=%s stderr=%s", code, tt.code, stdout.String(), stderr.String())
			}
			output := stdout.String()
			if tt.code != 0 {
				output = stderr.String()
				if strings.Contains(stdout.String(), "Input accepted.") {
					t.Fatal("rejected demo reported acceptance")
				}
			} else if stderr.Len() != 0 {
				t.Fatalf("successful demo wrote stderr: %s", stderr.String())
			}
			if !strings.Contains(output, tt.message) {
				t.Fatalf("output = %q, want %q", output, tt.message)
			}
		})
	}
}

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesStandaloneParsers(t *testing.T) {
	if testing.Short() {
		t.Skip("standalone acceptance compiles and executes generated parsers")
	}
	readDemo := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("../../testdata/yapar/demos", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, tt := range []struct {
		name, method, grammar, valid, invalid string
	}{
		{
			name: "LL1 required", method: "ll1", grammar: "required.yalp",
			valid: `[{"type":"INT","lexeme":"7","line":1}]`, invalid: `[{"type":"PLUS","lexeme":"+","line":1}]`,
		},
		{
			name: "LL1 nullable", method: "ll1", grammar: "nullable.yalp",
			valid: `[]`, invalid: `[{"type":"PLUS","lexeme":"+","line":1}]`,
		},
		{
			name: "LL1 predictive sequence and ignored tokens", method: "ll1", grammar: "predictive.yalp",
			valid: readDemo("predictive_valid.tokens.json"), invalid: readDemo("predictive_invalid.tokens.json"),
		},
		{
			name: "LALR assignment not SLR", method: "lalr", grammar: "assignment.yalp",
			valid: readDemo("assignment_valid.tokens.json"), invalid: readDemo("assignment_invalid.tokens.json"),
		},
		{
			name: "LALR nullable", method: "lalr", grammar: "nullable.yalp",
			valid: `[]`, invalid: `[{"type":"PLUS","lexeme":"+","line":1}]`,
		},
		{
			name: "existing SLR", method: "slr", grammar: "required.yalp",
			valid: `[{"type":"INT","lexeme":"7","line":1}]`, invalid: `[{"type":"PLUS","lexeme":"+","line":1}]`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "parser.go")
			var stdout, stderr bytes.Buffer
			args := []string{"-yalp", filepath.Join("../../testdata/yapar/demos", tt.grammar), "-method", tt.method, "-out", output}
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatalf("generate %s: %v; stderr=%s", tt.method, err, stderr.String())
			}
			if !strings.Contains(stdout.String(), "Standalone parser generated successfully.") || stderr.Len() != 0 {
				t.Fatalf("generation stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			generated, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			stdout.Reset()
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatalf("repeat generation: %v", err)
			}
			repeated, err := os.ReadFile(output)
			if err != nil || !bytes.Equal(generated, repeated) {
				t.Fatalf("generation is not reproducible: %v", err)
			}
			binary := filepath.Join(dir, "parser")
			build := exec.Command("go", "build", "-o", binary, output)
			build.Dir = dir
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("compile standalone parser: %v\n%s", err, out)
			}
			type tokenCase struct {
				name, json string
				code       int
				message    string
			}
			inputs := []tokenCase{
				{name: "accepted", json: tt.valid},
				{name: "rejected", json: tt.invalid, code: 1, message: "syntax error at line 1"},
			}
			if tt.method == "ll1" {
				inputs = append(inputs,
					tokenCase{name: "malformed JSON", json: "{", code: 1, message: "Error loading tokens:"},
					tokenCase{name: "extra tokens", json: `[{"type":"INT","lexeme":"7","line":1},{"type":"INT","lexeme":"8","line":7}]`, code: 1, message: "syntax error at line 7"},
				)
			}
			for _, input := range inputs {
				t.Run(input.name, func(t *testing.T) {
					tokens := writeTempFile(t, dir, input.name+".json", input.json)
					cmd := exec.Command(binary, "-tokens", tokens)
					cmd.Dir = dir
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
					if code != input.code {
						t.Fatalf("exit=%d, want %d; stdout=%s stderr=%s", code, input.code, stdout.String(), stderr.String())
					}
					if input.code == 0 {
						if !strings.Contains(stdout.String(), "Input accepted") || stderr.Len() != 0 {
							t.Fatalf("accepted stdout=%q stderr=%q", stdout.String(), stderr.String())
						}
					} else if stdout.Len() != 0 || !strings.Contains(stderr.String(), input.message) {
						t.Fatalf("rejected stdout=%q stderr=%q", stdout.String(), stderr.String())
					}
				})
			}
		})
	}
}

func TestStandaloneRejectionsPreserveDestination(t *testing.T) {
	for _, tt := range []struct {
		name, grammar string
		args          []string
		want          string
	}{
		{name: "left recursive ll1", grammar: "indirect.yalp", args: []string{"-method", "ll1"}, want: `build parser backend: yapar ll1 left recursion at non-terminal "tail" in production 2`},
		{name: "compare", args: []string{"-compare"}, want: "standalone parser generation is incompatible with -compare"},
		{name: "lr0", args: []string{"-method", "lr0"}, want: `build parser backend: parser method "lr0" not implemented yet: valid options are ll1, lr0, slr, lr1, lalr`},
		{name: "lr1", args: []string{"-method", "lr1"}, want: `build parser backend: parser method "lr1" not implemented yet: valid options are ll1, lr0, slr, lr1, lalr`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const sentinel = "existing destination\n"
			dest := writeTempFile(t, t.TempDir(), "parser.go", sentinel)
			grammar := tt.grammar
			if grammar == "" {
				grammar = "required.yalp"
			}
			args := append([]string{"-yalp", filepath.Join("../../testdata/yapar/demos", grammar), "-out", dest}, tt.args...)
			var stdout, stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error=%v, want %q", err, tt.want)
			}
			data, err := os.ReadFile(dest)
			if err != nil || string(data) != sentinel {
				t.Fatalf("rejected generation changed destination: %q, %v", data, err)
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected run stderr: %q", stderr.String())
			}
		})
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	dir, err := os.MkdirTemp("", "compiscript-cli-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	cliPath = filepath.Join(dir, "compiscript")
	command := exec.Command("go", "build", "-o", cliPath, ".")
	if output, err := command.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build CLI: %v\n%s", err, output)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestCLIReportMatchesFacade(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantErrors  bool
		wantEmpty   string
		wantPresent string
	}{
		{"valid analysis", "valid/types.cps", false, `"diagnostics": []`, `"scopes": [`},
		{"analysis with diagnostics", "invalid/types.cps", true, `"children": []`, `"diagnostics": [`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "compiscript", tt.fixture)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			report := compiscript.Analyze(source)
			if gotErrors := len(report.Diagnostics) > 0; gotErrors != tt.wantErrors {
				t.Fatalf("has diagnostics = %t, want %t", gotErrors, tt.wantErrors)
			}
			want, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, '\n')

			first := executeCLI(t, path)
			second := executeCLI(t, path)
			if first.exitCode != 0 || second.exitCode != 0 {
				t.Fatalf("exit codes = %d, %d; stderr = %q, %q", first.exitCode, second.exitCode, first.stderr, second.stderr)
			}
			if first.stderr != "" || second.stderr != "" {
				t.Fatalf("stderr = %q, %q; want empty", first.stderr, second.stderr)
			}
			if !bytes.Equal(first.stdout, want) {
				t.Fatalf("stdout differs from facade marshal\ngot:\n%s\nwant:\n%s", first.stdout, want)
			}
			if !bytes.Equal(first.stdout, second.stdout) {
				t.Fatal("stdout changed between identical runs")
			}
			if !bytes.Contains(first.stdout, []byte(tt.wantEmpty)) || !bytes.Contains(first.stdout, []byte(tt.wantPresent)) {
				t.Fatalf("stdout does not preserve report arrays:\n%s", first.stdout)
			}
		})
	}
}

func TestCLIRejectsInvalidInputs(t *testing.T) {
	temp := t.TempDir()
	sourceDirectory := filepath.Join(temp, "directory.cps")
	if err := os.Mkdir(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	invalidUTF8 := filepath.Join(temp, "invalid-utf8.cps")
	if err := os.WriteFile(invalidUTF8, []byte{0xff}, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
		wantErr  string
	}{
		{"missing argument", nil, 2, "usage: compiscript <file.cps>"},
		{"extra argument", []string{"one.cps", "two.cps"}, 2, "usage: compiscript <file.cps>"},
		{"wrong extension", []string{"program.txt"}, 2, "expected a .cps file"},
		{"missing file", []string{filepath.Join(temp, "missing.cps")}, 1, "compiscript: read"},
		{"source path is directory", []string{sourceDirectory}, 1, "compiscript: read"},
		{"invalid UTF-8", []string{invalidUTF8}, 1, "compiscript: source is not valid UTF-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executeCLI(t, tt.args...)
			if result.exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d; stderr = %q", result.exitCode, tt.wantExit, result.stderr)
			}
			if len(result.stdout) != 0 {
				t.Fatalf("stdout = %q, want empty", result.stdout)
			}
			if !strings.Contains(result.stderr, tt.wantErr) {
				t.Fatalf("stderr = %q, want substring %q", result.stderr, tt.wantErr)
			}
		})
	}
}

type commandResult struct {
	stdout   []byte
	stderr   string
	exitCode int
}

func executeCLI(t *testing.T, args ...string) commandResult {
	t.Helper()
	command := exec.Command(cliPath, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run CLI: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return commandResult{stdout.Bytes(), stderr.String(), exitCode}
}

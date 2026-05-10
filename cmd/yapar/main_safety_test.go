package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunMachineReadableFormatsKeepStdoutClean(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		assertStdout func(t *testing.T, stdout string)
	}{
		{
			name: "json",
			args: []string{"-format", "json"},
			assertStdout: func(t *testing.T, stdout string) {
				t.Helper()
				var payload struct {
					Method string `json:"method"`
				}
				if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
					t.Fatalf("json.Unmarshal(stdout) error = %v\nstdout=%s", err, stdout)
				}
				if payload.Method != "slr" {
					t.Fatalf("method = %q, want slr", payload.Method)
				}
			},
		},
		{
			name: "dot",
			args: []string{"-method", "lalr", "-format", "dot"},
			assertStdout: func(t *testing.T, stdout string) {
				t.Helper()
				if !strings.HasPrefix(stdout, "digraph LR0") {
					t.Fatalf("stdout = %q, want DOT graph prefix", stdout)
				}
				if strings.Contains(stdout, "[*] Loading YAPar specification") {
					t.Fatalf("stdout = %q, want no progress logs mixed into DOT output", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			yalpPath := writeParserSpec(t, dir, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			args := append([]string{"-yalp", yalpPath}, tt.args...)
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
			}

			tt.assertStdout(t, stdout.String())
			if !strings.Contains(stderr.String(), "Parser pipeline ready") {
				t.Fatalf("stderr = %q, want pipeline summary", stderr.String())
			}
		})
	}
}

func TestParseOutputFormatNormalizesCaseAndWhitespace(t *testing.T) {
	got, err := parseOutputFormat("  JSON  ")
	if err != nil {
		t.Fatalf("parseOutputFormat() error = %v", err)
	}
	if got != outputFormatJSON {
		t.Fatalf("parseOutputFormat() = %q, want %q", got, outputFormatJSON)
	}
}

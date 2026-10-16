package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"genanalex/internal/shared"
	"genanalex/internal/yapar"
)

func TestLL1StandaloneDiagnosticParity(t *testing.T) {
	if testing.Short() {
		t.Skip("diagnostic parity compiles a standalone parser")
	}
	spec, err := yapar.Parse("%token INT PLUS WS\nIGNORE WS\n%%\nstart : prefix INT tail ; prefix : a b ; a : ; b : ; tail : PLUS INT tail | ;")
	if err != nil {
		t.Fatal(err)
	}
	g, err := yapar.BuildGrammar(spec)
	if err != nil {
		t.Fatal(err)
	}
	ff, err := yapar.ComputeFirstFollow(g)
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := yapar.BuildParser(g, ff, yapar.MethodLL1)
	if err != nil {
		t.Fatal(err)
	}
	table, err := yapar.BuildLL1Table(g, ff)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source, binary := filepath.Join(dir, "parser.go"), filepath.Join(dir, "parser")
	if err := GenerateLL1ParserSource(source, g, table); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, tt := range []struct {
		name   string
		tokens []shared.Token
	}{
		{name: "nullable prefixes", tokens: []shared.Token{{Type: "INT"}}},
		{name: "ignored and recursive", tokens: []shared.Token{{Type: "WS"}, {Type: "INT"}, {Type: "PLUS"}, {Type: "INT"}}},
		{name: "empty"},
		{name: "unknown line fallback", tokens: []shared.Token{{Type: "UNKNOWN", Line: -1}, {Type: "WS", Line: 9}}},
		{name: "missing terminal EOF", tokens: []shared.Token{{Type: "INT", Line: 2}, {Type: "PLUS", Line: 2}, {Type: "WS", Line: 9}}},
		{name: "extra token", tokens: []shared.Token{{Type: "INT"}, {Type: "INT", Line: 7}}},
		{name: "external EOF", tokens: []shared.Token{{Type: "INT"}, {Type: "$", Line: 3}, {Type: "PLUS", Line: 7}}},
		{name: "external EOF line fallback", tokens: []shared.Token{{Type: "$"}, {Type: "INT", Line: 9}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, parseErr := oracle.Parse(tt.tokens)
			wantCode, wantOut, wantErr := 0, fmt.Sprintf("[+] Input accepted by parser for %s\n", g.Start), ""
			if parseErr != nil {
				wantCode, wantOut, wantErr = 1, "", fmt.Sprintf("Parse error: %v\n", parseErr)
			} else if result == nil || !result.Accepted {
				t.Fatal("oracle returned neither acceptance nor error")
			}
			data, err := json.Marshal(tt.tokens)
			if err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(dir, "tokens.json")
			if err := os.WriteFile(input, data, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "-tokens", input)
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
			if code != wantCode || stdout.String() != wantOut || stderr.String() != wantErr {
				t.Fatalf("got (%d,%q,%q), want (%d,%q,%q)", code, stdout.String(), stderr.String(), wantCode, wantOut, wantErr)
			}
			after, err := os.ReadFile(input)
			if err != nil || !bytes.Equal(data, after) {
				t.Fatalf("standalone parser changed token file: %v", err)
			}
		})
	}
}

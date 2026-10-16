package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStandaloneExplicitEmptyModeFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("mode validation compiles generated parsers and a local spy")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	t.Setenv("STANDALONE_MODE_MARKER", marker)
	spySource := writeTempFile(t, dir, "spy.go", `package main
import ("fmt"; "os")
func main() {
  if err := os.WriteFile(os.Getenv("STANDALONE_MODE_MARKER"), []byte("called"), 0600); err != nil { panic(err) }
  fmt.Print("[{\"type\":\"INT\",\"lexeme\":\"7\",\"line\":1}]")
}
`)
	spy := filepath.Join(dir, "trusted spy")
	buildStandaloneBinary(t, spySource, spy)
	source := writeTempFile(t, dir, "source.txt", "7")
	tokens := writeTempFile(t, dir, "tokens.json", "[]")
	for _, method := range []string{"ll1", "slr", "lalr"} {
		t.Run(method, func(t *testing.T) {
			generated, binary := filepath.Join(dir, method+".go"), filepath.Join(dir, method)
			var stdout, stderr bytes.Buffer
			if err := run([]string{"-yalp", "../../testdata/yapar/demos/nullable.yalp", "-method", method, "-out", generated}, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			buildStandaloneBinary(t, generated, binary)
			for _, tc := range []struct {
				name string
				args []string
			}{
				{name: "empty tokens with source mode", args: []string{"-tokens", "", "-lexer", spy, "-src", source}},
				{name: "empty lexer with token mode", args: []string{"-tokens", tokens, "-lexer", ""}},
				{name: "empty source with token mode", args: []string{"-tokens", tokens, "-src", ""}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
					code, out, diagnostics := standaloneResult(t, binary, tc.args...)
					if _, err := os.Stat(marker); !os.IsNotExist(err) {
						t.Errorf("invalid mode invoked child: %v", err)
					}
					if code != 1 || out != "" || diagnostics == "" {
						t.Errorf("must reject: (%d,%q,%q)", code, out, diagnostics)
					}
					for path, want := range map[string]string{source: "7", tokens: "[]"} {
						got, err := os.ReadFile(path)
						if err != nil || string(got) != want {
							t.Errorf("input changed: %q, %v", got, err)
						}
					}
				})
			}
		})
	}
}

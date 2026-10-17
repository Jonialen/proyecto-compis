package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeANTLRGoWorkaround(t *testing.T) {
	const header = "// Code generated from Example.g4 by ANTLR 4.13.2. DO NOT EDIT.\n"
	const tail = "goto errorExit // Trick to prevent compiler error if the label is not used\n"
	const unusedLabel = "package generated\nfunc rule() int { localctx := 7\nerrorExit: if localctx == 0 { localctx = 1 }; return localctx\n" + tail + "}\n"
	for _, tt := range []struct {
		name, source string
		jumps        int
		unchanged    bool
		wantError    bool
	}{
		{name: "live recovery jump", source: header + "package generated\nfunc rule(fail bool) int { localctx := 7; if fail { goto errorExit }; localctx = 9\nerrorExit: if fail { localctx = 1 }; return localctx\n" + tail + "}\n", jumps: 1},
		{name: "nested function label scope", source: header + "package generated\nfunc rule() int { localctx := 7; _ = func() { goto errorExit; errorExit: return }\nerrorExit: if localctx == 0 { localctx = 1 }; return localctx\n" + tail + "}\n", jumps: 1},
		{name: "unused recovery label", source: header + unusedLabel},
		{name: "non generated source", source: unusedLabel, unchanged: true},
		{name: "different ANTLR version", source: strings.Replace(header, "4.13.2", "4.14.0", 1) + unusedLabel, unchanged: true},
		{name: "invalid generated Go", source: header + "package generated\nfunc broken(\n", wantError: true},
		{name: "unexpected workaround shape", source: header + "package generated\nfunc rule() int { other := 7\nerrorExit: if other == 0 { other = 1 }; return other\n" + tail + "}\n", wantError: true},
		{name: "nonterminal workaround", source: header + "package generated\nfunc rule() int { localctx := 7\nerrorExit: if localctx == 0 { localctx = 1 }; return localctx\n" + tail + "return localctx\n}\n", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "stage with spaces")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "parser.go")
			if err := os.WriteFile(path, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			err := run([]string{dir})
			if (err != nil) != tt.wantError {
				t.Fatalf("run() error=%v, want error=%v", err, tt.wantError)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tt.unchanged || tt.wantError {
				if string(got) != tt.source {
					t.Fatal("normalization changed unsupported or rejected source")
				}
				return
			}
			if strings.Contains(string(got), "Trick to prevent compiler error") {
				t.Fatal("normalization kept the unreachable workaround")
			}
			if strings.Count(string(got), "goto errorExit") != tt.jumps || strings.Count(string(got), "errorExit:") != tt.jumps {
				t.Fatalf("recovery jump/label preservation failed:\n%s", got)
			}
			if err := run([]string{dir}); err != nil {
				t.Fatal(err)
			}
			again, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, again) {
				t.Fatalf("normalization is not idempotent: %v", err)
			}
		})
	}
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	compiscript "genanalex/internal/compiscript"
	"genanalex/internal/compiscript/model"
)

type acceptanceCase struct{ name, path, hash string }

func TestCompiscriptAcceptanceCorpus(t *testing.T) {
	cases := readAcceptanceManifest(t)
	cli := filepath.Join(t.TempDir(), "compiscript")
	if output, err := exec.Command("go", "build", "-o", cli, "../compiscript").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", "..", tc.path))
			if err != nil {
				t.Fatal(err)
			}
			want := compiscript.Analyze(source)
			if again := compiscript.Analyze(source); !reflect.DeepEqual(want, again) {
				t.Fatal("facade report changed between runs")
			}
			canonical, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(canonical)
			gotHash := hex.EncodeToString(digest[:])
			if gotHash != tc.hash {
				t.Errorf("report hash = %s, want %s", gotHash, tc.hash)
			}
			assertReportContract(t, want, canonical)

			cliFirst := runAcceptanceCLI(t, cli, filepath.Join("..", "..", tc.path))
			cliSecond := runAcceptanceCLI(t, cli, filepath.Join("..", "..", tc.path))
			if !bytes.Equal(cliFirst, cliSecond) {
				t.Fatal("CLI output changed between runs")
			}
			var cliReport model.AnalysisReport
			if err := json.Unmarshal(cliFirst, &cliReport); err != nil || !reflect.DeepEqual(cliReport, want) {
				t.Fatalf("CLI report differs from facade: %v", err)
			}

			body, _ := json.Marshal(map[string]string{"source": string(source)})
			ideFirst := request(t, http.MethodPost, "/api/compiscript/analyze", "application/json", string(body))
			ideSecond := request(t, http.MethodPost, "/api/compiscript/analyze", "application/json", string(body))
			if ideFirst.Code != http.StatusOK || !bytes.Equal(ideFirst.Body.Bytes(), ideSecond.Body.Bytes()) {
				t.Fatalf("IDE status/repeat = %d/%t", ideFirst.Code, bytes.Equal(ideFirst.Body.Bytes(), ideSecond.Body.Bytes()))
			}
			var ideReport model.AnalysisReport
			if err := json.Unmarshal(ideFirst.Body.Bytes(), &ideReport); err != nil || !reflect.DeepEqual(ideReport, want) || !reflect.DeepEqual(ideReport, cliReport) {
				t.Fatalf("IDE, CLI, and facade reports differ: %v", err)
			}
		})
	}
}

func readAcceptanceManifest(t *testing.T) []acceptanceCase {
	t.Helper()
	file, err := os.Open("../../testdata/compiscript/acceptance/manifest.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cases, err := parseAcceptanceManifest("../..", file)
	if err != nil {
		t.Fatal(err)
	}
	return cases
}

func parseAcceptanceManifest(root string, input io.Reader) ([]acceptanceCase, error) {
	reader := csv.NewReader(input)
	reader.Comma, reader.Comment = '\t', '#'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("manifest is empty")
	}
	cases := make([]acceptanceCase, len(records))
	names, paths := map[string]bool{}, map[string]bool{}
	for i, record := range records {
		if len(record) != 3 {
			return nil, fmt.Errorf("manifest row %d has %d fields", i+1, len(record))
		}
		name, path, hash := record[0], record[1], record[2]
		if names[name] || paths[path] || i > 0 && name <= cases[i-1].name {
			return nil, fmt.Errorf("duplicate or unordered manifest row %q", name)
		}
		clean := filepath.ToSlash(filepath.Clean(path))
		safeRoot := strings.HasPrefix(path, "testdata/compiscript/") || strings.HasPrefix(path, "internal/compiscript/semantic/testdata/")
		if filepath.IsAbs(path) || clean != path || !safeRoot || len(hash) != 64 || hash != strings.ToLower(hash) {
			return nil, fmt.Errorf("unsafe manifest row %q", name)
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return nil, fmt.Errorf("invalid hash for %q", name)
		}
		safeBase := "testdata/compiscript"
		if strings.HasPrefix(path, "internal/") {
			safeBase = "internal/compiscript/semantic/testdata"
		}
		resolvedRoot, err := filepath.EvalSymlinks(filepath.Join(root, safeBase))
		if err != nil {
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("manifest path %q: %w", path, err)
		}
		rel, err := filepath.Rel(resolvedRoot, resolved)
		info, statErr := os.Stat(resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || statErr != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("manifest path %q is not a safe regular file", path)
		}
		names[name], paths[path] = true, true
		cases[i] = acceptanceCase{name, path, hash}
	}
	return cases, nil
}

func runAcceptanceCLI(t *testing.T, cli, path string) []byte {
	t.Helper()
	command := exec.Command(cli, path)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("CLI failed: %v; stderr=%q", err, stderr.String())
	}
	return stdout.Bytes()
}

func assertReportContract(t *testing.T, report model.AnalysisReport, encoded []byte) {
	t.Helper()
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Span.Start.Line < 1 || diagnostic.Span.Start.Column < 1 || diagnostic.Span.End.Offset < diagnostic.Span.Start.Offset {
			t.Errorf("diagnostic is not source-located: %+v", diagnostic)
		}
	}
	for i, scope := range report.Scopes {
		if scope.ID != i+1 || !sort.SliceIsSorted(scope.Symbols, func(a, b int) bool { return scope.Symbols[a].Span.Start.Offset < scope.Symbols[b].Span.Start.Offset }) {
			t.Errorf("unordered scope/symbols: %+v", scope)
		}
	}
	for _, key := range []string{"children", "diagnostics", "scopes", "symbols", "params"} {
		if bytes.Contains(encoded, []byte(`"`+key+`":null`)) {
			t.Errorf("%s array is null", key)
		}
	}
}

func TestCompiscriptRecoveryPreservesValidSiblings(t *testing.T) {
	source, err := os.ReadFile("../../testdata/compiscript/acceptance/recoverable.cps")
	if err != nil {
		t.Fatal(err)
	}
	report := compiscript.Analyze(source)
	if len(report.Diagnostics) == 0 || len(report.AST.Children) < 3 {
		t.Fatalf("recovery lost diagnostics/tree siblings: %+v", report)
	}
	names := make([]string, len(report.Scopes[0].Symbols))
	for i, symbol := range report.Scopes[0].Symbols {
		names[i] = symbol.Name
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "before") || !strings.Contains(joined, "after") {
		t.Fatalf("recovery lost valid symbols: %v", names)
	}
}

func TestAcceptanceManifestRejectsUnsafeInput(t *testing.T) {
	hash := strings.Repeat("a", 64)
	row := func(name, path, digest string) string { return strings.Join([]string{name, path, digest}, "\t") + "\n" }
	validPath := "testdata/compiscript/valid/types.cps"
	tests := map[string]string{
		"empty":          "# comments only\n",
		"duplicate name": row("a", validPath, hash) + row("a", "testdata/compiscript/invalid/types.cps", hash),
		"duplicate path": row("a", validPath, hash) + row("b", validPath, hash),
		"absolute":       row("a", "/tmp/source.cps", hash),
		"escaping":       row("a", "../source.cps", hash),
		"noncanonical":   row("a", "testdata/compiscript/acceptance/../multibyte.cps", hash),
		"bad hash":       row("a", validPath, strings.ToUpper(hash)),
		"nonhex hash":    row("a", validPath, strings.Repeat("g", 64)),
		"short hash":     row("a", validPath, hash[:63]),
		"unstable order": row("b", validPath, hash) + row("a", "testdata/compiscript/invalid/types.cps", hash),
		"missing":        row("a", "testdata/compiscript/acceptance/missing.cps", hash),
		"non-regular":    row("a", "testdata/compiscript/acceptance", hash),
		"unsafe root":    row("a", "go.mod", hash),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseAcceptanceManifest("../..", strings.NewReader(input)); err == nil {
				t.Fatal("unsafe manifest was accepted")
			}
		})
	}
}

func TestAcceptanceManifestRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	safe := filepath.Join(root, "testdata/compiscript")
	outside := filepath.Join(root, "outside.cps")
	for _, err := range []error{os.MkdirAll(safe, 0o755), os.WriteFile(outside, nil, 0o644), os.Symlink(outside, filepath.Join(safe, "escape.cps"))} {
		if err != nil {
			t.Fatal(err)
		}
	}
	row := "a\ttestdata/compiscript/escape.cps\t" + strings.Repeat("a", 64) + "\n"
	if _, err := parseAcceptanceManifest(root, strings.NewReader(row)); err == nil {
		t.Fatal("escaping symlink was accepted")
	}
}

func TestAcceptanceRequirementEvidence(t *testing.T) {
	tests := []struct {
		name, path string
		want       []string
		message    string
		start, end int
	}{
		{"operators valid", "operators_valid.cps", nil, "", 0, 0},
		{"operators invalid", "operators_invalid.cps", []string{"SEM_OPERATOR", "SEM_OPERATOR", "SEM_OPERATOR", "SEM_OPERATOR", "SEM_OPERATOR", "SEM_OPERATOR"}, "", 0, 0},
		{"null valid", "null_valid.cps", nil, "", 0, 0},
		{"null invalid", "null_invalid.cps", []string{"SEM_TYPE", "SEM_TYPE"}, "", 0, 0},
		{"foreach valid", "foreach_valid.cps", nil, "", 0, 0},
		{"foreach invalid", "foreach_invalid.cps", []string{"SEM_TYPE"}, "foreach iterable must be a list", 17, 19},
		{"lexical invalid", "lexical_invalid.cps", []string{"lexical"}, "", 0, 0},
		{"constant initializer invalid", "const_initializer_invalid.cps", []string{"syntax"}, "mismatched input ';' expecting '='", 20, 21},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("../../testdata/compiscript/acceptance", tc.path))
			if err != nil {
				t.Fatal(err)
			}
			report := compiscript.Analyze(source)
			got := make([]string, len(report.Diagnostics))
			for i, diagnostic := range report.Diagnostics {
				got[i] = diagnostic.Code
				if diagnostic.Span.Start.Line < 1 || diagnostic.Span.Start.Column < 1 {
					t.Fatalf("unlocated diagnostic: %+v", diagnostic)
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("diagnostic codes = %v, want %v", got, tc.want)
			}
			if tc.message != "" {
				d := report.Diagnostics[0]
				if d.Message != tc.message || d.Span.Start.Offset != tc.start || d.Span.End.Offset != tc.end || d.Span.Start.Line != 1 || d.Span.Start.Column != tc.start+1 || d.Span.End.Column != tc.end+1 {
					t.Fatalf("diagnostic = %+v", d)
				}
			}
		})
	}

	source, err := os.ReadFile("../../testdata/compiscript/acceptance/multibyte.cps")
	if err != nil {
		t.Fatal(err)
	}
	report := compiscript.Analyze(source)
	span := report.Diagnostics[0].Span
	if span.Start.Offset != 63 || span.End.Offset != 70 || span.Start.Line != 2 || span.Start.Column != 22 || span.End.Line != 2 || span.End.Column != 29 {
		t.Fatalf("multibyte diagnostic span = %+v", span)
	}
}

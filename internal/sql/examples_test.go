package sql

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the .golden and .expected files of testdata/sql")

const examplesDir = "../../testdata/sql"

func exampleFiles(t *testing.T, sub string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(examplesDir, sub, "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples in %s (err %v)", sub, err)
	}
	return files
}

func readExample(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// compareFile checks got against the stored file, or rewrites it with -update.
func compareFile(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./internal/sql -update to create it)", err)
	}
	if string(want) != got {
		t.Errorf("%s differs:\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

// dumpDB renders every file of the database directory, sorted by name.
func dumpDB(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if strings.HasSuffix(n, ".idx") {
			// The stamp holds the table file's modification time: mask it.
			head, rest, _ := strings.Cut(content, "\n")
			f := strings.Split(head, ",")
			f[4] = "<stamp>"
			content = strings.Join(f, ",") + "\n" + rest
		}
		b.WriteString("=== " + n + " ===\n" + content)
	}
	return b.String()
}

// Every ok/*.sql script must execute cleanly and leave exactly the files in
// its .golden next to it: every statement result (rows and read plan for a
// SELECT) followed by the files of the database.
func TestExamplesExecuteAndPersist(t *testing.T) {
	for _, f := range exampleFiles(t, "ok") {
		t.Run(filepath.Base(f), func(t *testing.T) {
			e := newEngine(t)
			rep := e.Run(readExample(t, f), true)
			if len(rep.Diagnostics) != 0 || !rep.Executed {
				t.Fatalf("diagnostics = %v", rep.Diagnostics)
			}
			var out strings.Builder
			for _, r := range rep.Results {
				out.WriteString(FormatResult(r))
			}
			compareFile(t, strings.TrimSuffix(f, ".sql")+".golden", out.String()+dumpDB(t, e.Dir))

			// The stored database must reload and re-analyse cleanly.
			cat, err := LoadCatalog(e.Dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, tb := range cat.TableList() {
				if _, err := readRows(e.Dir, tb); err != nil {
					t.Errorf("reload %s: %v", tb.Name, err)
				}
			}
		})
	}
}

// analyze/*.sql hold complex statements that are only analysed. The header
// "-- statements: N" pins how many statements the AST must contain.
func TestExamplesAnalyzeOnly(t *testing.T) {
	for _, f := range exampleFiles(t, "analyze") {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src := readExample(t, f)
			e := newEngine(t)
			rep := e.Run(src, false)
			if len(rep.Diagnostics) != 0 {
				t.Fatalf("diagnostics = %v", rep.Diagnostics)
			}
			var want int
			if _, err := fmt.Sscanf(strings.SplitN(src, "\n", 2)[0], "-- statements: %d", &want); err != nil {
				t.Fatalf("first line must be '-- statements: N': %v", err)
			}
			if got := len(rep.AST.Children); got != want {
				t.Errorf("AST statements = %d, want %d", got, want)
			}
			if _, err := os.Stat(e.Dir); !os.IsNotExist(err) {
				t.Errorf("analysis touched disk: %v", err)
			}
		})
	}
}

// errors/*.sql must produce exactly the diagnostics of the .expected file
// (one "line:col: error phase: message" per line). "-- mode: execute" on the
// first line runs the script instead of only analysing it.
func TestExamplesReportLocatedErrors(t *testing.T) {
	for _, f := range exampleFiles(t, "errors") {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src := readExample(t, f)
			rep := newEngine(t).Run(src, strings.HasPrefix(src, "-- mode: execute"))
			if len(rep.Diagnostics) == 0 {
				t.Fatal("expected diagnostics, got none")
			}
			var b strings.Builder
			for _, d := range rep.Diagnostics {
				b.WriteString(d.String() + "\n")
			}
			compareFile(t, strings.TrimSuffix(f, ".sql")+".expected", b.String())
		})
	}
}

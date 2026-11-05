package sql

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func alterRun(t *testing.T, dir, src string) Report {
	t.Helper()
	rep := (&Engine{Dir: dir}).Run(src, true)
	if len(rep.Diagnostics) != 0 {
		t.Fatalf("%s: %v", src, rep.Diagnostics)
	}
	return rep
}

func alterFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if ent.IsDir() {
			out[ent.Name()] = "<directory>"
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[ent.Name()] = string(data)
	}
	return out
}

func TestAlterLifecycle(t *testing.T) {
	dir := t.TempDir()
	alterRun(t, dir, "CREATE TABLE Before (id INT PRIMARY KEY, name VARCHAR(30), spare INT); INSERT INTO Before VALUES (1, 'Ana', 8), (2, 'Luis', 9); CREATE INDEX keep ON Before(id); CREATE INDEX gone ON Before(spare);")
	for _, src := range []string{
		"ALTER TABLE before ADD COLUMN note VARCHAR(10);",
		"ALTER TABLE BEFORE DROP spare;",
		"ALTER TABLE before RENAME TO After;",
	} {
		rep := alterRun(t, dir, src)
		if len(rep.Results) != 1 || rep.Results[0].Kind != "ALTER TABLE" || rep.Results[0].RowsAffected != 0 {
			t.Fatalf("DDL result = %+v", rep.Results)
		}
		if rep.AST.Children[0].Name != "AlterTable" || !strings.Contains(treeText(rep.ParseTree), "alter_table") {
			t.Fatalf("ALTER missing from trees: %+v", rep)
		}
	}
	beforeQuery := alterFiles(t, dir)
	rep := alterRun(t, dir, "SELECT * FROM after WHERE id >= 1 ORDER BY id;")
	if !reflect.DeepEqual(beforeQuery, alterFiles(t, dir)) {
		t.Fatal("query had to repair an index after ALTER")
	}
	want := [][]any{{int64(1), "Ana", nil}, {int64(2), "Luis", nil}}
	if !reflect.DeepEqual(rep.Results[0].Rows, want) || !reflect.DeepEqual(rep.Results[0].Columns, []string{"id", "name", "note"}) {
		t.Fatalf("rows/columns = %+v", rep.Results[0])
	}
	if len(rep.Indexes) != 1 || rep.Indexes[0].Table != "After" || rep.Indexes[0].Name != "keep" {
		t.Fatalf("indexes = %+v", rep.Indexes)
	}
	if !strings.Contains(strings.Join(rep.Results[0].Plan, "\n"), "keep") {
		t.Fatalf("index not used: %v", rep.Results[0].Plan)
	}
	scan := (&Engine{Dir: dir, NoIndex: true}).Run("SELECT * FROM after WHERE id >= 1 ORDER BY id;", true)
	if !reflect.DeepEqual(scan.Results[0].Rows, want) {
		t.Fatalf("scan differs: %+v", scan)
	}
	files := alterFiles(t, dir)
	if _, ok := files["before.csv"]; ok {
		t.Fatal("old CSV remains")
	}
	if _, ok := files["gone.idx"]; ok {
		t.Fatal("dropped column index remains")
	}
	if !strings.Contains(files["keep.idx"], "#INDEX,keep,After,id,") {
		t.Fatal("index header not retargeted")
	}
	// Exercise old writes against the new schema, then physical table removal.
	writes := alterRun(t, dir, "UPDATE after SET note = 'ok' WHERE id = 1; DELETE FROM after WHERE id = 2; INSERT INTO after VALUES (3, 'Eva', NULL);")
	for _, result := range writes.Results {
		if result.RowsAffected != 1 {
			t.Fatalf("post-ALTER write counter = %+v", result)
		}
	}
	rep = alterRun(t, dir, "SELECT id, note FROM after ORDER BY id;")
	if !reflect.DeepEqual(rep.Results[0].Rows, [][]any{{int64(1), "ok"}, {int64(3), nil}}) {
		t.Fatalf("post-ALTER writes = %+v", rep)
	}
	alterRun(t, dir, "DROP TABLE after;")
	if got := alterFiles(t, dir); len(got) != 1 || got["metadata.txt"] == "" {
		t.Fatalf("DROP left files: %v", got)
	}
}

func treeText(n *TreeNode) string {
	if n == nil {
		return ""
	}
	s := n.Name
	for _, child := range n.Children {
		s += " " + treeText(child)
	}
	return s
}

func TestAlterShadowAndContextualNames(t *testing.T) {
	dir := t.TempDir()
	src := "CREATE TABLE alter (add INT); ALTER TABLE alter ADD column INT; ALTER TABLE alter ADD COLUMN rename DATE; ALTER TABLE alter DROP COLUMN rename; ALTER TABLE alter RENAME TO to; INSERT INTO to VALUES (1, 2); SELECT add, column FROM to;"
	before := alterFiles(t, dir)
	rep := (&Engine{Dir: dir}).Run(src, false)
	if len(rep.Diagnostics) != 0 || rep.Executed || len(rep.Results) != 0 || !reflect.DeepEqual(before, alterFiles(t, dir)) {
		t.Fatalf("analysis changed database or rejected shadow schema: %+v", rep)
	}
	rep = alterRun(t, dir, src)
	if !reflect.DeepEqual(rep.Results[len(rep.Results)-1].Rows, [][]any{{int64(1), int64(2)}}) {
		t.Fatalf("contextual names: %+v", rep)
	}
	alterRun(t, dir, "ALTER TABLE to DROP column; ALTER TABLE to RENAME TO TO;")
	// All new words remain valid in pre-existing identifier/alias positions.
	alterRun(t, dir, "CREATE TABLE rename (alter INT, add INT, column INT, to INT); INSERT INTO rename VALUES (1,2,3,4); SELECT alter AS rename, add AS column, column AS to FROM rename alter;")
}

func TestAlterRejectsWithoutMutation(t *testing.T) {
	tests := []struct {
		name, setup, statement, phase, message string
	}{
		{"missing table", "", "ALTER TABLE absent ADD x INT;", "semantic", `La tabla "absent" no existe`},
		{"duplicate column", "", "ALTER TABLE t ADD ID INT;", "semantic", `La columna "ID" está duplicada`},
		{"invalid length", "", "ALTER TABLE t ADD x VARCHAR(0);", "semantic", "debe ser mayor que 0"},
		{"second key", "", "ALTER TABLE t ADD x INT PRIMARY KEY;", "semantic", "más de una PRIMARY KEY"},
		{"missing column", "", "ALTER TABLE t DROP absent;", "semantic", `La columna "absent" no existe`},
		{"last column", "CREATE TABLE one (v INT);", "ALTER TABLE one DROP COLUMN v;", "semantic", "última columna"},
		{"rename collision", "CREATE TABLE other (v INT);", "ALTER TABLE t RENAME TO OTHER;", "semantic", `La tabla "OTHER" ya existe`},
		{"not null data", "INSERT INTO t VALUES (1, 'Ana');", "ALTER TABLE t ADD x INT NOT NULL;", "execution", "NOT NULL"},
		{"primary key data", "CREATE TABLE plain (v INT); INSERT INTO plain VALUES (1);", "ALTER TABLE plain ADD x INT PRIMARY KEY;", "execution", "NOT NULL"},
		{"default unsupported", "", "ALTER TABLE t ADD x INT DEFAULT 1;", "syntax", "Se esperaba ';'"},
		{"unknown action", "", "ALTER TABLE t MODIFY id INT;", "syntax", "ADD, DROP o RENAME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			alterRun(t, dir, "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(10)); CREATE INDEX ix ON t(id); "+tt.setup)
			before := alterFiles(t, dir)
			rep := (&Engine{Dir: dir}).Run(tt.statement+" DELETE FROM t;", true)
			if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != tt.phase || !strings.Contains(rep.Diagnostics[0].Message, tt.message) {
				t.Fatalf("diagnostics = %v, want %s / %s", rep.Diagnostics, tt.phase, tt.message)
			}
			if len(rep.Results) != 0 || !reflect.DeepEqual(before, alterFiles(t, dir)) {
				t.Fatalf("rejection changed files/counters: %+v", rep)
			}
		})
	}
}

func TestAlterEmptyConstraintsAndShadowFailure(t *testing.T) {
	dir := t.TempDir()
	rep := alterRun(t, dir, "CREATE TABLE t (v INT); ALTER TABLE t ADD id INT PRIMARY KEY; ALTER TABLE t ADD required VARCHAR(5) NOT NULL; INSERT INTO t VALUES (8, 1, 'ok'); SELECT * FROM t;")
	if !reflect.DeepEqual(rep.Results[4].Rows, [][]any{{int64(8), int64(1), "ok"}}) {
		t.Fatalf("empty constrained addition = %+v", rep)
	}
	before := alterFiles(t, dir)
	rep = (&Engine{Dir: dir}).Run("ALTER TABLE t DROP v; SELECT v FROM t;", true)
	if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "semantic" || rep.Executed || !reflect.DeepEqual(before, alterFiles(t, dir)) {
		t.Fatalf("invalid later statement changed files: %+v", rep)
	}
}

func TestAlterValidatesStoredConstraints(t *testing.T) {
	for _, data := range []string{"id,name\n1,too-long-value\n", "id,name\n\\N,Ana\n", "id,name\n1,Ana\n1,Eva\n"} {
		t.Run(data, func(t *testing.T) {
			dir := t.TempDir()
			alterRun(t, dir, "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(10));")
			if err := os.WriteFile(filepath.Join(dir, "t.csv"), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			before := alterFiles(t, dir)
			rep := (&Engine{Dir: dir}).Run("ALTER TABLE t ADD note INT;", true)
			if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "execution" || len(rep.Results) != 0 || !reflect.DeepEqual(before, alterFiles(t, dir)) {
				t.Fatalf("invalid data accepted or mutated: %+v", rep)
			}
		})
	}
}

func TestAlterFilesystemPreflight(t *testing.T) {
	for _, obstacle := range []string{"metadata.txt", "ix.idx", "new.csv", "t.csv"} {
		t.Run(obstacle, func(t *testing.T) {
			dir := t.TempDir()
			alterRun(t, dir, "CREATE TABLE t (id INT); INSERT INTO t VALUES (1); CREATE INDEX ix ON t(id);")
			// Keep original bytes in a sibling file; an occupied directory must
			// never be treated as an ordinary replaceable database file.
			target := filepath.Join(dir, obstacle)
			if _, err := os.Stat(target); err == nil {
				if err := os.Rename(target, target+".saved"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(target, 0700); err != nil {
				t.Fatal(err)
			}
			before := alterFiles(t, dir)
			rep := (&Engine{Dir: dir}).Run("ALTER TABLE t RENAME TO new;", true)
			if len(rep.Diagnostics) == 0 || rep.Diagnostics[0].Phase != "execution" || len(rep.Results) != 0 || !reflect.DeepEqual(before, alterFiles(t, dir)) {
				t.Fatalf("filesystem failure changed database: %+v", rep)
			}
		})
	}
}

func TestAlterPreservesUncatalogedRenameTarget(t *testing.T) {
	dir := t.TempDir()
	alterRun(t, dir, "CREATE TABLE t (id INT);")
	if err := os.WriteFile(filepath.Join(dir, "new.csv"), []byte("unrelated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := alterFiles(t, dir)
	rep := (&Engine{Dir: dir}).Run("ALTER TABLE t RENAME TO new;", true)
	if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "execution" || !strings.Contains(rep.Diagnostics[0].Message, "ya existe") || !reflect.DeepEqual(before, alterFiles(t, dir)) {
		t.Fatalf("occupied target overwritten: %+v", rep)
	}
}

func TestAlterContextualWordsRequireIdentifiers(t *testing.T) {
	for _, statement := range []string{
		"'ALTER' TABLE t ADD x INT;",
		"ALTER TABLE t 'ADD' x INT;",
		"ALTER TABLE t 'RENAME' TO other;",
		"ALTER TABLE t RENAME 'TO' other;",
	} {
		t.Run(statement, func(t *testing.T) {
			dir := t.TempDir()
			alterRun(t, dir, "CREATE TABLE t (id INT);")
			before := alterFiles(t, dir)
			rep := (&Engine{Dir: dir}).Run(statement, true)
			if len(rep.Diagnostics) == 0 || rep.Diagnostics[0].Phase != "syntax" || rep.Executed || !reflect.DeepEqual(before, alterFiles(t, dir)) {
				t.Fatalf("string accepted as syntax: %+v", rep)
			}
		})
	}
}

func TestAlterRewritesEscapedDataAndKeepsCatalogOrder(t *testing.T) {
	dir := t.TempDir()
	alterRun(t, dir, "CREATE TABLE first (v INT); CREATE TABLE middle (id INT, text VARCHAR(20), spare INT); CREATE TABLE last (v INT); INSERT INTO middle VALUES (1, '', 8), (2, '\\N', 9), (3, 'a,b', 10), (4, 'line\nbreak', 11); CREATE INDEX unrelated ON first(v); CREATE INDEX retained ON middle(text); CREATE INDEX removed ON middle(id);")
	for _, statement := range []string{
		"ALTER TABLE middle ADD optional DATE;",
		"ALTER TABLE middle DROP COLUMN id;",
		"ALTER TABLE middle RENAME TO renamed;",
	} {
		rep := alterRun(t, dir, statement)
		if len(rep.Schema) != 3 || rep.Schema[0].Name != "first" || rep.Schema[2].Name != "last" {
			t.Fatalf("catalog order changed: %+v", rep.Schema)
		}
		// The index must already match the CSV before any indexed query.
		name := "middle"
		if strings.Contains(statement, "RENAME") {
			name = "renamed"
		}
		before := alterFiles(t, dir)
		indexed := alterRun(t, dir, "SELECT text FROM "+name+" WHERE text >= '' ORDER BY text;")
		scan := (&Engine{Dir: dir, NoIndex: true}).Run("SELECT text FROM "+name+" WHERE text >= '' ORDER BY text;", true)
		if len(scan.Diagnostics) != 0 || !reflect.DeepEqual(indexed.Results[0].Rows, scan.Results[0].Rows) || !reflect.DeepEqual(before, alterFiles(t, dir)) {
			t.Fatalf("index not rebuilt correctly: %+v / %+v", indexed, scan)
		}
	}
	rep := alterRun(t, dir, "SELECT * FROM renamed ORDER BY spare;")
	want := [][]any{{"", int64(8), nil}, {"\\N", int64(9), nil}, {"a,b", int64(10), nil}, {"line\nbreak", int64(11), nil}}
	if !reflect.DeepEqual(rep.Results[0].Rows, want) {
		t.Fatalf("escaped values changed: %v", rep.Results[0].Rows)
	}
	if len(rep.Indexes) != 2 || rep.Indexes[0].Name != "unrelated" || rep.Indexes[1].Name != "retained" {
		t.Fatalf("index ordering changed: %v", rep.Indexes)
	}
	if !strings.HasPrefix(alterFiles(t, dir)["renamed.csv"], "text,spare,optional\n") {
		t.Fatal("CSV header not changed")
	}
}

func TestAlterAnalysisPreservesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	alterRun(t, dir, "CREATE TABLE t (id INT); INSERT INTO t VALUES (1); CREATE INDEX ix ON t(id);")
	before := alterFiles(t, dir)
	rep := (&Engine{Dir: dir}).Run("ALTER TABLE t ADD x INT NOT NULL; ALTER TABLE t RENAME TO other; SELECT x FROM other;", false)
	if len(rep.Diagnostics) != 0 || rep.Executed || len(rep.Results) != 0 || !reflect.DeepEqual(before, alterFiles(t, dir)) {
		t.Fatalf("analysis mutated data or checked runtime constraints: %+v", rep)
	}
}

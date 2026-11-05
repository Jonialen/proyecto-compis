package sql

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const leftJoinData = `
CREATE TABLE a (id INT PRIMARY KEY);
CREATE TABLE b (id INT NOT NULL, aid INT, value INT);
CREATE TABLE c (id INT PRIMARY KEY);
CREATE TABLE empty (id INT);
INSERT INTO a VALUES (1), (2), (3);
INSERT INTO b VALUES (10, 1, 5), (11, 1, NULL), (20, 2, 9), (30, NULL, 5);
INSERT INTO c VALUES (10), (20);
CREATE INDEX ai ON a(id);
CREATE INDEX bv ON b(value);
`

func TestLeftJoinSemantics(t *testing.T) {
	cases := []struct {
		name, query string
		want        []string
	}{
		{"matches and extension", "SELECT a.id, b.id, b.value FROM a LEFT JOIN b ON a.id = b.aid;", []string{"1|10|5", "1|11|NULL", "2|20|9", "3|NULL|NULL"}},
		{"outer star", "SELECT * FROM a LEFT OUTER JOIN empty ON a.id = empty.id;", []string{"1|NULL", "2|NULL", "3|NULL"}},
		{"no matches", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.id;", []string{"1|NULL", "2|NULL", "3|NULL"}},
		{"empty left", "SELECT * FROM empty LEFT JOIN a ON empty.id = a.id;", nil},
		{"on filters matches", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.aid AND b.value = 5;", []string{"1|10", "2|NULL", "3|NULL"}},
		{"where filters result", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.aid WHERE b.value = 5;", []string{"1|10"}},
		{"where must not fabricate nulls", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.aid WHERE b.value = 5 AND b.id IS NULL;", nil},
		{"where disjunction", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.aid WHERE b.value = 5 OR b.id IS NULL;", []string{"1|10", "3|NULL"}},
		{"chained left", "SELECT a.id, b.id, c.id FROM a LEFT JOIN b ON a.id = b.aid LEFT JOIN c ON b.id = c.id;", []string{"1|10|10", "1|11|NULL", "2|20|20", "3|NULL|NULL"}},
		{"left then inner", "SELECT a.id, c.id FROM a LEFT JOIN b ON a.id = b.aid JOIN c ON b.id = c.id;", []string{"1|10", "2|20"}},
		{"inner then left", "SELECT a.id, c.id FROM a JOIN b ON a.id = b.aid LEFT JOIN c ON b.id = c.id;", []string{"1|10", "1|NULL", "2|20"}},
		{"aggregate", "SELECT a.id, COUNT(*), COUNT(b.id), SUM(b.value) FROM a LEFT JOIN b ON a.id = b.aid GROUP BY a.id HAVING COUNT(*) >= 1 ORDER BY a.id;", []string{"1|2|2|5", "2|1|1|9", "3|1|0|NULL"}},
		{"indexed preserved side", "SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.aid WHERE a.id >= 2;", []string{"2|20", "3|NULL"}},
		{"standard left precedence", "SELECT a.id, b.id FROM a left JOIN b ON a.id = b.aid;", []string{"1|10", "1|11", "2|20", "3|NULL"}},
		{"explicit left alias retains inner join", "SELECT left.id, b.id FROM a AS left JOIN b ON left.id = b.aid;", []string{"1|10", "1|11", "2|20"}},
		{"inner compatibility", "SELECT a.id, b.id FROM a INNER JOIN b ON a.id = b.aid;", []string{"1|10", "1|11", "2|20"}},
		{"join compatibility", "SELECT a.id, b.id FROM a JOIN b ON a.id = b.aid;", []string{"1|10", "1|11", "2|20"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEngine(t)
			mustRun(t, e, leftJoinData)
			before := leftJoinFiles(t, e.Dir)
			indexed := expectRows(t, e, tc.query, tc.want...)
			if indexed.RowsAffected != len(tc.want) {
				t.Errorf("rowsAffected = %d", indexed.RowsAffected)
			}
			e.NoIndex = true
			scanned := query(t, e, tc.query)
			if !reflect.DeepEqual(indexed.Rows, scanned.Rows) || !reflect.DeepEqual(indexed.Columns, scanned.Columns) {
				t.Errorf("index/scan differ: %+v / %+v", indexed, scanned)
			}
			if !reflect.DeepEqual(before, leftJoinFiles(t, e.Dir)) {
				t.Fatal("SELECT changed persisted files")
			}
		})
	}
}

func leftJoinFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = string(data)
	}
	return files
}

func TestLeftJoinContextualIdentifiersAndAST(t *testing.T) {
	e := newEngine(t)
	mustRun(t, e, "CREATE TABLE left (outer INT); CREATE TABLE outer (left INT); INSERT INTO left VALUES (1); INSERT INTO outer VALUES (1);")
	expectRows(t, e, "SELECT left.outer, outer.left FROM left left JOIN outer outer ON left.outer = outer.left;", "1|1")
	expectRows(t, e, "SELECT left.outer, outer.left FROM left left LEFT OUTER JOIN outer outer ON left.outer = outer.left;", "1|1")
	expectRows(t, e, "SELECT outer AS left FROM left;", "1")
	rep := mustRun(t, e, "SELECT * FROM left LEFT JOIN outer ON left.outer = outer.left;")
	if rep.AST.Children[0].Children[2].Name != "LeftJoin" {
		t.Fatalf("AST = %+v", rep.AST.Children[0].Children[2])
	}
	rep = mustRun(t, e, "SELECT * FROM left INNER JOIN outer ON left.outer = outer.left;")
	if rep.AST.Children[0].Children[2].Name != "InnerJoin" {
		t.Fatal("INNER JOIN AST changed")
	}
}

func TestLeftJoinRejectedStatementsPreserveState(t *testing.T) {
	e := newEngine(t)
	mustRun(t, e, leftJoinData)
	before := leftJoinFiles(t, e.Dir)
	for _, tc := range []struct{ source, phase, message string }{
		{"SELECT * FROM a LEFT JOIN b;", "syntax", "'ON'"},
		{"SELECT * FROM a LEFT OUTER JOIN b ON a.id = b.missing;", "semantic", `La columna "missing" no existe`},
		{"SELECT * FROM a LEFT JOIN b ON 1;", "semantic", "BOOLEAN"},
	} {
		rep := e.Run(tc.source, true)
		if rep.Executed || len(rep.Results) != 0 || len(rep.Diagnostics) != 1 {
			t.Fatalf("unexpected report: %+v", rep)
		}
		d := rep.Diagnostics[0]
		if d.Phase != tc.phase || !strings.Contains(d.Message, tc.message) || d.Line != 1 || d.Col < 1 {
			t.Errorf("diagnostic = %+v", d)
		}
		if !reflect.DeepEqual(before, leftJoinFiles(t, e.Dir)) {
			t.Fatal("rejected statement changed files")
		}
	}
	reopened := &Engine{Dir: e.Dir}
	expectRows(t, reopened, "SELECT COUNT(*) FROM a;", "3")
	expectRows(t, reopened, "SELECT COUNT(*) FROM b;", "4")
}

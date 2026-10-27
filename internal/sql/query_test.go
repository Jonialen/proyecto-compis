package sql

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The data set below is small enough to compute every expected result by hand.
const school = `
CREATE TABLE estudiantes (id INT PRIMARY KEY, nombre VARCHAR(10) NOT NULL, edad INT, promedio FLOAT, activo BOOLEAN);
CREATE TABLE cursos (id INT PRIMARY KEY, nombre VARCHAR(20) NOT NULL, creditos INT NOT NULL);
CREATE TABLE inscripciones (estudiante_id INT NOT NULL, curso_id INT NOT NULL, nota FLOAT);
INSERT INTO estudiantes VALUES (1, 'Ana', 20, 88.5, TRUE), (2, 'Luis', 17, 71, TRUE), (3, 'Marta', 22, NULL, FALSE), (4, 'Zoe', 21, 95.25, TRUE), (5, 'Pedro', 18, 90, NULL);
INSERT INTO cursos VALUES (1, 'Compiladores', 5), (2, 'Bases', 4), (3, 'Algebra', 3), (4, 'Fisica', 6);
INSERT INTO inscripciones VALUES (1, 1, 90), (2, 1, NULL), (3, 2, 70), (4, 1, 100), (4, 3, 80), (5, 4, 60), (1, 2, 85);
`

func schoolEngine(t *testing.T) *Engine {
	t.Helper()
	e := newEngine(t)
	mustRun(t, e, school)
	return e
}

func mustRun(t *testing.T, e *Engine, src string) Report {
	t.Helper()
	rep := e.Run(src, true)
	if len(rep.Diagnostics) != 0 {
		t.Fatalf("%s\n  diagnostics: %v", src, rep.Diagnostics)
	}
	return rep
}

func rowsText(r StmtResult) []string {
	out := make([]string, len(r.Rows))
	for i, row := range r.Rows {
		cells := make([]string, len(row))
		for j, v := range row {
			cells[j] = FormatValue(v)
		}
		out[i] = strings.Join(cells, "|")
	}
	return out
}

func query(t *testing.T, e *Engine, q string) StmtResult {
	t.Helper()
	rep := mustRun(t, e, q)
	return rep.Results[len(rep.Results)-1]
}

func expectRows(t *testing.T, e *Engine, q string, want ...string) StmtResult {
	t.Helper()
	res := query(t, e, q)
	got := rowsText(res)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s\n  got  %q\n  want %q", q, got, want)
	}
	return res
}

func TestSelectWhereAndOrderWithNulls(t *testing.T) {
	e := schoolEngine(t)
	cases := []struct {
		q    string
		want []string
	}{
		{"SELECT nombre FROM estudiantes WHERE edad >= 18;", []string{"Ana", "Marta", "Zoe", "Pedro"}},
		{"SELECT nombre FROM estudiantes WHERE edad >= 18 AND activo = TRUE;", []string{"Ana", "Zoe"}},
		// Pedro: activo is NULL and 18 > 21 is false, so the OR is NULL and he is excluded.
		{"SELECT nombre FROM estudiantes WHERE activo = TRUE OR edad > 21;", []string{"Ana", "Luis", "Marta", "Zoe"}},
		{"SELECT nombre FROM estudiantes WHERE NOT activo;", []string{"Marta"}},
		{"SELECT nombre FROM estudiantes WHERE promedio IS NULL;", []string{"Marta"}},
		{"SELECT COUNT(*) FROM estudiantes WHERE promedio IS NOT NULL;", []string{"4"}},
		{"SELECT nombre FROM estudiantes ORDER BY promedio;", []string{"Marta", "Luis", "Ana", "Pedro", "Zoe"}},
		{"SELECT nombre FROM estudiantes ORDER BY promedio DESC;", []string{"Zoe", "Pedro", "Ana", "Luis", "Marta"}},
		{"SELECT nombre FROM estudiantes ORDER BY edad DESC, nombre;", []string{"Marta", "Zoe", "Ana", "Pedro", "Luis"}},
		{"SELECT nombre, promedio + 1 AS mas FROM estudiantes WHERE id = 1;", []string{"Ana|89.5"}},
		{"SELECT * FROM cursos WHERE creditos > 3 ORDER BY id;", []string{"1|Compiladores|5", "2|Bases|4", "4|Fisica|6"}},
		{"SELECT nombre FROM estudiantes WHERE edad * 2 - 1 > 36 AND nombre <> 'Zoe';", []string{"Ana", "Marta"}},
		{"SELECT nombre FROM estudiantes WHERE id IN (1, 3, 99);", []string{"Ana", "Marta"}},
		{"SELECT nombre FROM estudiantes WHERE id NOT IN (1, 2, 3);", []string{"Zoe", "Pedro"}},
		{"SELECT nombre FROM estudiantes ORDER BY nombre DESC;", []string{"Zoe", "Pedro", "Marta", "Luis", "Ana"}},
	}
	for _, c := range cases {
		expectRows(t, e, c.q, c.want...)
	}
	res := query(t, e, "SELECT nombre, promedio + 1 AS mas FROM estudiantes WHERE id = 1;")
	if strings.Join(res.Columns, ",") != "nombre,mas" {
		t.Errorf("columns = %v", res.Columns)
	}
	res = query(t, e, "SELECT COUNT(*), AVG(promedio), e.nombre FROM estudiantes e GROUP BY e.nombre;")
	if strings.Join(res.Columns, ",") != "COUNT(*),AVG(promedio),e.nombre" {
		t.Errorf("default column names = %v", res.Columns)
	}
}

func TestJoinsAndAggregates(t *testing.T) {
	e := schoolEngine(t)
	cases := []struct {
		q    string
		want []string
	}{
		{`SELECT e.nombre, c.nombre FROM estudiantes e INNER JOIN inscripciones i ON i.estudiante_id = e.id
		  INNER JOIN cursos c ON c.id = i.curso_id WHERE i.nota >= 85 ORDER BY e.nombre, c.nombre;`,
			[]string{"Ana|Bases", "Ana|Compiladores", "Zoe|Compiladores"}},
		{"SELECT COUNT(*), COUNT(promedio), SUM(edad), AVG(promedio), MIN(nombre), MAX(promedio) FROM estudiantes;",
			[]string{"5|4|98|86.1875|Ana|95.25"}},
		{"SELECT curso_id, COUNT(*) AS n, AVG(nota) FROM inscripciones GROUP BY curso_id ORDER BY curso_id;",
			[]string{"1|3|95", "2|2|77.5", "3|1|80", "4|1|60"}},
		{"SELECT curso_id FROM inscripciones GROUP BY curso_id HAVING COUNT(*) >= 2 ORDER BY curso_id;", []string{"1", "2"}},
		{"SELECT COUNT(*), SUM(edad), AVG(edad), MIN(edad), MAX(edad) FROM estudiantes WHERE edad > 100;", []string{"0|NULL|NULL|NULL|NULL"}},
		{"SELECT curso_id FROM inscripciones GROUP BY curso_id ORDER BY COUNT(*) DESC, curso_id;", []string{"1", "2", "3", "4"}},
		{`SELECT c.nombre, COUNT(*) FROM cursos c INNER JOIN inscripciones i ON i.curso_id = c.id
		  GROUP BY c.nombre ORDER BY c.nombre;`, []string{"Algebra|1", "Bases|2", "Compiladores|3", "Fisica|1"}},
		{"SELECT activo, COUNT(*) FROM estudiantes GROUP BY activo ORDER BY activo;", []string{"NULL|1", "FALSE|1", "TRUE|3"}},
		{"SELECT SUM(creditos), MAX(creditos) FROM cursos;", []string{"18|6"}},
		{"SELECT COUNT(*) FROM estudiantes e INNER JOIN inscripciones i ON i.estudiante_id = e.id;", []string{"7"}},
		{"SELECT c.nombre, SUM(i.nota) FROM cursos c INNER JOIN inscripciones i ON i.curso_id = c.id GROUP BY c.nombre HAVING SUM(i.nota) > 100 ORDER BY SUM(i.nota) DESC;",
			[]string{"Compiladores|190", "Bases|155"}},
		{"SELECT e.nombre, i.nota FROM estudiantes e INNER JOIN inscripciones i ON e.id = i.estudiante_id AND i.nota > 85 ORDER BY e.nombre;",
			[]string{"Ana|90", "Zoe|100"}},
	}
	for _, c := range cases {
		expectRows(t, e, c.q, c.want...)
	}
}

func TestSubqueries(t *testing.T) {
	e := schoolEngine(t)
	cases := []struct {
		q    string
		want []string
	}{
		{"SELECT nombre FROM estudiantes WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 3);", []string{"Zoe"}},
		{"SELECT nombre FROM estudiantes WHERE id NOT IN (SELECT estudiante_id FROM inscripciones WHERE curso_id = 1);", []string{"Marta", "Pedro"}},
		// The subquery returns a NULL, so NOT IN can never be true.
		{"SELECT nombre FROM estudiantes WHERE id NOT IN (SELECT nota FROM inscripciones);", nil},
		{"SELECT nombre FROM estudiantes e WHERE EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id AND i.nota > 85);", []string{"Ana", "Zoe"}},
		{"SELECT nombre FROM estudiantes e WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = e.id AND i.nota > 85);", []string{"Luis", "Marta", "Pedro"}},
		{"SELECT nombre FROM estudiantes WHERE promedio > (SELECT AVG(promedio) FROM estudiantes);", []string{"Ana", "Zoe", "Pedro"}},
		{"SELECT nombre FROM estudiantes WHERE edad = (SELECT edad FROM estudiantes WHERE id = 99);", nil},
		{"SELECT nombre, (SELECT COUNT(*) FROM inscripciones i WHERE i.estudiante_id = e.id) AS n FROM estudiantes e ORDER BY id;",
			[]string{"Ana|2", "Luis|1", "Marta|1", "Zoe|2", "Pedro|1"}},
		{`SELECT nombre FROM estudiantes WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE curso_id IN
		  (SELECT id FROM cursos WHERE creditos > (SELECT MIN(creditos) FROM cursos)));`, []string{"Ana", "Luis", "Marta", "Zoe", "Pedro"}},
		{"SELECT nombre FROM cursos c WHERE (SELECT COUNT(*) FROM inscripciones i WHERE i.curso_id = c.id) >= 2 ORDER BY nombre;", []string{"Bases", "Compiladores"}},
		{"SELECT nombre FROM estudiantes WHERE id IN (SELECT 1 FROM cursos WHERE id = 99);", nil},
		{"SELECT nombre FROM estudiantes WHERE id NOT IN (SELECT id FROM cursos WHERE id = 99);", []string{"Ana", "Luis", "Marta", "Zoe", "Pedro"}},
	}
	for _, c := range cases {
		expectRows(t, e, c.q, c.want...)
	}
}

func TestRuntimeErrorsAreLocatedAndStopTheScript(t *testing.T) {
	e := schoolEngine(t)
	cases := []struct{ q, want string }{
		{"SELECT nombre FROM estudiantes WHERE edad = (SELECT edad FROM estudiantes);", "retornó 5 registros"},
		{"SELECT 10 / (edad - 20) FROM estudiantes;", "División por cero"},
		{"UPDATE estudiantes SET id = 1 WHERE id = 2;", "PRIMARY KEY"},
		{"UPDATE estudiantes SET nombre = 'demasiado largo para la columna';", "excede la longitud"},
		{"UPDATE cursos SET creditos = creditos * 9223372036854775807;", "Desbordamiento"},
	}
	for _, c := range cases {
		rep := e.Run(c.q, true)
		if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != "execution" || !strings.Contains(rep.Diagnostics[0].Message, c.want) || rep.Diagnostics[0].Line == 0 {
			t.Errorf("%s\n  diagnostics = %+v, want execution error containing %q", c.q, rep.Diagnostics, c.want)
		}
	}
	// None of the failed statements may have changed the data.
	expectRows(t, e, "SELECT id, nombre, creditos FROM cursos WHERE id = 1;", "1|Compiladores|5")
	expectRows(t, e, "SELECT COUNT(*), MAX(nombre) FROM estudiantes;", "5|Zoe")
}

func TestUpdateAndDelete(t *testing.T) {
	e := schoolEngine(t)
	rep := mustRun(t, e, "UPDATE estudiantes SET promedio = promedio + 5 WHERE edad < 18;")
	if rep.Results[0].RowsAffected != 1 {
		t.Errorf("affected = %d", rep.Results[0].RowsAffected)
	}
	expectRows(t, e, "SELECT promedio FROM estudiantes WHERE id = 2;", "76")

	// SET expressions read the old row: this swaps the two columns.
	mustRun(t, e, "CREATE TABLE sw (a INT, b INT); INSERT INTO sw VALUES (1, 2);")
	mustRun(t, e, "UPDATE sw SET a = b, b = a;")
	expectRows(t, e, "SELECT a, b FROM sw;", "2|1")

	rep = mustRun(t, e, "UPDATE estudiantes SET activo = TRUE WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE nota >= 100);")
	if rep.Results[0].RowsAffected != 1 {
		t.Errorf("affected = %d", rep.Results[0].RowsAffected)
	}
	rep = mustRun(t, e, "UPDATE estudiantes SET edad = edad + 1;")
	if rep.Results[0].RowsAffected != 5 {
		t.Errorf("affected = %d, want 5", rep.Results[0].RowsAffected)
	}
	expectRows(t, e, "SELECT edad FROM estudiantes ORDER BY id;", "21", "18", "23", "22", "19")
	rep = mustRun(t, e, "UPDATE estudiantes SET edad = 0 WHERE id = 99;")
	if rep.Results[0].RowsAffected != 0 {
		t.Errorf("affected = %d, want 0", rep.Results[0].RowsAffected)
	}

	rep = mustRun(t, e, "DELETE FROM inscripciones WHERE nota IS NULL;")
	if rep.Results[0].RowsAffected != 1 {
		t.Errorf("deleted = %d", rep.Results[0].RowsAffected)
	}
	rep = mustRun(t, e, "DELETE FROM estudiantes WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = estudiantes.id);")
	if rep.Results[0].RowsAffected != 1 { // Luis: his only enrollment had the NULL grade
		t.Errorf("deleted = %d, want 1", rep.Results[0].RowsAffected)
	}
	mustRun(t, e, "DELETE FROM inscripciones WHERE estudiante_id = 3;")
	rep = mustRun(t, e, "DELETE FROM estudiantes WHERE NOT EXISTS (SELECT 1 FROM inscripciones i WHERE i.estudiante_id = estudiantes.id);")
	if rep.Results[0].RowsAffected != 1 { // Marta; Luis is already gone
		t.Errorf("deleted = %d, want 1", rep.Results[0].RowsAffected)
	}
	expectRows(t, e, "SELECT nombre FROM estudiantes ORDER BY id;", "Ana", "Zoe", "Pedro")
	mustRun(t, e, "DELETE FROM cursos;")
	expectRows(t, e, "SELECT COUNT(*) FROM cursos;", "0")
}

func TestOneColumnTableKeepsEmptyStrings(t *testing.T) {
	e := newEngine(t)
	mustRun(t, e, "CREATE TABLE s (v VARCHAR(5)); INSERT INTO s VALUES (''), ('a'), (NULL), ('');")
	expectRows(t, e, "SELECT COUNT(*), COUNT(v) FROM s;", "4|3")
	expectRows(t, e, "SELECT COUNT(*) FROM s WHERE v = '';", "2")
}

// --- indexes ---

func indexNote(res StmtResult) string {
	for _, p := range res.Plan {
		if strings.HasPrefix(p, "Índice") {
			return p
		}
	}
	return ""
}

func TestWhereUsesIndexInsteadOfScanningEveryRecord(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad); CREATE INDEX idx_nombre ON estudiantes (nombre);")

	res := expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21;", "Zoe")
	if got, want := indexNote(res), `Índice "idx_edad" usado en "estudiantes" por edad = 21: 1 de 5 registros leídos`; got != want {
		t.Errorf("plan = %q, want %q (all: %v)", got, want, res.Plan)
	}
	res = expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad >= 18 AND edad < 22;", "Ana", "Zoe", "Pedro")
	if got, want := indexNote(res), `Índice "idx_edad" usado en "estudiantes" por edad >= 18 AND edad < 22: 3 de 5 registros leídos`; got != want {
		t.Errorf("plan = %q, want %q", got, want)
	}
	res = expectRows(t, e, "SELECT nombre FROM estudiantes WHERE 18 < edad;", "Ana", "Marta", "Zoe")
	if !strings.Contains(indexNote(res), "edad > 18") {
		t.Errorf("flipped comparison must use the index as 'edad > 18': %v", res.Plan)
	}
	res = expectRows(t, e, "SELECT id FROM estudiantes WHERE nombre = 'Zoe' AND activo = TRUE;", "4")
	if !strings.Contains(indexNote(res), `"idx_nombre"`) || !strings.Contains(indexNote(res), "1 de 5") {
		t.Errorf("string index not used: %v", res.Plan)
	}
	// FLOAT constant against an INT column.
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21.0;", "Zoe")
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21.5;")
	// Equality on an indexed column wins over a range on another one.
	res = query(t, e, "SELECT nombre FROM estudiantes WHERE edad > 0 AND nombre = 'Ana';")
	if !strings.Contains(indexNote(res), `"idx_nombre"`) {
		t.Errorf("equality index not preferred: %v", res.Plan)
	}
	// OR, NOT and conditions on other columns cannot use the index.
	for _, q := range []string{
		"SELECT nombre FROM estudiantes WHERE edad = 21 OR edad = 20;",
		"SELECT nombre FROM estudiantes WHERE NOT edad = 21;",
		"SELECT nombre FROM estudiantes WHERE activo = TRUE;",
		"SELECT nombre FROM estudiantes WHERE edad IS NULL;",
		"SELECT nombre FROM estudiantes WHERE edad = NULL;",
	} {
		if n := indexNote(query(t, e, q)); n != "" {
			t.Errorf("%s must not use an index, got %q", q, n)
		}
	}
	res = query(t, e, "SELECT nombre FROM estudiantes WHERE activo = TRUE;")
	if len(res.Plan) != 1 || !strings.HasPrefix(res.Plan[0], "Recorrido completo") {
		t.Errorf("plan = %v", res.Plan)
	}
}

func TestIndexOnJoinedTableAndInSubqueries(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_ins ON inscripciones (estudiante_id);")
	res := expectRows(t, e, `SELECT c.nombre FROM estudiantes e INNER JOIN inscripciones i ON i.estudiante_id = e.id
		INNER JOIN cursos c ON c.id = i.curso_id WHERE i.estudiante_id = 4 ORDER BY c.nombre;`, "Algebra", "Compiladores")
	if !strings.Contains(indexNote(res), `usado en "inscripciones" por estudiante_id = 4: 2 de 7`) {
		t.Errorf("plan = %v", res.Plan)
	}
	res = expectRows(t, e, "SELECT nombre FROM estudiantes WHERE id IN (SELECT estudiante_id FROM inscripciones WHERE estudiante_id = 5);", "Pedro")
	if !strings.Contains(indexNote(res), `"idx_ins"`) {
		t.Errorf("subquery must use the index: %v", res.Plan)
	}
}

func TestUpdateAndDeleteUseIndexToChooseRecords(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad);")
	rep := mustRun(t, e, "UPDATE estudiantes SET promedio = 0 WHERE edad = 22;")
	if n := indexNote(rep.Results[0]); !strings.Contains(n, "1 de 5 registros evaluados") {
		t.Errorf("plan = %v", rep.Results[0].Plan)
	}
	expectRows(t, e, "SELECT promedio FROM estudiantes WHERE id = 3;", "0")
	rep = mustRun(t, e, "DELETE FROM estudiantes WHERE edad < 18;")
	if n := indexNote(rep.Results[0]); !strings.Contains(n, "1 de 5 registros evaluados") {
		t.Errorf("plan = %v", rep.Results[0].Plan)
	}
	expectRows(t, e, "SELECT COUNT(*) FROM estudiantes;", "4")
}

// indexMatchesTable rebuilds the expected content of every index from the
// table data and compares it with what the index file holds.
func indexMatchesTable(t *testing.T, e *Engine) {
	t.Helper()
	cat, err := LoadCatalog(e.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range cat.idxOrd {
		ix := cat.Indexes[key]
		tb := cat.Table(ix.Table)
		li, ok, err := readIndexFile(e.Dir, tb, ix)
		if err != nil || !ok {
			t.Fatalf("index %s unreadable or stale (ok=%v, err=%v)", ix.Name, ok, err)
		}
		rows, offs, err := readRowsOffsets(e.Dir, tb)
		if err != nil {
			t.Fatal(err)
		}
		ci := tb.columnIndex(ix.Column)
		want := map[string][]int64{}
		nonNull := 0
		for i, r := range rows {
			if r[ci] != nil {
				k := encodeCell(r[ci])
				want[k] = append(want[k], offs[i])
				nonNull++
			}
		}
		got := 0
		for i, en := range li.entries {
			if i > 0 && compareValues(li.entries[i-1].key, en.key) >= 0 {
				t.Errorf("index %s keys not strictly increasing at %d", ix.Name, i)
			}
			w := want[encodeCell(en.key)]
			if fmt.Sprint(w) != fmt.Sprint(en.offs) {
				t.Errorf("index %s key %v: offsets %v, want %v", ix.Name, en.key, en.offs, w)
			}
			got += len(en.offs)
		}
		if got != nonNull || len(li.entries) != len(want) || li.total != len(rows) {
			t.Errorf("index %s: %d offsets / %d keys / total %d, want %d / %d / %d", ix.Name, got, len(li.entries), li.total, nonNull, len(want), len(rows))
		}
	}
}

func TestIndexesAreMaintainedByInsertUpdateDelete(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad); CREATE INDEX idx_prom ON estudiantes (promedio); CREATE INDEX idx_nom ON estudiantes (nombre);")
	indexMatchesTable(t, e)

	mustRun(t, e, "INSERT INTO estudiantes VALUES (6, 'Nora', 21, 50.5, FALSE), (7, 'Sin edad', NULL, NULL, NULL);")
	indexMatchesTable(t, e)
	res := expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21;", "Zoe", "Nora")
	if !strings.Contains(indexNote(res), "2 de 7") {
		t.Errorf("plan = %v", res.Plan)
	}

	mustRun(t, e, "UPDATE estudiantes SET edad = 30, nombre = 'Zoe Z' WHERE id = 4;")
	indexMatchesTable(t, e)
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21;", "Nora")
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 30;", "Zoe Z")
	expectRows(t, e, "SELECT id FROM estudiantes WHERE nombre = 'Zoe';")

	mustRun(t, e, "DELETE FROM estudiantes WHERE edad >= 21;")
	indexMatchesTable(t, e)
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21;")
	expectRows(t, e, "SELECT COUNT(*) FROM estudiantes;", "4")

	// A rejected statement leaves table and indexes as they were.
	before := dumpDB(t, e.Dir)
	rep := e.Run("UPDATE estudiantes SET id = 1 WHERE id = 2;", true)
	if len(rep.Diagnostics) != 1 || dumpDB(t, e.Dir) != before {
		t.Errorf("a failed UPDATE must not change any file (diagnostics %v)", rep.Diagnostics)
	}
}

func TestStaleOrMissingIndexIsRebuiltInsteadOfTrusted(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad);")

	// Edit the table behind the engine's back.
	f := filepath.Join(e.Dir, "estudiantes.csv")
	data, _ := os.ReadFile(f)
	if err := os.WriteFile(f, append(data, []byte("6,Extra,21,1,TRUE\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21 ORDER BY id;", "Zoe", "Extra")
	indexMatchesTable(t, e)

	// Remove or damage the index file.
	idx := filepath.Join(e.Dir, "idx_edad.idx")
	os.Remove(idx)
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21 ORDER BY id;", "Zoe", "Extra")
	indexMatchesTable(t, e)
	os.WriteFile(idx, []byte("basura"), 0o644)
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad = 20;", "Ana")
	indexMatchesTable(t, e)
}

func TestDropIndexAndDropTableRemoveIndexFiles(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX i1 ON estudiantes (edad); CREATE INDEX i2 ON cursos (creditos);")
	for _, f := range []string{"i1.idx", "i2.idx"} {
		if _, err := os.Stat(filepath.Join(e.Dir, f)); err != nil {
			t.Fatalf("%s missing: %v", f, err)
		}
	}
	mustRun(t, e, "DROP INDEX i1;")
	if _, err := os.Stat(filepath.Join(e.Dir, "i1.idx")); !os.IsNotExist(err) {
		t.Errorf("i1.idx must be deleted: %v", err)
	}
	if n := indexNote(query(t, e, "SELECT nombre FROM estudiantes WHERE edad = 21;")); n != "" {
		t.Errorf("dropped index still used: %q", n)
	}
	mustRun(t, e, "DROP TABLE cursos;")
	if _, err := os.Stat(filepath.Join(e.Dir, "i2.idx")); !os.IsNotExist(err) {
		t.Errorf("i2.idx must be deleted with its table: %v", err)
	}
	cat, _ := LoadCatalog(e.Dir)
	if len(cat.Indexes) != 0 {
		t.Errorf("catalog indexes = %v", cat.Indexes)
	}
	// The same index name can be created again afterwards.
	mustRun(t, e, "CREATE INDEX i1 ON estudiantes (nombre);")
	indexMatchesTable(t, e)
}

func TestIndexFileFormat(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad);")
	data, err := os.ReadFile(filepath.Join(e.Dir, "idx_edad.idx"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 6 || !strings.HasPrefix(lines[0], "#INDEX,idx_edad,estudiantes,edad,") || !strings.HasSuffix(lines[0], ",5") {
		t.Errorf("index file:\n%s", data)
	}
	keys := []string{}
	for _, l := range lines[1:] {
		keys = append(keys, strings.SplitN(l, ",", 2)[0])
	}
	if strings.Join(keys, " ") != "17 18 20 21 22" {
		t.Errorf("keys = %v, want sorted numerically", keys)
	}
	meta, _ := os.ReadFile(filepath.Join(e.Dir, "metadata.txt"))
	if !strings.Contains(string(meta), "INDEX idx_edad ON estudiantes(edad)") {
		t.Errorf("metadata.txt lacks the index:\n%s", meta)
	}
}

// Random tables and conditions: with indexes the answer must equal a full scan,
// and after random writes every index must still match its table.
func TestIndexedResultsEqualFullScanOnRandomData(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	e := newEngine(t)
	mustRun(t, e, "CREATE TABLE r (id INT PRIMARY KEY, a INT, b VARCHAR(4), c FLOAT, d DATE);")
	words := []string{"ab", "b", "cd", "z", "", "ab9"}
	var ins strings.Builder
	ins.WriteString("INSERT INTO r VALUES ")
	for i := 1; i <= 150; i++ {
		if i > 1 {
			ins.WriteString(", ")
		}
		val := func(s string, null int) string {
			if rng.Intn(null) == 0 {
				return "NULL"
			}
			return s
		}
		fmt.Fprintf(&ins, "(%d, %s, %s, %s, %s)", i,
			val(fmt.Sprint(rng.Intn(20)-5), 8),
			val("'"+words[rng.Intn(len(words))]+"'", 8),
			val(fmt.Sprintf("%.1f", float64(rng.Intn(100))/4), 8),
			val(fmt.Sprintf("'2024-%02d-%02d'", 1+rng.Intn(12), 1+rng.Intn(28)), 8))
	}
	mustRun(t, e, ins.String()+";")
	mustRun(t, e, "CREATE INDEX ia ON r (a); CREATE INDEX ib ON r (b); CREATE INDEX ic ON r (c); CREATE INDEX id_ ON r (d);")
	plain := &Engine{Dir: e.Dir, NoIndex: true}

	ops := []string{"=", "<", "<=", ">", ">="}
	constFor := map[string]func() string{
		"a": func() string { return fmt.Sprint(rng.Intn(24) - 7) },
		"b": func() string { return "'" + words[rng.Intn(len(words))] + "'" },
		"c": func() string { return fmt.Sprintf("%.2f", float64(rng.Intn(110))/4) },
		"d": func() string { return fmt.Sprintf("'2024-%02d-%02d'", 1+rng.Intn(12), 1+rng.Intn(28)) },
	}
	cols := []string{"a", "b", "c", "d"}
	used := 0
	for n := 0; n < 300; n++ {
		var conds []string
		for k := 0; k <= rng.Intn(3); k++ {
			col := cols[rng.Intn(len(cols))]
			cond := fmt.Sprintf("%s %s %s", col, ops[rng.Intn(len(ops))], constFor[col]())
			if rng.Intn(5) == 0 {
				cond = fmt.Sprintf("%s %s %s", constFor[col](), "<", col) // constant on the left
			}
			conds = append(conds, cond)
		}
		q := "SELECT id FROM r WHERE " + strings.Join(conds, " AND ") + ";"
		with := query(t, e, q)
		without := plain.Run(q, true).Results[0]
		if strings.Join(rowsText(with), ",") != strings.Join(rowsText(without), ",") {
			t.Fatalf("%s\n  indexed %v\n  scan    %v", q, rowsText(with), rowsText(without))
		}
		if indexNote(with) != "" {
			used++
		}
	}
	if used < 100 {
		t.Errorf("only %d of 300 queries used an index; the generator is not exercising it", used)
	}

	for n := 0; n < 40; n++ {
		var q string
		switch rng.Intn(3) {
		case 0:
			q = fmt.Sprintf("INSERT INTO r VALUES (%d, %d, 'ab', %d.5, NULL);", 1000+n, rng.Intn(20)-5, rng.Intn(20))
		case 1:
			q = fmt.Sprintf("UPDATE r SET a = a + %d, b = 'cd' WHERE a = %d;", rng.Intn(5), rng.Intn(20)-5)
		default:
			q = fmt.Sprintf("DELETE FROM r WHERE a >= %d AND a <= %d;", rng.Intn(20)-5, rng.Intn(20)-5)
		}
		mustRun(t, e, q)
		indexMatchesTable(t, e)
	}
}

func TestCRLFInsideStringsIsStoredAsLF(t *testing.T) {
	e := newEngine(t)
	mustRun(t, e, "CREATE TABLE s (v VARCHAR(20)); INSERT INTO s VALUES ('a\r\nb');")
	expectRows(t, e, "SELECT v FROM s;", "a\nb")
}

func TestInListUsesIndex(t *testing.T) {
	e := schoolEngine(t)
	mustRun(t, e, "CREATE INDEX idx_edad ON estudiantes (edad);")
	res := expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad IN (21, 17, 21, 99);", "Luis", "Zoe")
	if got, want := indexNote(res), `Índice "idx_edad" usado en "estudiantes" por edad IN (21, 17, 21, 99): 2 de 5 registros leídos`; got != want {
		t.Errorf("plan = %q, want %q", got, want)
	}
	// NULL in the list cannot make a row match; the other values still use the index.
	res = expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad IN (NULL, 22);", "Marta")
	if !strings.Contains(indexNote(res), "1 de 5") {
		t.Errorf("plan = %v", res.Plan)
	}
	// Combined with a range on the same column and with other conditions.
	res = expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad IN (17, 20, 21, 22) AND edad > 17 AND activo = TRUE;", "Ana", "Zoe")
	if !strings.Contains(indexNote(res), "3 de 5") {
		t.Errorf("plan = %v", res.Plan)
	}
	// NOT IN and IN over a non-constant list need the full scan.
	for _, q := range []string{
		"SELECT nombre FROM estudiantes WHERE edad NOT IN (21, 17);",
		"SELECT nombre FROM estudiantes WHERE edad IN (21, id);",
	} {
		if n := indexNote(query(t, e, q)); n != "" {
			t.Errorf("%s must not use an index, got %q", q, n)
		}
	}
	expectRows(t, e, "SELECT nombre FROM estudiantes WHERE edad NOT IN (21, 17);", "Ana", "Marta", "Pedro")
}

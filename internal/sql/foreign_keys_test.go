package sql

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fkSchema creates a parent table and a child table that references it.
const fkSchema = `CREATE TABLE clientes (id INT PRIMARY KEY, nombre VARCHAR(20));
CREATE TABLE pedidos (id INT PRIMARY KEY, cliente_id INT, FOREIGN KEY (cliente_id) REFERENCES clientes(id));`

func fkRun(t *testing.T, dir, src string) Report {
	t.Helper()
	rep := (&Engine{Dir: dir}).Run(src, true)
	if len(rep.Diagnostics) != 0 {
		t.Fatalf("%s: %v", src, rep.Diagnostics)
	}
	return rep
}

// fkReject runs src expecting exactly one diagnostic with the given phase and
// message, and no statement results.
func fkReject(t *testing.T, dir, src, phase, msg string) {
	t.Helper()
	rep := (&Engine{Dir: dir}).Run(src, true)
	if len(rep.Diagnostics) != 1 || rep.Diagnostics[0].Phase != phase || rep.Diagnostics[0].Message != msg {
		t.Fatalf("%s\ndiagnostics = %+v\nwant %s: %s", src, rep.Diagnostics, phase, msg)
	}
	if len(rep.Results) != 0 {
		t.Fatalf("%s: rejected statement produced results %+v", src, rep.Results)
	}
}

// fkRows reopens the database and returns the rows of a SELECT.
func fkRows(t *testing.T, dir, query string) [][]any {
	t.Helper()
	rep := fkRun(t, dir, query)
	return rep.Results[0].Rows
}

func fkTables(t *testing.T, dir string) []string {
	t.Helper()
	rep := (&Engine{Dir: dir}).Run("", false)
	var out []string
	for _, ti := range rep.Schema {
		out = append(out, ti.Name)
	}
	return out
}

func TestForeignKeyDropTableRestrict(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, fkSchema)
	// The child has no rows: the schema dependency alone blocks the drop.
	fkReject(t, dir, "DROP TABLE clientes;", "semantic",
		`No se puede eliminar la tabla "clientes": la FOREIGN KEY pedidos.cliente_id la referencia; elimine antes esa tabla o columna`)
	if got := fkTables(t, dir); !reflect.DeepEqual(got, []string{"clientes", "pedidos"}) {
		t.Fatalf("tables after rejected DROP = %v", got)
	}
	rep := fkRun(t, dir, "DROP TABLE pedidos; DROP TABLE clientes;")
	if len(rep.Results) != 2 || rep.Results[1].Message != `Tabla "clientes" eliminada` {
		t.Fatalf("results = %+v", rep.Results)
	}
	if got := fkTables(t, dir); got != nil {
		t.Fatalf("tables after cleanup = %v", got)
	}
}

func TestForeignKeyDeleteRestrict(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, fkSchema+`
INSERT INTO clientes VALUES (1, 'Ana'), (2, 'Luis'), (3, 'Eva');
INSERT INTO pedidos VALUES (10, 1), (11, NULL);`)
	want := [][]any{{int64(1), "Ana"}, {int64(2), "Luis"}, {int64(3), "Eva"}}
	// Every selected row is checked before writing: nothing is deleted.
	fkReject(t, dir, "DELETE FROM clientes WHERE id <= 2;", "execution",
		`Violación de FOREIGN KEY: no se puede eliminar de "clientes" porque el valor 1 de "id" es referenciado por pedidos.cliente_id; no se eliminó ninguna fila`)
	if got := fkRows(t, dir, "SELECT * FROM clientes ORDER BY id;"); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows after rejected DELETE = %v", got)
	}
	// Unreferenced rows can be deleted; the PRIMARY KEY alone does not block.
	rep := fkRun(t, dir, "DELETE FROM clientes WHERE id >= 2;")
	if rep.Results[0].RowsAffected != 2 {
		t.Fatalf("unreferenced delete = %+v", rep.Results)
	}
	// Removing the child rows first allows the parent delete.
	rep = fkRun(t, dir, "DELETE FROM pedidos WHERE cliente_id = 1; DELETE FROM clientes;")
	if rep.Results[0].RowsAffected != 1 || rep.Results[1].RowsAffected != 1 {
		t.Fatalf("cleanup = %+v", rep.Results)
	}
	if got := fkRows(t, dir, "SELECT * FROM pedidos;"); !reflect.DeepEqual(got, [][]any{{int64(11), nil}}) {
		t.Fatalf("pedidos = %v", got)
	}
}

func TestForeignKeyDeclarationsAndReload(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, `CREATE TABLE p (id INT PRIMARY KEY, other INT); INSERT INTO p VALUES (1, 7);`)
	for _, tc := range []struct{ sql, contains string }{
		{`CREATE TABLE c (x INT REFERENCES missing(id));`, "tabla referenciada"},
		{`CREATE TABLE c (x INT REFERENCES p(missing));`, "no existe"},
		{`CREATE TABLE c (x INT REFERENCES p(other));`, "PRIMARY KEY"},
		{`CREATE TABLE c (x VARCHAR(4) REFERENCES p(id));`, "Tipos incompatibles"},
		{`CREATE TABLE c (x INT REFERENCES p(id), FOREIGN KEY (x) REFERENCES p(id));`, "ya tiene"},
		{`CREATE TABLE c (x INT, FOREIGN KEY (missing) REFERENCES p(id));`, "no existe"},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			r := (&Engine{Dir: dir}).Run(tc.sql, true)
			if len(r.Diagnostics) != 1 || !strings.Contains(r.Diagnostics[0].Message, tc.contains) || len(r.Results) != 0 {
				t.Fatalf("report: %+v", r)
			}
			if got := fkTables(t, dir); !reflect.DeepEqual(got, []string{"p"}) {
				t.Fatalf("schema: %v", got)
			}
		})
	}
	fkRun(t, dir, `CREATE TABLE c (id INT PRIMARY KEY, x INT REFERENCES p(id), y INT NOT NULL REFERENCES p(id)); INSERT INTO c VALUES (1, NULL, 1);`)
	fkReject(t, dir, `INSERT INTO c VALUES (2, NULL, NULL);`, "semantic", `La columna "y" no admite NULL`)
	fkReject(t, dir, `INSERT INTO c VALUES (2, 99, 1);`, "execution", `Violación de FOREIGN KEY: el valor 99 de c.x no existe en p.id; no se insertó ninguna fila`)
	if got := fkRows(t, dir, `SELECT * FROM c;`); !reflect.DeepEqual(got, [][]any{{int64(1), nil, int64(1)}}) {
		t.Fatalf("rows: %v", got)
	}
}

func TestForeignKeyAlterAndSelfReference(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, `CREATE TABLE p (id INT PRIMARY KEY, spare INT); CREATE TABLE c (id INT PRIMARY KEY); ALTER TABLE c ADD pid INT REFERENCES p(id); INSERT INTO p VALUES (1, 9); INSERT INTO c VALUES (1, 1); CREATE INDEX pidx ON p(id); ALTER TABLE p RENAME TO parent;`)
	fkReject(t, dir, `ALTER TABLE parent DROP id;`, "semantic", `No se puede eliminar la columna "id" de la tabla "parent": la FOREIGN KEY c.pid la referencia; elimine antes esa tabla o columna`)
	fkReject(t, dir, `DELETE FROM parent WHERE id = 1;`, "execution", `Violación de FOREIGN KEY: no se puede eliminar de "parent" porque el valor 1 de "id" es referenciado por c.pid; no se eliminó ninguna fila`)
	r := fkRun(t, dir, `SELECT * FROM parent WHERE id = 1;`)
	if !reflect.DeepEqual(r.Results[0].Rows, [][]any{{int64(1), int64(9)}}) || len(r.Indexes) != 1 || !strings.Contains(strings.Join(r.Results[0].Plan, " "), "pidx") {
		t.Fatalf("index/state: %+v", r)
	}
	fkRun(t, dir, `ALTER TABLE c DROP pid; DROP TABLE parent; DROP TABLE c;`)
	fkRun(t, dir, `CREATE TABLE s (id INT PRIMARY KEY, pid INT REFERENCES s(id)); INSERT INTO s VALUES (1, NULL), (2, 1);`)
	fkReject(t, dir, `DELETE FROM s WHERE id = 1;`, "execution", `Violación de FOREIGN KEY: no se puede eliminar de "s" porque el valor 1 de "id" es referenciado por s.pid; no se eliminó ninguna fila`)
	r = fkRun(t, dir, `DELETE FROM s; DROP TABLE s;`)
	if len(r.Results) != 2 || r.Results[0].RowsAffected != 2 {
		t.Fatalf("self cleanup: %+v", r.Results)
	}
}

func TestForeignKeyLegacyIdentifiersAndMetadata(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, `CREATE TABLE foreign (references INT PRIMARY KEY, restrict INT, cascade INT); INSERT INTO foreign VALUES (1, 2, 3);`)
	b, err := os.ReadFile(filepath.Join(dir, "metadata.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Compiscript SQL metadata v1\nTABLE foreign\nCOLUMN references INT PRIMARY_KEY NOT_NULL\nCOLUMN restrict INT\nCOLUMN cascade INT\nEND\n"
	if string(b) != want {
		t.Fatalf("metadata: %q", b)
	}
	r := fkRun(t, dir, `SELECT * FROM foreign;`)
	j, err := json.Marshal(r.Schema)
	if err != nil || strings.Contains(string(j), "references\":{") {
		t.Fatalf("schema JSON: %s, %v", j, err)
	}
	if !reflect.DeepEqual(r.Results[0].Rows, [][]any{{int64(1), int64(2), int64(3)}}) {
		t.Fatal(r.Results)
	}
	fkRun(t, dir, `DELETE FROM foreign; DROP TABLE foreign;`)
}

func TestForeignKeyInsertUpdateIntegrity(t *testing.T) {
	dir := t.TempDir()
	fkRun(t, dir, fkSchema+`
INSERT INTO clientes VALUES (1, 'Ana'), (2, 'Luis');
INSERT INTO pedidos VALUES (10, 1), (11, NULL);`)
	clientes := [][]any{{int64(1), "Ana"}, {int64(2), "Luis"}}
	pedidos := [][]any{{int64(10), int64(1)}, {int64(11), nil}}
	unchanged := func() {
		t.Helper()
		if got := fkRows(t, dir, "SELECT * FROM clientes ORDER BY id;"); !reflect.DeepEqual(got, clientes) {
			t.Fatalf("clientes = %v", got)
		}
		if got := fkRows(t, dir, "SELECT * FROM pedidos ORDER BY id;"); !reflect.DeepEqual(got, pedidos) {
			t.Fatalf("pedidos = %v", got)
		}
	}
	for _, tc := range []struct{ name, src, msg string }{
		{"multi-row insert with one missing key", "INSERT INTO pedidos VALUES (12, 2), (13, 9);",
			`Violación de FOREIGN KEY: el valor 9 de pedidos.cliente_id no existe en clientes.id; no se insertó ninguna fila`},
		{"child update to missing key", "UPDATE pedidos SET cliente_id = 7 WHERE id = 11;",
			`Violación de FOREIGN KEY: el valor 7 de pedidos.cliente_id no existe en clientes.id; no se modificó ninguna fila`},
		{"referenced parent key update", "UPDATE clientes SET id = id + 10;",
			`Violación de FOREIGN KEY: no se puede cambiar el valor 1 de "id" en "clientes" porque es referenciado por pedidos.cliente_id; no se modificó ninguna fila`},
		{"existing primary key still unique", "INSERT INTO clientes VALUES (1, 'Otra');",
			`Violación de PRIMARY KEY: el valor 1 de la columna "id" ya existe en la tabla "clientes" (fila 1 del INSERT); no se insertó ninguna fila`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fkReject(t, dir, tc.src, "execution", tc.msg)
			unchanged()
		})
	}
	// Valid writes: existing parent, NULL reference, unreferenced key change.
	rep := fkRun(t, dir, "INSERT INTO pedidos VALUES (12, 1), (13, NULL); UPDATE clientes SET id = 5 WHERE id = 2;")
	if len(rep.Results) != 2 || rep.Results[0].RowsAffected != 2 || rep.Results[1].RowsAffected != 1 {
		t.Fatalf("results = %+v", rep.Results)
	}
	if got := fkRows(t, dir, "SELECT * FROM clientes ORDER BY id;"); !reflect.DeepEqual(got, [][]any{{int64(1), "Ana"}, {int64(5), "Luis"}}) {
		t.Fatalf("clientes = %v", got)
	}
	if got := fkRows(t, dir, "SELECT * FROM pedidos ORDER BY id;"); !reflect.DeepEqual(got, [][]any{{int64(10), int64(1)}, {int64(11), nil}, {int64(12), int64(1)}, {int64(13), nil}}) {
		t.Fatalf("pedidos = %v", got)
	}
}

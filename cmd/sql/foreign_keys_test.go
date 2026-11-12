package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIForeignKeyRestrict(t *testing.T) {
	dir := t.TempDir()
	check := func(src string, code int, stdout, stderr string) {
		t.Helper()
		var out, err bytes.Buffer
		got := run([]string{"-db", dir}, strings.NewReader(src), &out, &err)
		if got != code || out.String() != stdout || err.String() != stderr {
			t.Fatalf("%s: code=%d stdout=%q stderr=%q", src, got, out.String(), err.String())
		}
	}
	check("CREATE TABLE p (id INT PRIMARY KEY); CREATE TABLE c (pid INT REFERENCES p(id));", 0, "[1] Tabla \"p\" creada\n[2] Tabla \"c\" creada\n", "")
	check("DROP TABLE p;", 1, "", "<stdin>:1:12: error semantic: No se puede eliminar la tabla \"p\": la FOREIGN KEY c.pid la referencia; elimine antes esa tabla o columna\n")
	check("INSERT INTO p VALUES (1); INSERT INTO c VALUES (1);", 0, "[1] 1 fila(s) insertada(s) en \"p\"\n[2] 1 fila(s) insertada(s) en \"c\"\n", "")
	check("DELETE FROM p;", 1, "", "<stdin>:1:1: error execution: Violación de FOREIGN KEY: no se puede eliminar de \"p\" porque el valor 1 de \"id\" es referenciado por c.pid; no se eliminó ninguna fila\n")
	check("SELECT * FROM p;", 0, "[1] 1 fila(s)\nid\n--\n1\n  plan: Recorrido completo de \"p\": 1 registros\n", "")
	check("DELETE FROM c; DELETE FROM p; DROP TABLE c; DROP TABLE p;", 0, "[1] 1 fila(s) eliminada(s) de \"c\"\n  plan: Recorrido completo de \"c\": 1 registros\n[2] 1 fila(s) eliminada(s) de \"p\"\n  plan: Recorrido completo de \"p\": 1 registros\n[3] Tabla \"c\" eliminada\n[4] Tabla \"p\" eliminada\n", "")
}

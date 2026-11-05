package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestAlterCLI(t *testing.T) {
	dir := t.TempDir()
	check := func(src string, wantCode int, wantOut, wantErr string) {
		t.Helper()
		var out, err bytes.Buffer
		code := run([]string{"-db", dir}, strings.NewReader(src), &out, &err)
		if code != wantCode || out.String() != wantOut || err.String() != wantErr {
			t.Fatalf("%s\ncode=%d stdout=%q stderr=%q\nwant code=%d stdout=%q stderr=%q", src, code, out.String(), err.String(), wantCode, wantOut, wantErr)
		}
	}
	check("CREATE TABLE t (id INT);", 0, "[1] Tabla \"t\" creada\n", "")
	check("INSERT INTO t VALUES (1);", 0, "[1] 1 fila(s) insertada(s) en \"t\"\n", "")
	check("ALTER TABLE t ADD COLUMN note INT;", 0, "[1] Tabla \"t\" modificada\n", "")
	check("ALTER TABLE t DROP COLUMN id;", 0, "[1] Tabla \"t\" modificada\n", "")
	check("ALTER TABLE t RENAME TO renamed;", 0, "[1] Tabla \"t\" renombrada a \"renamed\"\n", "")
	check("ALTER TABLE renamed DROP note;", 1, "", "<stdin>:1:26: error semantic: No se puede eliminar la última columna de la tabla \"renamed\"\n")
	check("ALTER TABLE renamed ADD required INT NOT NULL;", 1, "", "<stdin>:1:1: error execution: No se puede agregar la columna \"required\" con NOT NULL a la tabla \"renamed\" con registros\n")
	check("SELECT * FROM renamed;", 0, "[1] 1 fila(s)\nnote\n----\nNULL\n  plan: Recorrido completo de \"renamed\": 1 registros\n", "")
	check("DROP TABLE renamed;", 0, "[1] Tabla \"renamed\" eliminada\n", "")
	check("ALTER TABLE renamed ADD x INT;", 1, "", "<stdin>:1:13: error semantic: La tabla \"renamed\" no existe\n")
}

func TestAlterCLIAnalysisDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	var out, err bytes.Buffer
	code := run([]string{"-db", dir, "-analyze"}, strings.NewReader("CREATE TABLE t (id INT); ALTER TABLE t ADD x INT; ALTER TABLE t RENAME TO new; SELECT x FROM new;"), &out, &err)
	if code != 0 || out.String() != "Análisis correcto: sin errores léxicos, sintácticos ni semánticos.\n" || err.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), err.String())
	}
	files, readErr := os.ReadDir(dir)
	if readErr != nil || len(files) != 0 {
		t.Fatalf("analysis created files: %v / %v", files, readErr)
	}
}

func TestAlterCLIHelp(t *testing.T) {
	var out, err bytes.Buffer
	code := run([]string{"-h"}, strings.NewReader(""), &out, &err)
	if code != 2 || out.Len() != 0 || !strings.Contains(err.String(), "ALTER TABLE tabla ADD [COLUMN] columna tipo [PRIMARY KEY] [NOT NULL]") || !strings.Contains(err.String(), "ALTER TABLE tabla DROP [COLUMN] columna") || !strings.Contains(err.String(), "ALTER TABLE tabla RENAME TO nueva_tabla") {
		t.Fatalf("help: code=%d stdout=%q stderr=%q", code, out.String(), err.String())
	}
}

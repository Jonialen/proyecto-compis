package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestSQLForeignKeyRestrict(t *testing.T) {
	withSQLDir(t)
	runSQL := func(src string) sqlResponse {
		t.Helper()
		body, err := json.Marshal(map[string]any{"source": src, "execute": true})
		if err != nil {
			t.Fatal(err)
		}
		rec := request(t, http.MethodPost, "/api/sql/run", "application/json", string(body))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
		}
		var r sqlResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	r := runSQL(`CREATE TABLE p (id INT PRIMARY KEY); CREATE TABLE c (pid INT REFERENCES p(id)); INSERT INTO p VALUES (1); INSERT INTO c VALUES (1);`)
	if len(r.Diagnostics) != 0 || len(r.Results) != 4 || r.Results[2]["rowsAffected"] != float64(1) {
		t.Fatalf("setup: %+v", r)
	}
	r = runSQL(`DROP TABLE p;`)
	if len(r.Results) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Phase != "semantic" || r.Diagnostics[0].Message != `No se puede eliminar la tabla "p": la FOREIGN KEY c.pid la referencia; elimine antes esa tabla o columna` {
		t.Fatalf("drop: %+v", r)
	}
	r = runSQL(`DELETE FROM p;`)
	if len(r.Results) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Phase != "execution" || r.Diagnostics[0].Message != `Violación de FOREIGN KEY: no se puede eliminar de "p" porque el valor 1 de "id" es referenciado por c.pid; no se eliminó ninguna fila` {
		t.Fatalf("delete: %+v", r)
	}
	r = runSQL(`SELECT * FROM p;`)
	if len(r.Diagnostics) != 0 || len(r.Results) != 1 || !reflect.DeepEqual(r.Results[0]["rows"], []any{[]any{float64(1)}}) || r.Results[0]["rowsAffected"] != float64(1) {
		t.Fatalf("reopened: %+v", r)
	}
}

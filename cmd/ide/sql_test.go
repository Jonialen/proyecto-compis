package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withSQLDir(t *testing.T) string {
	t.Helper()
	old := sqlDataDir
	sqlDataDir = filepath.Join(t.TempDir(), "db")
	t.Cleanup(func() { sqlDataDir = old })
	return sqlDataDir
}

type sqlResponse struct {
	Tokens      []map[string]any `json:"tokens"`
	ParseTree   map[string]any   `json:"parseTree"`
	AST         map[string]any   `json:"ast"`
	Diagnostics []struct {
		Phase   string `json:"phase"`
		Message string `json:"message"`
		Line    int    `json:"line"`
		Col     int    `json:"col"`
	} `json:"diagnostics"`
	Results  []map[string]any `json:"results"`
	Executed bool             `json:"executed"`
}

func TestSQLRunExecutesAndReturnsEveryView(t *testing.T) {
	dir := withSQLDir(t)
	body := `{"source":"CREATE TABLE a (id INT PRIMARY KEY, n VARCHAR(5)); INSERT INTO a VALUES (1, 'x');","execute":true}`
	rec := request(t, http.MethodPost, "/api/sql/run", "application/json", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var rep sqlResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Executed || len(rep.Results) != 2 || len(rep.Diagnostics) != 0 || len(rep.Tokens) == 0 || rep.ParseTree == nil || rep.AST == nil {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.csv")); err != nil {
		t.Errorf("table file missing: %v", err)
	}
}

func TestSQLRunAnalyzeOnlyDoesNotExecute(t *testing.T) {
	dir := withSQLDir(t)
	rec := request(t, http.MethodPost, "/api/sql/run", "application/json", `{"source":"CREATE TABLE a (id INT);"}`)
	var rep sqlResponse
	json.Unmarshal(rec.Body.Bytes(), &rep)
	if rep.Executed || len(rep.Diagnostics) != 0 {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("analysis-only run touched disk: %v", err)
	}
}

func TestSQLRunReportsAllErrorPhases(t *testing.T) {
	withSQLDir(t)
	for source, phase := range map[string]string{
		"SELECT @;":          "lexical",
		"SELECT FROM t;":     "syntax",
		"DROP TABLE nada;":   "semantic",
		"SELECT 'sin cerrar": "lexical",
	} {
		b, _ := json.Marshal(map[string]any{"source": source, "execute": true})
		rec := request(t, http.MethodPost, "/api/sql/run", "application/json", string(b))
		var rep sqlResponse
		json.Unmarshal(rec.Body.Bytes(), &rep)
		if len(rep.Diagnostics) == 0 || rep.Diagnostics[0].Phase != phase || rep.Diagnostics[0].Line != 1 || rep.Diagnostics[0].Col == 0 {
			t.Errorf("%q: diagnostics = %+v, want first phase %s", source, rep.Diagnostics, phase)
		}
		if rep.Executed {
			t.Errorf("%q: must not execute with errors", source)
		}
	}
}

func TestSQLRunTransportBoundary(t *testing.T) {
	withSQLDir(t)
	cases := []struct {
		name, method, ct, body string
		want                   int
	}{
		{"wrong method", http.MethodGet, "", "", http.StatusMethodNotAllowed},
		{"wrong content type", http.MethodPost, "text/plain", `{}`, http.StatusUnsupportedMediaType},
		{"unknown field", http.MethodPost, "application/json", `{"source":"","dir":"/etc"}`, http.StatusBadRequest},
		{"invalid json", http.MethodPost, "application/json", `{`, http.StatusBadRequest},
		{"too large", http.MethodPost, "application/json", `{"source":"` + strings.Repeat("a", maxCompiscriptRequestBytes) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, c := range cases {
		if rec := request(t, c.method, "/api/sql/run", c.ct, c.body); rec.Code != c.want {
			t.Errorf("%s: status = %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

func TestSQLRunReturnsSelectRowsPlanAndIndexes(t *testing.T) {
	withSQLDir(t)
	body := `{"source":"CREATE TABLE t (id INT PRIMARY KEY, n VARCHAR(5)); INSERT INTO t VALUES (1, 'a'), (2, NULL), (3, 'c'); CREATE INDEX i ON t (id); SELECT n FROM t WHERE id >= 2 ORDER BY id;","execute":true}`
	rec := request(t, http.MethodPost, "/api/sql/run", "application/json", body)
	var rep struct {
		Diagnostics []any `json:"diagnostics"`
		Results     []struct {
			Kind    string   `json:"kind"`
			Columns []string `json:"columns"`
			Rows    [][]any  `json:"rows"`
			Plan    []string `json:"plan"`
		} `json:"results"`
		Indexes []struct{ Name, Table, Column string } `json:"indexes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil || len(rep.Diagnostics) != 0 {
		t.Fatalf("body = %s (err %v)", rec.Body, err)
	}
	sel := rep.Results[len(rep.Results)-1]
	if sel.Kind != "SELECT" || len(sel.Columns) != 1 || sel.Columns[0] != "n" || len(sel.Rows) != 2 || sel.Rows[0][0] != nil || sel.Rows[1][0] != "c" {
		t.Errorf("select result = %+v", sel)
	}
	if len(sel.Plan) != 1 || !strings.Contains(sel.Plan[0], `Índice "i" usado en "t"`) {
		t.Errorf("plan = %v", sel.Plan)
	}
	if len(rep.Indexes) != 1 || rep.Indexes[0].Name != "i" {
		t.Errorf("indexes = %+v", rep.Indexes)
	}
}

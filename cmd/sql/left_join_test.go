package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLeftJoinCLI(t *testing.T) {
	dir := t.TempDir()
	var out, stderr bytes.Buffer
	setup := "CREATE TABLE a (id INT); CREATE TABLE b (id INT); INSERT INTO a VALUES (1), (2); INSERT INTO b VALUES (1);"
	if code := run([]string{"-db", dir}, strings.NewReader(setup), &out, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("setup exit=%d stderr=%q", code, stderr.String())
	}
	for _, tc := range []struct {
		source, want string
		code         int
	}{
		{"SELECT a.id, b.id FROM a LEFT JOIN b ON a.id = b.id;", "[1] 2 fila(s)\na.id | b.id\n-----+-----\n1    | 1\n2    | NULL\n  plan: Recorrido completo de \"a\": 2 registros\n  plan: Recorrido completo de \"b\": 1 registros\n", 0},
		{"SELECT a.id, b.id FROM a LEFT OUTER JOIN b ON a.id = b.id;", "[1] 2 fila(s)\na.id | b.id\n-----+-----\n1    | 1\n2    | NULL\n  plan: Recorrido completo de \"a\": 2 registros\n  plan: Recorrido completo de \"b\": 1 registros\n", 0},
		{"SELECT a.id, b.id FROM a INNER JOIN b ON a.id = b.id;", "[1] 1 fila(s)\na.id | b.id\n-----+-----\n1    | 1\n  plan: Recorrido completo de \"a\": 2 registros\n  plan: Recorrido completo de \"b\": 1 registros\n", 0},
	} {
		out.Reset()
		stderr.Reset()
		code := run([]string{"-db", dir}, strings.NewReader(tc.source), &out, &stderr)
		if code != tc.code || out.String() != tc.want || stderr.String() != "" {
			t.Errorf("%s: exit=%d stdout=%q stderr=%q", tc.source, code, out.String(), stderr.String())
		}
	}
	out.Reset()
	stderr.Reset()
	code := run([]string{"-db", dir}, strings.NewReader("SELECT * FROM a LEFT JOIN b;"), &out, &stderr)
	wantErr := "<stdin>:1:28: error syntax: Se esperaba 'ON' pero se encontró ';'\n"
	if code != 1 || out.String() != "" || stderr.String() != wantErr {
		t.Errorf("rejected: exit=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
}

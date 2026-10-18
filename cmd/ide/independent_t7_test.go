package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript"
)

func TestIndependentT7(t *testing.T) {
	type item struct {
		Method, URL, CT, Body string
		Code                  int
		Error                 string
	}
	rows := []item{
		{"GET", "/api/health", "", "", 200, ""},
		{"GET", "/api/compiscript/analyze", "", "", 405, "method not allowed"},
		{"OPTIONS", "/api/compiscript/analyze", "", "", 204, ""},
		{"POST", "/api/compiscript/analyze", "text/plain", "{}", 415, "content type must be application/json"},
	}
	for _, v := range []struct {
		b string
		c int
		e string
	}{
		{"", 400, "invalid JSON"},
		{"{}", 400, "source is required"},
		{"null", 400, "source is required"},
		{`{"source":null}`, 400, "source is required"},
		{`{"source":true}`, 400, "invalid JSON"},
		{`{"Source":""}`, 400, "invalid JSON"},
		{`{"source":"","extra":0}`, 400, "invalid JSON"},
		{`{"source":"","sour\u0063e":"x"}`, 400, "duplicate source field"},
		{`{"source":""}true`, 400, "invalid JSON"},
		{"{\"source\":\"\xff\"}", 400, "source must be valid UTF-8"},
		{`{"source":"` + strings.Repeat(" ", 1<<20) + `"}`, 413, "request body too large"},
		{`{"source":"let x = 1;"}`, 200, ""},
		{`{"source":"@"}`, 200, ""},
		{`{"source":"let x = ;"}`, 200, ""},
		{`{"source":"let x: integer = true;"}`, 200, ""},
	} {
		rows = append(rows, item{"POST", "/api/compiscript/analyze", "application/json", v.b, v.c, v.e})
	}
	for _, b := range []string{
		`{}`,
		`{"yapar_content":"%token INT\n%%\ns : INT ;","methods":["ll1","slr","lalr"]}`,
		`{"yapar_content":"%token INT\n%%\ns : INT ;","methods":["ll1"],"input_content":"1","yalex_content":"rule tokens = ['0'-'9']+ { return INT }"}`,
	} {
		rows = append(rows, item{"POST", "/api/process", "application/json", b, 0, ""})
	}
	for i, row := range rows {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			call := func(url string) *httptest.ResponseRecorder {
				data := []byte(row.Body)
				h := sha256.Sum256(data)
				file := filepath.Join(t.TempDir(), "request.json")
				if e := os.WriteFile(file, data, 0600); e != nil {
					t.Fatal(e)
				}
				r := httptest.NewRequest(row.Method, url, bytes.NewReader(data))
				r.Header.Set("Content-Type", row.CT)
				w := httptest.NewRecorder()
				newMux().ServeHTTP(w, r)
				stored, e := os.ReadFile(file)
				if e != nil || sha256.Sum256(stored) != h || sha256.Sum256(data) != h {
					t.Fatal("input changed")
				}
				return w
			}
			w := call(row.URL)
			if row.Code != 0 && w.Code != row.Code {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}
			if row.Error != "" && w.Body.String() != fmt.Sprintf("{\"error\":%q}\n", row.Error) {
				t.Fatalf("error %q", w.Body.String())
			}
			if strings.Contains(row.URL, "compiscript") && row.Code != 200 {
				c := call("/api/compiscript/compile")
				if c.Code != w.Code || c.Body.String() != w.Body.String() || !reflect.DeepEqual(c.Header(), w.Header()) {
					t.Fatalf("compile transport differs: %d %s", c.Code, c.Body)
				}
			}
		})
	}
	for _, src := range []string{
		"", "let x = 1;", "@", "let x = ;", "let x: integer = true;", "print(1);",
		"let x = [1,2]; foreach (v in x) { v = v + 1; }",
	} {
		t.Run("compile/"+src, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"source": src})
			before := sha256.Sum256(body)
			r := httptest.NewRequest("POST", "/api/compiscript/compile", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json; charset=utf-8")
			w := httptest.NewRecorder()
			newMux().ServeHTTP(w, r)
			var got map[string]json.RawMessage
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil {
				t.Fatalf("response %d %s", w.Code, w.Body)
			}
			want := compiscript.Compile([]byte(src))
			text := ""
			if want.TAC != nil {
				text = want.TAC.String()
			} else if string(got["tac"]) != "null" || got["codegen"] != nil {
				t.Fatal("non atomic")
			}
			var actual string
			json.Unmarshal(got["tacText"], &actual)
			if text != actual {
				t.Fatal("TAC differs")
			}
			delete(got, "tacText")
			encoded, _ := json.Marshal(want)
			var expected map[string]json.RawMessage
			json.Unmarshal(encoded, &expected)
			if !reflect.DeepEqual(got, expected) || sha256.Sum256(body) != before {
				t.Fatal("report or input differs")
			}
		})
	}
	for _, n := range []int{1 << 20, (1 << 20) + 1} {
		body := []byte(`{"source":""}` + strings.Repeat(" ", n-13))
		if len(body) != n {
			t.Fatalf("fixture length %d want %d", len(body), n)
		}
		h := sha256.Sum256(body)
		r := httptest.NewRequest("POST", "/api/compiscript/compile", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newMux().ServeHTTP(w, r)
		want := 200
		if n > 1<<20 {
			want = 413
		}
		if w.Code != want || sha256.Sum256(body) != h {
			t.Fatalf("limit %d: %d %s", n, w.Code, w.Body)
		}
	}
}

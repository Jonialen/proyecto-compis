package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript"
)

func TestCompileFacadeParity(t *testing.T) {
	for _, tt := range []struct{ name, source, phase string }{
		{"empty", "", ""},
		{"numeric", "let x: integer = 2 + 3;", ""},
		{"closure", "function outer(): integer { let x: integer = 2; function inner(): integer { return x; } return inner(); } let n = outer();", ""},
		{"object", "class Box { let value: integer = 3; } let box = new Box(); let n = box.value;", ""},
		{"collection", "let values = [1, 2]; let sum: integer = 0; foreach (value in values) { sum = sum + value; }", ""},
		{"lexical", "@", "lexical"},
		{"syntax", "let x: integer = ;", "syntax"},
		{"semantic", "let x: integer = true;", "semantic"},
		{"codegen", "try{}catch(e){}", "codegen"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"source": tt.source})
			before := append([]byte(nil), body...)
			req := httptest.NewRequest(http.MethodPost, "/api/compiscript/compile", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			newMux().ServeHTTP(rec, req)
			if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatalf("response: %d %v %s", rec.Code, rec.Header(), rec.Body)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			want := compiscript.Compile([]byte(tt.source))
			if tt.phase == "" {
				if want.TAC == nil || want.Codegen == nil || len(want.Diagnostics) != 0 {
					t.Fatalf("valid fixture failed: %+v", want)
				}
			} else {
				found := false
				for _, diagnostic := range want.Diagnostics {
					found = found || string(diagnostic.Phase) == tt.phase
					if diagnostic.Span.Start.Line < 1 || diagnostic.Span.Start.Column < 1 {
						t.Fatalf("unlocated diagnostic: %+v", diagnostic)
					}
				}
				if !found || want.TAC != nil || got["codegen"] != nil || string(got["tac"]) != "null" {
					t.Fatalf("non-atomic or wrong phase failure: %s", rec.Body)
				}
			}
			text := ""
			if want.TAC != nil {
				text = want.TAC.String()
			}
			var gotText string
			if err := json.Unmarshal(got["tacText"], &gotText); err != nil || gotText != text {
				t.Fatalf("tacText = %q, want %q (%v)", gotText, text, err)
			}
			delete(got, "tacText")
			encoded, _ := json.Marshal(want)
			var expected map[string]json.RawMessage
			json.Unmarshal(encoded, &expected)
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("core report differs: %s vs %s", rec.Body, encoded)
			}
			if !bytes.Equal(body, before) {
				t.Fatal("request modified")
			}
		})
	}
}

func TestCompileStrictTransportMatchesAnalyze(t *testing.T) {
	cases := []struct{ method, contentType, body string }{
		{"GET", "", ""}, {"PUT", "application/json", `{}`}, {"OPTIONS", "", ""},
		{"POST", "", `{}`}, {"POST", "text/plain", `{}`},
		{"POST", "application/json", ``}, {"POST", "application/json", `{`},
		{"POST", "application/json", `{}`}, {"POST", "application/json", `null`},
		{"POST", "application/json", `{"source":null}`}, {"POST", "application/json", `{"source":4}`},
		{"POST", "application/json", `{"Source":""}`}, {"POST", "application/json", `{"source":"","SOURCE":"x"}`},
		{"POST", "application/json", `{"source":"","extra":0}`},
		{"POST", "application/json", `{"source":"","sour\u0063e":"x"}`},
		{"POST", "application/json", `{"source":""} {}`},
		{"POST", "application/json", "{\"source\":\"\xff\"}"},
		{"POST", "application/json", `{"source":"` + strings.Repeat("a", maxCompiscriptRequestBytes) + `"}`},
	}
	for i, tt := range cases {
		a := request(t, tt.method, "/api/compiscript/analyze", tt.contentType, tt.body)
		c := request(t, tt.method, "/api/compiscript/compile", tt.contentType, tt.body)
		if a.Code != c.Code || !reflect.DeepEqual(a.Header(), c.Header()) || a.Body.String() != c.Body.String() {
			t.Errorf("case %d: Analyze %d %v %q; Compile %d %v %q", i, a.Code, a.Header(), a.Body, c.Code, c.Header(), c.Body)
		}
	}
	if rec := request(t, http.MethodGet, "/api/health", "", ""); rec.Code != 200 || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatal("health changed")
	}
}

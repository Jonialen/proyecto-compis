package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	compiscript "genanalex/internal/compiscript"
	"genanalex/internal/compiscript/model"
)

func request(t *testing.T, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	return rec
}

func TestCompiscriptAnalyzeReturnsFacadeReport(t *testing.T) {
	sources := []string{
		"let x: integer = 1;",
		"let mensaje: string = \"¡Hola, 世界!\";",
		"let x: integer = ; print(x);",
		"",
	}
	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"source": source})
			if err != nil {
				t.Fatal(err)
			}
			first := request(t, http.MethodPost, "/api/compiscript/analyze", "application/json", string(body))
			second := request(t, http.MethodPost, "/api/compiscript/analyze", "application/json; charset=utf-8", string(body))
			if first.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", first.Code, first.Body.String())
			}
			if !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
				t.Fatal("repeated responses differ")
			}
			var got model.AnalysisReport
			if err := json.Unmarshal(first.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			want := compiscript.Analyze([]byte(source))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("report differs from facade\ngot: %#v\nwant: %#v", got, want)
			}
		})
	}
}

func TestCompiscriptAnalyzeRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name, contentType, body, message string
		status                           int
	}{
		{"missing content type", "", `{"source":""}`, "content type must be application/json", http.StatusUnsupportedMediaType},
		{"wrong content type", "text/plain", `{"source":""}`, "content type must be application/json", http.StatusUnsupportedMediaType},
		{"empty body", "application/json", "", "invalid JSON", http.StatusBadRequest},
		{"malformed JSON", "application/json", `{`, "invalid JSON", http.StatusBadRequest},
		{"missing source", "application/json", `{}`, "source is required", http.StatusBadRequest},
		{"null source", "application/json", `{"source":null}`, "source is required", http.StatusBadRequest},
		{"wrong source type", "application/json", `{"source":7}`, "invalid JSON", http.StatusBadRequest},
		{"invalid UTF-8", "application/json", "{\"source\":\"" + string([]byte{0xff}) + "\"}", "source must be valid UTF-8", http.StatusBadRequest},
		{"duplicate source", "application/json", `{"source":"first","source":"second"}`, "duplicate source field", http.StatusBadRequest},
		{"unknown field", "application/json", `{"source":"","extra":true}`, "invalid JSON", http.StatusBadRequest},
		{"trailing JSON", "application/json", `{"source":""}{}`, "invalid JSON", http.StatusBadRequest},
		{"oversized body", "application/json", `{"source":"` + strings.Repeat("a", 1<<20) + `"}`, "request body too large", http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := request(t, http.MethodPost, "/api/compiscript/analyze", tt.contentType, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
			var got map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got["error"] != tt.message {
				t.Fatalf("error = %q, want %q", got["error"], tt.message)
			}
		})
	}
}

func TestCompiscriptAnalyzeMethodAndCORS(t *testing.T) {
	get := request(t, http.MethodGet, "/api/compiscript/analyze", "", "")
	if get.Code != http.StatusMethodNotAllowed || get.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("GET status/Allow = %d/%q", get.Code, get.Header().Get("Allow"))
	}
	options := request(t, http.MethodOptions, "/api/compiscript/analyze", "", "")
	if options.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d", options.Code)
	}
}

func TestLegacyProcessRouteStillWorks(t *testing.T) {
	yalp, err := os.ReadFile("../../testdata/yapar/arithmetic.yalp")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"yapar_content": string(yalp), "methods": []string{"slr"}})
	if err != nil {
		t.Fatal(err)
	}
	rec := request(t, http.MethodPost, "/api/process", "application/json", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got processResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Success || got.Methods["slr"] == nil {
		t.Fatalf("legacy response = %#v", got)
	}
	if hash := fmt.Sprintf("%x", sha256.Sum256(rec.Body.Bytes())); hash != "e9539811ecefe6b5bfb70e3ba4782c0514a4d4f51dc48e349f7774c62a34c946" {
		t.Fatalf("legacy response hash = %s", hash)
	}
}

func TestCompiscriptBrowserContract(t *testing.T) {
	html, err := os.ReadFile("../../web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("../../web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"data-editor=\"cps\"", "data-result=\"source\"", "data-result=\"ast\"", "data-result=\"diagnostics\"", "data-result=\"environments\""} {
		if !bytes.Contains(html, []byte(token)) {
			t.Errorf("index.html lacks %q", token)
		}
	}
	js := string(script)
	for _, token := range []string{"/api/compiscript/analyze", "function renderCompiscript", "clearCompiscriptResults(err.message)", "textContent", "parentId", "symbols"} {
		if !strings.Contains(js, token) {
			t.Errorf("app.js lacks %q", token)
		}
	}
	runStart, renderStart := strings.Index(js, "async function runCompiscript"), strings.Index(js, "function renderCompiscript")
	if runStart < 0 || renderStart < 0 || !strings.Contains(js[runStart:renderStart], "clearCompiscriptResults(err.message)") {
		t.Error("Compiscript request failure does not clear stale results")
	}
	start := strings.Index(js, "function renderCompiscript")
	end := strings.Index(js[start:], "// ── UWU mode")
	if start >= 0 && end >= 0 && strings.Contains(js[start:start+end], "innerHTML") {
		t.Error("Compiscript renderer must not inject HTML")
	}
}

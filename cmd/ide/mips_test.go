package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"genanalex/internal/compiscript"
)

func TestMIPSAPI(t *testing.T) {
	for _, tt := range []struct {
		name, method, body, content string
		status                      int
	}{
		{"compile", "POST", `{"source":"print(3);"}`, "application/json", 200},
		{"source error", "POST", `{"source":"let n=missing;"}`, "application/json", 200},
		{"unknown field", "POST", `{"source":"","target":"mips"}`, "application/json", 400},
		{"method", "GET", "", "application/json", 405},
		{"content type", "POST", `{"source":""}`, "text/plain", 415},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/compiscript/mips", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.content)
			w := httptest.NewRecorder()
			newMux().ServeHTTP(w, req)
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			if tt.status == 200 {
				var report struct {
					compiscript.MIPSCompilationReport
					TACText string `json:"tacText"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
					t.Fatal(err)
				}
				if tt.name == "compile" {
					if !strings.Contains(report.MIPSText, ".globl main") || report.TACText != "print 3:integer\n" || len(report.Diagnostics) != 0 {
						t.Fatalf("report=%+v", report)
					}
				} else if report.MIPSText != "" || len(report.Diagnostics) == 0 {
					t.Fatalf("invalid rejection=%+v", report)
				}
			}
			if tt.status == 405 && w.Header().Get("Allow") != "POST" {
				t.Fatal("missing Allow POST")
			}
		})
	}
}

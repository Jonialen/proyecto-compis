package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"
)

func TestYAParProcessAnalyzesEmptyTokenStreams(t *testing.T) {
	lexer, err := os.ReadFile("../../testdata/yapar/arithmetic.yal")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, rule, input string
		accepted          bool
	}{
		{name: "nullable empty", accepted: true},
		{name: "nullable skipped whitespace", input: " \t\n", accepted: true},
		{name: "required token empty", rule: "INT"},
		{name: "required token skipped whitespace", rule: "INT", input: " \t\n"},
		{name: "existing nonempty success", rule: "INT", input: "7", accepted: true},
		{name: "existing nonempty rejection", rule: "INT", input: "+"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{
				"yalex_content": string(lexer), "yapar_content": "%token INT\n%%\nstart : " + tt.rule + " ;",
				"input_content": tt.input, "methods": []string{"ll1", "slr", "lalr"},
			})
			if err != nil {
				t.Fatal(err)
			}
			rec := request(t, http.MethodPost, "/api/process", "application/json", string(body))
			var got processResponse
			if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil || !got.Success || len(got.LexicalErrors) != 0 {
				t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
			}
			for _, method := range []string{"ll1", "slr", "lalr"} {
				result := got.Methods[method]
				if result == nil || result.Accepted == nil || *result.Accepted != tt.accepted {
					t.Fatalf("%s result = %#v, want Accepted=%v", method, result, tt.accepted)
				}
				if (result.Error == "") != tt.accepted {
					t.Fatalf("%s error = %q, accepted = %v", method, result.Error, tt.accepted)
				}
			}
		})
	}
}

func TestYAParProcessClassifiesGrammarErrors(t *testing.T) {
	for _, tt := range []struct {
		name, grammar, kind string
	}{
		{name: "specification", grammar: "%token INT\nstart : INT ;", kind: "spec"},
		{name: "grammar", grammar: "%token INT\n%%\nstart : missing ;", kind: "grammar"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"yapar_content": tt.grammar})
			if err != nil {
				t.Fatal(err)
			}
			rec := request(t, http.MethodPost, "/api/process", "application/json", string(body))
			var got map[string]any
			if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil {
				t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
			}
			if got["success"] != false || got["grammar_error"] == nil || got["grammar_error"] == "" || got["grammar_error_kind"] != tt.kind {
				t.Fatalf("response = %s, want error kind %q", rec.Body.String(), tt.kind)
			}
		})
	}
}

func TestYAParBrowserBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required for browser behavior tests")
	}
	cmd := exec.Command(node, "--test", "../../web/app.test.cjs")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("browser behavior tests: %v\n%s", err, output)
	}
}

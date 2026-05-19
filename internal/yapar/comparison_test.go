package yapar

import (
	"encoding/json"
	"strings"
	"testing"

	"genanalex/internal/shared"
)

func TestBuildComparisonReportAllMethodsSucceed(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS WS
IGNORE WS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	report := BuildComparisonReport(g, ff, []shared.Token{{Type: "ID", Lexeme: "x", Line: 1}, {Type: "PLUS", Lexeme: "+", Line: 1}, {Type: "ID", Lexeme: "y", Line: 1}}, ValidMethods())
	if report == nil {
		t.Fatal("BuildComparisonReport() = nil, want report")
	}
	if !report.HasSuccess() {
		t.Fatal("HasSuccess() = false, want true")
	}
	if len(report.Methods) != len(ValidMethods()) {
		t.Fatalf("len(report.Methods) = %d, want %d", len(report.Methods), len(ValidMethods()))
	}

	for i, method := range ValidMethods() {
		result := report.Methods[i]
		if result.Method != method {
			t.Fatalf("report.Methods[%d].Method = %q, want %q", i, result.Method, method)
		}
		if method == MethodLR0 || method == MethodLR1 {
			if got := result.Error; !strings.Contains(got, "not implemented yet") {
				t.Fatalf("report.Methods[%d].Error = %q, want not implemented yet", i, got)
			}
			continue
		}
		if result.Report == nil {
			t.Fatalf("report.Methods[%d].Report = nil, want visualization report", i)
		}
		if result.Accepted == nil || !*result.Accepted {
			t.Fatalf("report.Methods[%d].Accepted = %v, want true", i, result.Accepted)
		}
		if result.Error != "" {
			t.Fatalf("report.Methods[%d].Error = %q, want empty", i, result.Error)
		}
	}
}

func TestBuildComparisonReportCapturesMethodErrors(t *testing.T) {
	g := mustBuildGrammar(t, `%token A B
%%
s : A | A B ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	report := BuildComparisonReport(g, ff, nil, ValidMethods())
	if report == nil {
		t.Fatal("BuildComparisonReport() = nil, want report")
	}
	if !report.HasSuccess() {
		t.Fatal("HasSuccess() = false, want true when at least one method succeeds")
	}

	if got, want := report.Methods[0].Method, MethodLL1; got != want {
		t.Fatalf("report.Methods[0].Method = %q, want %q", got, want)
	}
	if got := report.Methods[0].Error; !strings.Contains(got, "ll1 conflict") {
		t.Fatalf("report.Methods[0].Error = %q, want LL1 conflict", got)
	}
	if got, want := report.Methods[1].Method, MethodLR0; got != want {
		t.Fatalf("report.Methods[1].Method = %q, want %q", got, want)
	}
	if got := report.Methods[1].Error; !strings.Contains(got, "not implemented yet") {
		t.Fatalf("report.Methods[1].Error = %q, want not implemented yet", got)
	}
	if got, want := report.Methods[2].Method, MethodSLR; got != want {
		t.Fatalf("report.Methods[2].Method = %q, want %q", got, want)
	}
	if report.Methods[2].Error != "" {
		t.Fatalf("report.Methods[2].Error = %q, want empty", report.Methods[2].Error)
	}
	if got, want := report.Methods[3].Method, MethodLR1; got != want {
		t.Fatalf("report.Methods[3].Method = %q, want %q", got, want)
	}
	if got := report.Methods[3].Error; !strings.Contains(got, "not implemented yet") {
		t.Fatalf("report.Methods[3].Error = %q, want not implemented yet", got)
	}
	if got, want := report.Methods[4].Method, MethodLALR; got != want {
		t.Fatalf("report.Methods[4].Method = %q, want %q", got, want)
	}
	if report.Methods[4].Error != "" {
		t.Fatalf("report.Methods[4].Error = %q, want empty", report.Methods[4].Error)
	}
}

func TestBuildComparisonReportWithoutTokensLeavesAcceptedNil(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	report := BuildComparisonReport(g, ff, nil, ValidMethods())
	if report == nil {
		t.Fatal("BuildComparisonReport() = nil, want report")
	}
	for i, result := range report.Methods {
		if result.Accepted != nil {
			t.Fatalf("report.Methods[%d].Accepted = %v, want nil", i, *result.Accepted)
		}
	}
}

func TestBuildComparisonReportAllFailuresReportNoSuccess(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	report := BuildComparisonReport(g, ff, nil, []Method{"broken-a", "broken-b"})
	if report == nil {
		t.Fatal("BuildComparisonReport() = nil, want report")
	}
	if report.HasSuccess() {
		t.Fatal("HasSuccess() = true, want false when all methods fail")
	}
	for i, result := range report.Methods {
		if result.Error == "" {
			t.Fatalf("report.Methods[%d].Error = empty, want failure message", i)
		}
	}
}

func TestRenderComparisonJSONIncludesMethodsAndNullAccepted(t *testing.T) {
	g := mustBuildGrammar(t, `%token ID PLUS
%%
expr : ID PLUS ID ;
`)
	ff, err := ComputeFirstFollow(g)
	if err != nil {
		t.Fatalf("ComputeFirstFollow() error = %v", err)
	}

	payload, err := RenderComparisonJSON(BuildComparisonReport(g, ff, nil, ValidMethods()))
	if err != nil {
		t.Fatalf("RenderComparisonJSON() error = %v", err)
	}

	var decoded struct {
		TotalTimeMS float64 `json:"total_time_ms"`
		Methods     []struct {
			Method   string     `json:"method"`
			Accepted *bool      `json:"accepted"`
			Error    string     `json:"error"`
			States   []struct{} `json:"states"`
		} `json:"methods"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\npayload=%s", err, payload)
	}
	if len(decoded.Methods) != len(ValidMethods()) {
		t.Fatalf("len(decoded.Methods) = %d, want %d", len(decoded.Methods), len(ValidMethods()))
	}
	if decoded.Methods[0].Method != string(MethodLL1) {
		t.Fatalf("decoded.Methods[0].Method = %q, want %q", decoded.Methods[0].Method, MethodLL1)
	}
	if decoded.Methods[0].Accepted != nil {
		t.Fatalf("decoded.Methods[0].Accepted = %v, want nil", *decoded.Methods[0].Accepted)
	}
	if len(decoded.Methods[0].States) == 0 {
		t.Fatal("decoded.Methods[0].States = 0, want serialized table states")
	}
}

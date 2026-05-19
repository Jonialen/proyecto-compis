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
	if len(report.Results) != len(ValidMethods()) {
		t.Fatalf("len(report.Results) = %d, want %d", len(report.Results), len(ValidMethods()))
	}

	for i, method := range ValidMethods() {
		result := report.Results[i]
		if result.Method != method {
			t.Fatalf("report.Results[%d].Method = %q, want %q", i, result.Method, method)
		}
		if result.Report == nil {
			t.Fatalf("report.Results[%d].Report = nil, want visualization report", i)
		}
		if result.Accepted == nil || !*result.Accepted {
			t.Fatalf("report.Results[%d].Accepted = %v, want true", i, result.Accepted)
		}
		if result.Error != "" {
			t.Fatalf("report.Results[%d].Error = %q, want empty", i, result.Error)
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

	if got, want := report.Results[0].Method, MethodSLR; got != want {
		t.Fatalf("report.Results[0].Method = %q, want %q", got, want)
	}
	if report.Results[0].Error != "" {
		t.Fatalf("report.Results[0].Error = %q, want empty", report.Results[0].Error)
	}
	if got, want := report.Results[1].Method, MethodLL1; got != want {
		t.Fatalf("report.Results[1].Method = %q, want %q", got, want)
	}
	if got := report.Results[1].Error; !strings.Contains(got, "ll1 conflict") {
		t.Fatalf("report.Results[1].Error = %q, want LL1 conflict", got)
	}
	if report.Results[2].Error != "" {
		t.Fatalf("report.Results[2].Error = %q, want empty", report.Results[2].Error)
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
	for i, result := range report.Results {
		if result.Accepted != nil {
			t.Fatalf("report.Results[%d].Accepted = %v, want nil", i, *result.Accepted)
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
	for i, result := range report.Results {
		if result.Error == "" {
			t.Fatalf("report.Results[%d].Error = empty, want failure message", i)
		}
	}
}

func TestRenderComparisonJSONIncludesResultsAndNullAccepted(t *testing.T) {
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
		DurationMS float64 `json:"duration_ms"`
		Results    []struct {
			Method   string     `json:"method"`
			Accepted *bool      `json:"accepted"`
			Error    string     `json:"error"`
			States   []struct{} `json:"states"`
		} `json:"results"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\npayload=%s", err, payload)
	}
	if len(decoded.Results) != len(ValidMethods()) {
		t.Fatalf("len(decoded.Results) = %d, want %d", len(decoded.Results), len(ValidMethods()))
	}
	if decoded.Results[0].Method != string(MethodSLR) {
		t.Fatalf("decoded.Results[0].Method = %q, want %q", decoded.Results[0].Method, MethodSLR)
	}
	if decoded.Results[0].Accepted != nil {
		t.Fatalf("decoded.Results[0].Accepted = %v, want nil", *decoded.Results[0].Accepted)
	}
	if len(decoded.Results[0].States) == 0 {
		t.Fatal("decoded.Results[0].States = 0, want serialized table states")
	}
}

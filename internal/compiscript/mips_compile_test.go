package compiscript_test

import (
	"encoding/json"
	"strings"
	"testing"

	"genanalex/internal/compiscript"
	"genanalex/internal/compiscript/mips"
)

func TestCompileMIPS(t *testing.T) {
	source := []byte(`function f(n:integer):integer{return n+1;}print(f(2));`)
	before := string(source)
	old := compiscript.Compile(source)
	oldJSON, _ := json.Marshal(old)
	report := compiscript.CompileMIPS(source)
	want, err := mips.Generate(old.TAC)
	if err != nil || len(report.Diagnostics) != 0 || report.MIPSText != want || report.TAC == nil {
		t.Fatalf("MIPS report=%+v err=%v", report, err)
	}
	afterJSON, _ := json.Marshal(compiscript.Compile(source))
	if string(oldJSON) != string(afterJSON) || string(source) != before {
		t.Fatal("changed source or original Compile contract")
	}
	for _, source := range []string{`let x=missing;`, `print(`} {
		old := compiscript.Compile([]byte(source))
		report := compiscript.CompileMIPS([]byte(source))
		a, _ := json.Marshal(old.Diagnostics)
		b, _ := json.Marshal(report.Diagnostics)
		if report.MIPSText != "" || string(a) != string(b) || report.TAC != nil {
			t.Fatalf("non-atomic frontend rejection: %+v", report)
		}
	}
	report = compiscript.CompileMIPS([]byte("print(" + strings.Repeat("9", 400) + ");"))
	if report.MIPSText != "" || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "MIPS_GENERATION" || report.Diagnostics[0].Phase != "codegen" || report.Diagnostics[0].Span.Start.Line != 1 {
		t.Fatalf("missing located backend rejection: %+v", report)
	}
}

package compiscript

import (
	"os"
	"strings"
	"testing"
)

func TestIndependentT4MixedSeed(t *testing.T) {
	source, err := os.ReadFile("../../testdata/compiscript/tac/acceptance.cps")
	if err != nil {
		t.Fatal(err)
	}
	source = []byte(strings.Replace(string(source), "install(basket, 10)", "install(basket, 2)", 1))
	report := Compile(source)
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		t.Fatal(report.Diagnostics)
	}
	values, err := runFunctionFrames(report.TAC)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"first": float64(14), "second": float64(19),
		"independent": float64(7), "continued": float64(25),
		"shared": float64(13), "total": float64(10),
		"result": float64(75),
	}
	for name, expected := range want {
		if values["s1."+name] != expected {
			t.Fatalf("%s=%v, want %v", name, values["s1."+name], expected)
		}
	}
	assertSymbolReport(t, report)
	assertClosureCLI(t, buildClosureCLI(t), string(source), want)
}

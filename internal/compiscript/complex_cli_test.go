package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestComplexSourceFilesCLI(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct {
		name, output, fault, phase string
		values                     map[string]any
	}{
		{
			name: "closures-ledger", output: "13\n17\n22\n103\n29\n184\n",
			values: map[string]any{"first": float64(13), "second": float64(17), "third": float64(22), "fourth": float64(103), "finalBalance": float64(29), "result": float64(184)},
		},
		{
			name: "matrix-transfers", output: "6\n20\n21\n32\n59\n",
			values: map[string]any{"visited": float64(6), "sum": float64(21), "trace": float64(32), "result": float64(59)},
		},
		{
			name: "recursive-exceptions", output: "19\n2\n11\n720\n7\n738\n",
			values: map[string]any{"total": float64(19), "caught": float64(2), "nested": float64(11), "factorial": float64(720), "calls": float64(7), "result": float64(738)},
		},
		{
			name: "evaluation-order", output: "1234\n20\n17\nchanged\n",
			values: map[string]any{"trace": float64(1234), "sum": float64(17), "finalText": "changed", "x": float64(5), "y": float64(6)},
		},
		{
			name: "uncaught-runtime-fault", output: "50\n", fault: "division by zero",
			values: map[string]any{"attempts": float64(2), "total": float64(50)},
		},
		{name: "invalid-scope", phase: "semantic"},
		{name: "invalid-list-argument", phase: "semantic"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "compiscript", "stress", tt.name+".cps"))
			if err != nil {
				t.Fatal(err)
			}
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256(source)
			want := Compile(source)
			if tt.phase == "" {
				if len(want.Diagnostics) != 0 || want.TAC == nil || want.Codegen == nil {
					t.Fatalf("valid source rejected: %+v", want.Diagnostics)
				}
				assertSymbolReport(t, want)
			} else {
				if want.TAC != nil || want.Codegen != nil || len(want.Diagnostics) == 0 {
					t.Fatalf("expected atomic rejection: %+v", want)
				}
				for _, diagnostic := range want.Diagnostics {
					if string(diagnostic.Phase) != tt.phase || diagnostic.Span.Start.Line < 1 {
						t.Fatalf("unexpected diagnostic: %+v", diagnostic)
					}
				}
			}
			for _, format := range []string{"tac", "json"} {
				command := exec.Command(cli, "-format="+format, path)
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				err := command.Run()
				exit := 0
				if err != nil {
					failure, ok := err.(*exec.ExitError)
					if !ok {
						t.Fatal(err)
					}
					exit = failure.ExitCode()
				}
				expectedExit := 0
				if tt.phase != "" {
					expectedExit = 1
				}
				if exit != expectedExit {
					t.Fatalf("%s: exit=%d want=%d stderr=%s", format, exit, expectedExit, &stderr)
				}
				if format == "tac" {
					if tt.phase == "" {
						if stdout.String() != want.TAC.String() || stderr.Len() != 0 {
							t.Fatal("TAC output differs from core")
						}
					} else {
						var diagnostics bytes.Buffer
						for _, diagnostic := range want.Diagnostics {
							fmt.Fprintf(&diagnostics, "%s %s at %d:%d: %s\n", diagnostic.Phase, diagnostic.Code, diagnostic.Span.Start.Line, diagnostic.Span.Start.Column, diagnostic.Message)
						}
						if stdout.Len() != 0 || stderr.String() != diagnostics.String() {
							t.Fatal("rejected text output is not atomic or diagnostics differ")
						}
					}
				} else {
					encoded, err := json.MarshalIndent(want, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					if stdout.String() != string(encoded)+"\n" || stderr.Len() != 0 {
						t.Fatal("CLI JSON differs from core")
					}
					if tt.phase == "" {
						var report CompilationReport
						if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
							t.Fatal(err)
						}
						values, output, fault := runFunctionFramesOutput(report.TAC)
						if (tt.fault == "" && fault != nil) || (tt.fault != "" && (fault == nil || fault.Error() != tt.fault)) {
							t.Fatalf("runtime fault=%v want=%q", fault, tt.fault)
						}
						if output != tt.output {
							t.Errorf("output=%q want=%q", output, tt.output)
						}
						for name, value := range tt.values {
							if values["s1."+name] != value {
								t.Errorf("%s=%v want=%v", name, values["s1."+name], value)
							}
						}
					}
				}
				if after, err := os.ReadFile(path); err != nil || sha256.Sum256(after) != before {
					t.Fatal("CLI changed source file")
				}
				t.Logf("%s: exit %d; source unchanged", format, exit)
			}
		})
	}
}

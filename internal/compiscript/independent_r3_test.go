package compiscript

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestIndependentR3(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tc := range []struct{ name, body string }{
		{"unqualified", "return value.read();"},
		{"qualified", "return this.value.read();"},
		{"nested capture", "function inner():integer{return value.read();}return inner();"},
		{"shadowed nominal origin", "class Box{function read():integer{return 99;}}return value.read();"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `class Box{function read():integer{return 7;}}class Holder{let value:Box;function constructor(v:Box){this.value=v;}function read():integer{` +
				tc.body +
				`}}let holder=new Holder(new Box());let x=holder.read();let y=holder.value.read();`
			input := []byte(source)
			before := append([]byte(nil), input...)
			analysis := Analyze(input)
			old, _ := json.Marshal(analysis)
			if len(analysis.Diagnostics) != 0 {
				t.Fatalf("Analyze: %+v", analysis.Diagnostics)
			}
			report := Compile(input)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("Compile: %+v", report.Diagnostics)
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil || values["s1.x"] != float64(7) || values["s1.y"] != float64(7) {
				t.Fatalf("values=%v err=%v", values, err)
			}
			after, _ := json.Marshal(Analyze(input))
			again, _ := json.Marshal(Compile(input))
			first, _ := json.Marshal(report)
			if !bytes.Equal(input, before) || !bytes.Equal(old, after) || !bytes.Equal(first, again) {
				t.Fatal("mutation or nondeterminism")
			}
			assertClosureCLI(t, cli, source, map[string]any{"x": float64(7), "y": float64(7)})
		})
	}
}

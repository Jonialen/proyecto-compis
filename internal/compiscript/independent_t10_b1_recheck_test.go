package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIndependentT10B1NominalFields(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct{ name, source string }{
		{"dynamic receiver", `class A{}class Holder{let value:A=null;}let h=new Holder();let effects=0;function set(b,v){effects=1;b.value=v;effects=99;}{class A{}try{set(h,new A());}catch(e){print(effects);}}print(h.value);`},
		{"known receiver unknown value", `class A{}class Holder{let value:A=null;}let h=new Holder();let effects=0;let v;{class A{}v=new A();}try{effects=1;h.value=v;effects=99;}catch(e){print(effects);}print(h.value);`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertDynamicExceptionsAccepted(t, tt.source)
			report := Compile([]byte(tt.source))
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatal(report.Diagnostics)
			}
			_, output, err := runFunctionFramesOutput(report.TAC)
			if err != nil || output != "1\nnull\n" {
				t.Fatalf("core output=%q error=%v", output, err)
			}
			path := filepath.Join(t.TempDir(), "input.cps")
			source := []byte(tt.source)
			if err := os.WriteFile(path, source, 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(cli, "-format=json", path)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("exit=%v stderr=%q", err, stderr.String())
			}
			var decoded CompilationReport
			if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.TAC == nil || len(decoded.Diagnostics) != 0 {
				t.Fatal(decoded.Diagnostics)
			}
			_, output, err = runFunctionFramesOutput(decoded.TAC)
			if err != nil || output != "1\nnull\n" {
				t.Fatalf("CLI output=%q error=%v", output, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(after) != sha256.Sum256(source) {
				t.Fatal("source mutated")
			}
		})
	}
}

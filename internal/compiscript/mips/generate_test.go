package mips_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"genanalex/internal/compiscript"
	"genanalex/internal/compiscript/mips"
	"genanalex/internal/compiscript/tac"
)

func execute(t *testing.T, source string) (string, string, int) {
	t.Helper()
	if testing.Short() {
		t.Skip("MARS integration omitted in short mode")
	}
	report := compiscript.Compile([]byte(source))
	if len(report.Diagnostics) != 0 || report.TAC == nil {
		t.Fatalf("frontend: %+v", report.Diagnostics)
	}
	assembly, err := mips.Generate(report.TAC)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "tools/mars/Mars4_5.jar")); err != nil {
		t.Fatal("install MARS with bash scripts/install-mars.sh:", err)
	}
	file := filepath.Join(t.TempDir(), "program.asm")
	if err := os.WriteFile(file, []byte(assembly), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "scripts/mars.sh"), "1000000", file)
	var out, stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		t.Fatal("MARS instruction/time budget:", ctx.Err(), stderr.String())
	}
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return out.String(), stderr.String(), code
}

func TestScalarsInMARS(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"large remainder", `print(1000000000000%3); print(-1000000000000%3);`, "1\n-1\n"},
		{"string bytes", `print("á🌎"); print("a\u0000b");`, "á🌎\na\x00b\n"},
		{"arithmetic", `let x=10; print(x+2*3); print(x-4); print(-x); print(x%3); print(1/2);`, "16\n6\n-10\n1\n0.5\n"},
		{"control", `let x=0; while(x<3){print(x);x=x+1;} if(x==3){print(true);}else{print(false);} print(!false);`, "0\n1\n2\ntrue\ntrue\n"},
		{"strings", `let a="hello"; print(a+" world"); print(a=="hello"); print(a!="other"); print(null);`, "hello world\ntrue\ntrue\nnull\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, code := execute(t, tt.source)
			if code != 0 || out != tt.want || stderr != "\n" {
				t.Fatalf("exit=%d stdout=%q stderr=%q want=%q", code, out, stderr, tt.want)
			}
		})
	}
}

func TestMalformedIRIsAtomic(t *testing.T) {
	for _, program := range []*tac.Program{nil, {Instructions: []tac.Instruction{{Op: "unknown"}}}, {Instructions: []tac.Instruction{{Op: tac.Jump, Label: "missing"}}}} {
		assembly, err := mips.Generate(program)
		if err == nil || assembly != "" {
			t.Fatalf("want error and no partial assembly, got %q %v", assembly, err)
		}
	}
}

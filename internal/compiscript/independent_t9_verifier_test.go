package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndependentT9(t *testing.T) {
	cases := []struct{ name, source, output, fault string }{{"recursive output", "function f(n:integer):integer{print(n);if(n==0){return 0;}return n+f(n-1);}print(f(3));", "3\n2\n1\n0\n6\n", ""},
		{"mixed signs", "print((-8)%3);print(8%(-3));print((-8.5)%3);print(8%2.5);", "-2\n2\n-2.5\n0.5\n", ""},
		{"zero guard", "print(4);print(8%0);print(9);", "4\n", "remainder by zero"},
		{"lazy null effects", "let n=0;function f(){n=n+1;}let x=false?f():null;print(n);print(x);", "0\nnull\n", ""},
		{"widened nested lists", "let a:float[][]=false?[[1.5]]:[[],[3]];print(a[1][0]/2);", "1.5\n", ""},
		{"function join signature", "function a(x:integer):float{return x/2;}function b(x:integer):float{return x/4;}let f=false?a:b;print(f(6));", "1.5\n", ""},
		{"assignment arguments", "function p(a:integer,b:integer,c:integer):integer{return 100*a+10*b+c;}let x=1;print(p(x,x=2,x=3));", "123\n", ""},
		{"assignment constructor", "class A{let v:integer;function constructor(a:integer,b:integer){this.v=a*10+b;}}let x=4;let a=new A(x,x=5);print(a.v);", "45\n", ""},
		{"receiver before rhs", "class A{let v:integer=0;}let a=new A();let old=a;let b=new A();print(a.v=((a=b)==b?7:8));print(old.v);print(b.v);", "7\n7\n0\n", ""},
		{"index before rhs", "let a=[3,4];let old=a;let b=[8,9];print(a[0]=((a=b)==b?6:7));print(old);print(a);", "6\n[6, 4]\n[8, 9]\n", ""},
		{"typed read", "let x:float;print(x);", "", "undefined"},
		{"block reentry", "let i=0;while(i<2){let x:integer;if(i==0){x=5;}i=i+1;print(x);}", "5\n", "undefined"},
		{"captured reentry", "function f(ok:boolean):integer{let x:integer;function set(){x=8;}function read():integer{return x;}if(ok){set();}return read();}print(f(true));print(f(false));", "8\n", "undefined"},
		{"null inherited aliases", "let n=0;function f(){n=n+1;return null;}let alias=f;class A{let v=alias();}class B:A{}let b=new B();print(b.v);print(n);print([null,alias()]);print(n);", "null\n1\n[null, null]\n2\n", ""},
		{"nominal origin shadow", "class A{}let a=new A();{class A{}let b=false?a:a;let xs=[b];print(xs[0]==a);}", "true\n", ""},
		{"common nominal join", "class A{}class B:A{}let a=new A();let b=new B();let c=true?b:a;print(c==b);", "true\n", ""},
		{"loop temporary liveness", "let sum=0;for(let i=0;i<6;i=i+1){sum=sum+(i<3?(i==0?2:3):(i==5?7:4));}print(sum);", "23\n", ""},
		{"strict ternary guard", "let n=0;print(0?(n=1):(n=2));print(n);", "", "nonboolean branch"},
		{"callee snapshot", "function a(x:integer):integer{return x+1;}function b(x:integer):integer{return x+9;}let f=a;function change():integer{f=b;return 2;}print(f(change()));print(f(2));", "3\n11\n", ""},
		{"print order", "let n=0;function f():integer{n=n+1;return n;}print([f(),f()]);print(n);", "[1, 2]\n2\n", ""}}
	cli := buildClosureCLI(t)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := []byte(tt.source)
			hash := sha256.Sum256(src)
			ast, pd := frontend.Parse(src)
			scopes, sd := semantic.Analyze(ast)
			if len(pd)+len(sd) != 0 {
				t.Fatalf("fixture diagnostics %v %v", pd, sd)
			}
			before, _ := json.Marshal([]any{ast, scopes})
			_, ds := tac.Generate(ast, scopes)
			after, _ := json.Marshal([]any{ast, scopes})
			if len(ds) != 0 || !bytes.Equal(before, after) {
				t.Fatalf("generate diagnostics/mutation: %v", ds)
			}
			r := Compile(src)
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("compile: %v", r.Diagnostics)
			}
			enc, _ := json.Marshal(r)
			again, _ := json.Marshal(Compile(src))
			if !bytes.Equal(enc, again) || sha256.Sum256(src) != hash {
				t.Fatal("nondeterministic/source mutation")
			}
			check := func(r CompilationReport) {
				_, out, e := runFunctionFramesOutput(r.TAC)
				if out != tt.output || (e == nil) != (tt.fault == "") || (e != nil && !strings.Contains(e.Error(), tt.fault)) {
					t.Errorf("out=%q err=%v expected=%q fault=%q", out, e, tt.output, tt.fault)
				}
			}
			check(r)
			path := filepath.Join(t.TempDir(), "input.cps")
			os.WriteFile(path, src, 0600)
			cmd := exec.Command(cli, "-format=json", path)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, e := cmd.Output()
			if e != nil || stderr.Len() != 0 {
				t.Fatalf("CLI: %v %s", e, &stderr)
			}
			var rr CompilationReport
			if e = json.Unmarshal(out, &rr); e != nil {
				t.Fatal(e)
			}
			check(rr)
			contents, _ := os.ReadFile(path)
			if sha256.Sum256(contents) != hash {
				t.Fatal("input mutation")
			}
		})
	}
}

// S11: metadata reserves undefined cells without fabricating a value.
func TestIndependentT9UndefinedLayout(t *testing.T) {
	for _, src := range []string{`let pending:integer;`, `function f(){let pending:integer;}f();`, `function f(){let pending:integer;function read():integer{return pending;}}f();`} {
		r := Compile([]byte(src))
		if r.TAC == nil || r.Codegen == nil || len(r.Diagnostics) != 0 {
			t.Fatal(r.Diagnostics)
		}
		found := false
		for _, symbol := range r.Codegen.Symbols {
			if symbol.Symbol.Name == "pending" {
				found = true
				if len(symbol.Locations) == 0 {
					t.Fatal("no reserved storage")
				}
			}
		}
		if !found {
			t.Fatal("missing pending")
		}
		check := func(tape []tac.Instruction) {
			for _, i := range tape {
				if i.Target.Kind == tac.Symbol && strings.HasSuffix(i.Target.Value, ".pending") && i.Op != tac.Declare {
					t.Fatalf("fabricated initializer: %+v", i)
				}
			}
		}
		check(r.TAC.Instructions)
		for _, p := range r.TAC.Procedures {
			check(p.Instructions)
		}
	}
}
func TestIndependentT9ReceiverAndReferences(t *testing.T) {
	for _, tt := range []struct{ source, want string }{
		{`class A{let n:integer=4;function get(x:integer):integer{return this.n+x;}}let a=new A();let b=new A();b.n=8;print(a.get((a=b)==b?2:3));`, "6\n"},
		{`class A{}function f(){}print(new A());print(f);print([null,null]);`, "<object s1.A>\n<function s1.f>\n[null, null]\n"},
		{`function f(ok:boolean):integer{let x:integer;if(ok){x=9;}return x;}print(f(true));print(f(false));`, "9\n"},
	} {
		r := Compile([]byte(tt.source))
		if r.TAC == nil {
			t.Fatal(r.Diagnostics)
		}
		_, out, e := runFunctionFramesOutput(r.TAC)
		if out != tt.want {
			t.Fatalf("out=%q expected=%q err=%v", out, tt.want, e)
		}
		if strings.Contains(tt.source, "f(false)") {
			if e == nil || !strings.Contains(e.Error(), "undefined") {
				t.Fatal(e)
			}
		} else if e != nil {
			t.Fatal(e)
		}
	}
}

func TestIndependentT9DocumentedExample(t *testing.T) {
	cli := buildClosureCLI(t)
	out, err := exec.Command(cli, "-format=json", "../../testdata/compiscript/tac/values-effects.cps").Output()
	if err != nil {
		t.Fatal(err)
	}
	var r CompilationReport
	if err = json.Unmarshal(out, &r); err != nil || r.TAC == nil {
		t.Fatal(err, r.Diagnostics)
	}
	values, text, err := runFunctionFramesOutput(r.TAC)
	if err != nil || text != "3\n1\nnull\n" || values["s1.x"] != float64(2) || values["s1.y"] != float64(3) || values["s1.count"] != float64(1) {
		t.Fatal(values, text, err)
	}
}

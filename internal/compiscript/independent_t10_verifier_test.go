package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

// S11 dynamic contracts and handler composition; S5 immutability; S6 CLI JSON execution.
// Retained independent probes always build the current CLI, without baseline overrides.
func TestIndependentT10Contracts(t *testing.T) {
	cli := buildClosureCLI(t)
	cases := []struct{ name, source, output string }{
		{"field null dynamic store", `class Box{let value=null;}function set(box,v){box.value=v;}let b=new Box();set(b,2);print(b.value);`, "2\n"},
		{"field inferred numeric", `class Box{let value=2;}let b=new Box();print(b.value);`, "2\n"},
		{"field actual tags", `class Box{let value;}let b=new Box();b.value=1;print(b.value);b.value="changed";print(b.value);`, "1\nchanged\n"},
		{"field null then tags", `class Box{let value=null;}let b=new Box();b.value=2;print(b.value);b.value=false;print(b.value);`, "2\nfalse\n"},
		{"unknown undefined until write", `let x;try{print(x);}catch(e){print("missing");}x="set";print(x);`, "missing\nset\n"},
		{"known noncallable", `let x=2;try{x(1);}catch(e){print(e);}`, "invalid function value\n"},
		{"known nonobject", `let x;x=2;try{print(x.n);}catch(e){print(e);}`, "invalid object receiver\n"},
		{"noncallable member", `class A{let n:integer=3;}let a=new A();try{a.n();}catch(e){print(e);}`, "invalid function value\n"},
		{"unknown call selection", `function one(x):integer{return 1;}function two(x):integer{return 2;}let selected;selected=one;function change():integer{selected=two;return 0;}print(selected(change()));print(selected(0));`, "1\n2\n"},
		{"unknown index selection", `let original=[1];let a; a=original;let b=[9];function change():integer{a=b;return 4;}a[0]=change();print(original[0]);print(b[0]);`, "4\n9\n"},
		{"null return validation inside handler", `function run(x){try{return x;}catch(e){print("caught");return null;}}print(run(1));`, "caught\nnull\n"},
		{"typed float boundary", `function f(x:float){print(x/2);}function g(x){f(x);}g(3);`, "1.5\n"},
		{"nested list boundary", `let effects=0;function f(x:integer[][]){effects=99;}function g(x){f(x);}try{g([[true]]);}catch(e){print("shape");}print(effects);`, "shape\n0\n"},
		{"list unknown content guard", `function f(x):integer[]{return [x];}try{print(f(true));}catch(e){print("bad");}`, "bad\n"},
		{"generic nested empty identity", `let saved;function f(x):integer[][]{saved=x;return x;}let xs=f([]);print(saved==xs);print(xs);`, "true\n[]\n"},
		{"do while crossed handler", `let n=0;try{do{n=n+1;try{continue;}catch(e){n=99;}}while(n<3);let bad=1%0;}catch(e){print(e);}print(n);`, "remainder by zero\n3\n"},
		{"for break handler depth", `try{for(let i=0;i<3;i=i+1){try{break;}catch(e){print("wrong");}}let bad=1/0;}catch(e){print("outer");}`, "outer\n"},
		{"switch break stays in try", `try{switch(true){case true:break;default:print("wrong");}let bad=1/0;}catch(e){print("kept");}`, "kept\n"},
		{"dynamic method receiver", `class A{let n:integer=2;function m(x):integer{return this.n+x;}}function get(x){let f=x.m;print(f(3));}get(new A());`, "5\n"},
		{"strict lazy branch", `function f(x){print(x?1:(1/0));}f(true);try{f(1);}catch(e){print("strict");}`, "1\nstrict\n"},
		{"nested catch return unwind", `function f():integer{try{try{let x=1/0;}catch(a){return 4;}}catch(b){return 99;}return 0;}try{print(f());let x=1/0;}catch(e){print("caller");}`, "4\ncaller\n"},
		{"fresh catch closures", `function no():boolean{return false;}let f=no;let g=no;let n=0;while(n<2){try{let bad=1/0;}catch(e){function same():boolean{return e==e;}if(n==0){f=same;}else{g=same;}}n=n+1;}print(f());print(g());`, "true\ntrue\n"},
		{"uninitialized field catch", `class A{let n;}let a=new A();try{print(a.n);}catch(e){print("undefined");}a.n=7;print(a.n);`, "undefined\n7\n"},
		{"guarded arithmetic remainder", `function f(x,y){print(x%y);print(x>=y);print(-(x*y));}f(5.5,2);`, "1.5\ntrue\n-11\n"},
		{"catch no lexical inheritance", `function no(){}let run=no;try{function fail(){let x=1/0;}run=fail;}catch(e){print("wrong");}try{run();}catch(e){print("dynamic");}`, "dynamic\n"},
		{"constructor guards before initializer", `let effects=0;function touch():integer{effects=effects+1;return 4;}class A{let n:integer=touch();function constructor(x:integer){effects=99;}}function make(x){let a=new A(x);}try{make(true);}catch(e){print(effects);}`, "0\n"},
		{"annotated null remains constrained", `class A{let n:string=null;}function set(a,v){a.n=v;}let a=new A();try{set(a,3);}catch(e){print("typed");}print(a.n);`, "typed\nnull\n"},
		{"nominal parameter origin", `class A{}let effects=0;function use(x:A){effects=99;}function pass(x){use(x);}{class A{}try{pass(new A());}catch(e){print(effects);}}`, "0\n"},
		{"null receiver dynamic", `function get(x){print(x.n);}try{get(null);}catch(e){print(e);}`, "null receiver\n"},
		{"negative dynamic index", `function get(x,i){print(x[i]);}try{get([1],-1);}catch(e){print(e);}`, "list index out of bounds\n"},
		{"empty dynamic foreach", `function f(x){foreach(v in x){print("wrong");}print("done");}f([]);`, "done\n"},
		{"request assignment example", `let x=1;let y=x+(x=2);print(x);print(y);`, "2\n3\n"},
		{"request try example", `try{let x=1;}catch(e){print(e);}`, ""},
		{"document example", `function inc(x):integer{return x+1;}let result=inc(2);let effects=0;try{effects=effects+1;print(result);let bad=1/0;effects=100;}catch(error){print(error);effects=effects+10;}print(effects);`, "3\ndivision by zero\n11\n"},
		{"inherited untyped field and method capture", `class Base{let value;function set(v){value=v;}function read():integer{function capture():integer{return value;}return capture();}}class Child:Base{let other=null;}let b=new Child();try{b.read();}catch(e){print("undefined");}b.set(3);print(b.read());b.value=false;print(b.value);b.other=2;print(b.other);`, "undefined\n3\nfalse\n2\n"},
		{"annotated direct field guard preserves prefix", `class A{let value:string=null;}let b=new A();let v;v=2;let effects=0;try{effects=1;b.value=v;effects=99;}catch(e){print("typed");}print(effects);print(b.value);`, "typed\n1\nnull\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			ast, pd := frontend.Parse(source)
			scopes, sd := semantic.Analyze(ast)
			if len(pd)+len(sd) != 0 {
				t.Fatalf("Analyze rejects: %v %v", pd, sd)
			}
			before, _ := json.Marshal([]any{ast, scopes})
			generated, ds := tac.Generate(ast, scopes)
			after, _ := json.Marshal([]any{ast, scopes})
			if !bytes.Equal(before, after) {
				t.Fatal("AST/scopes mutated")
			}
			if generated == nil || len(ds) != 0 {
				t.Fatalf("Generate: %v", ds)
			}
			report := Compile(source)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("Compile: %v", report.Diagnostics)
			}
			first, _ := json.Marshal(report)
			again, _ := json.Marshal(Compile(source))
			if !bytes.Equal(first, again) {
				t.Fatal("nondeterministic")
			}
			_, out, err := runFunctionFramesOutput(report.TAC)
			if err != nil || out != tt.output {
				t.Errorf("core output=%q error=%v want=%q", out, err, tt.output)
			}
			file := filepath.Join(t.TempDir(), "input.cps")
			if err := os.WriteFile(file, source, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(cli, "-format=json", file)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("CLI exit=%v stderr=%q stdout=%s", err, stderr.String(), stdout.String())
			}
			var decoded CompilationReport
			if err = json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			_, out, err = runFunctionFramesOutput(decoded.TAC)
			if err != nil || out != tt.output {
				t.Errorf("CLI output=%q error=%v want=%q", out, err, tt.output)
			}
			actual, _ := os.ReadFile(file)
			if sha256.Sum256(actual) != sha256.Sum256(source) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestIndependentT10UnannotatedFieldLayouts(t *testing.T) {
	source := []byte(`class Base{let value;let fixed:string=null;}class Child:Base{let own=null;}let b=new Child();`)
	ast, pd := frontend.Parse(source)
	scopes, sd := semantic.Analyze(ast)
	if len(pd)+len(sd) != 0 {
		t.Fatalf("fixture: %v %v", pd, sd)
	}
	before, _ := json.Marshal([]any{ast, scopes})
	p, ds := tac.Generate(ast, scopes)
	after, _ := json.Marshal([]any{ast, scopes})
	if p == nil || len(ds) != 0 || !bytes.Equal(before, after) {
		t.Fatalf("Generate changed/rejected inputs: %v", ds)
	}
	if len(p.Classes) != 2 || len(p.Classes[0].Fields) != 2 || len(p.Classes[1].Fields) != 3 {
		t.Fatal("missing actual inherited field layouts")
	}
	for _, class := range p.Classes {
		for _, field := range class.Fields {
			want := model.TypeError
			if field.Name == "fixed" {
				want = model.TypeString
			}
			if field.Type.Kind != want {
				t.Fatalf("%s field %s kind=%s want=%s", class.ID, field.ID, field.Type.Kind, want)
			}
			if field.Name == "value" {
				for _, procedure := range p.Procedures {
					for _, instruction := range procedure.Instructions {
						if instruction.Op == tac.StoreField && instruction.Member == field.ID {
							t.Fatalf("fabricated initializer for %s", field.ID)
						}
					}
				}
			}
		}
	}
	for i, base := range p.Classes[0].Fields {
		child := p.Classes[1].Fields[i]
		a, _ := json.Marshal(base)
		b, _ := json.Marshal(child)
		if !bytes.Equal(a, b) {
			t.Fatalf("inherited declaration contract changed: %s vs %s", a, b)
		}
	}
}

// S5 typed guards retain their exact fault timing in core and current CLI tapes.
func TestIndependentT10LegacyTiming(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct{ source, fault string }{
		{`let effects=0;class A{}class B:A{let n:integer=2;}let b=[new B()];let a:A[]=b;a[0]=new A();let x=b[0].n;effects=1;`, "nominal object type mismatch"},
		{`let effects=0;function a(x:integer):integer{return 1;}function b(x:boolean):integer{effects=1;return 2;}let f=a;f=b;let value=f(3);`, "indirect call signature mismatch"},
	} {
		t.Run(tt.fault, func(t *testing.T) {
			assertDynamicExceptionsAccepted(t, tt.source)
			file := filepath.Join(t.TempDir(), "input.cps")
			if err := os.WriteFile(file, []byte(tt.source), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(cli, "-format=json", file)
			var raw, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &raw, &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("CLI exit=%v stderr=%q", err, &stderr)
			}
			var decoded CompilationReport
			if err := json.Unmarshal(raw.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			for _, r := range []CompilationReport{decoded, Compile([]byte(tt.source))} {
				if r.TAC == nil || len(r.Diagnostics) != 0 {
					t.Fatal(r.Diagnostics)
				}
				values, out, err := runFunctionFramesOutput(r.TAC)
				if err == nil || err.Error() != tt.fault || out != "" || values["s1.effects"] != float64(0) {
					t.Fatalf("values=%v out=%q fault=%v", values, out, err)
				}
			}
			actual, err := os.ReadFile(file)
			if err != nil || sha256.Sum256(actual) != sha256.Sum256([]byte(tt.source)) {
				t.Fatal("source changed")
			}
		})
	}
}

// S11 malformed IR and resource limits must not enter source handlers.
func TestIndependentT10FatalBoundary(t *testing.T) {
	for _, tt := range []struct {
		name, source, want string
		mutate             bool
	}{
		{"budget", `try{while(true){}}catch(e){print("wrong");}`, "instruction budget exceeded", false},
		{"bad opcode", `try{print(1);}catch(e){print("wrong");}`, "unknown opcode independent-broken", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := Compile([]byte(tt.source))
			if r.TAC == nil {
				t.Fatal(r.Diagnostics)
			}
			if tt.mutate {
				for i := range r.TAC.Instructions {
					if r.TAC.Instructions[i].Op == tac.Print {
						r.TAC.Instructions[i].Op = "independent-broken"
						break
					}
				}
			}
			_, out, err := runFunctionFramesOutput(r.TAC)
			if err == nil || err.Error() != tt.want || out != "" {
				t.Fatalf("out=%q err=%v want=%q", out, err, tt.want)
			}
		})
	}
}

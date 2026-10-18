package compiscript

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

type valuesEffectsCase struct {
	name, source string
	want         map[string]any
	output       string
	fault        string
}

var valuesEffectsCases = []valuesEffectsCase{
	{name: "print primitives and frames", source: `print(1); print(2.5); print(true); print(false); print("hello"); print(null); function f(){print("callee");} f();`, output: "1\n2.5\ntrue\nfalse\nhello\nnull\ncallee\n"},
	{name: "numeric remainder", source: `let a=7%3; let b=7.5%2.0; let c=7%2.5; let d=7.5%2; let e=(-7)%3;`, want: map[string]any{"a": float64(1), "b": 1.5, "c": float64(2), "d": 1.5, "e": float64(-1)}},
	{name: "remainder zero", source: `let count=1; let bad=2.5%0; count=2;`, want: map[string]any{"count": float64(1)}, fault: "remainder by zero"},
	{name: "lazy ternary", source: `let x=0; let y=true?(x=1):(x=2); let z=false?(1/0):(y+2);`, want: map[string]any{"x": float64(1), "y": float64(1), "z": float64(3)}},
	{name: "assignment snapshots", source: `let x=1; let y=x+(x=2); let z=(x=3)+(x=4);`, want: map[string]any{"x": float64(4), "y": float64(3), "z": float64(7)}},
	{name: "argument order", source: `function pair(a:integer,b:integer):integer{return a*10+b;} let x=1; let y=pair(x,x=2);`, want: map[string]any{"x": float64(2), "y": float64(12)}},
	{name: "typed uninitialized", source: `let x:integer; x=3; let y=x;`, want: map[string]any{"x": float64(3), "y": float64(3)}},
	{name: "undefined read", source: `let count=1; let x:integer; let y=x; count=2;`, want: map[string]any{"count": float64(1)}, fault: "undefined"},
	{name: "fresh block cells", source: `let count=0; while(count<2){let x:integer; if(count==0){x=9;} count=count+1; print(x);}`, want: map[string]any{"count": float64(2)}, output: "9\n", fault: "undefined"},
	{name: "null storage fields and lists", source: `let n=null; class Box{let value=null;} let b=new Box(); let xs=[null,null]; let a=xs[1]; let c=b.value;`, want: map[string]any{"n": nil, "a": nil, "c": nil}},
	{name: "null calls and returns", source: `let count=0; function f(){count=count+1; return null;} function g(){return f();} let n=g(); let xs=[f(),null]; let a=xs[0];`, want: map[string]any{"count": float64(2), "n": nil, "a": nil}},
	{name: "nested contextual joins", source: `let xs:float[][]=true?[[],[1]]:[[2.5]]; let n=xs[1][0]; let ys:float[][]=[]; let result=(ys=false?[[3]]:[[],[4]]); let m=result[1][0];`, want: map[string]any{"n": float64(1), "m": float64(4)}},
	{name: "old behavior", source: `function f(x:integer):integer{return x+1;} let x=1; let y=x+f(2); for(let i=0;i<2;i=i+1){x=x+1;}`, want: map[string]any{"x": float64(3), "y": float64(4)}},
	{name: "function join with capture", source: `function outer(seed:integer):integer{let n=seed; function a(x:integer):integer{n=n+x;return n;} function b(x:integer):integer{n=n-x;return n;} let f=true?a:b; return f(3);} let result=outer(7);`, want: map[string]any{"result": float64(10)}},
	{name: "nominal join under shadow", source: `class A{} let original=new A(); {class A{} let selected=true?original:original; print(selected==original);}`, output: "true\n"},
	{name: "nominal list join", source: `class A{} let a=new A(); let xs:A[]=true?[]:[a]; let ys:A[]=false?[]:[a]; let same=ys[0]==a;`, want: map[string]any{"same": true}},
	{name: "callee chosen before arguments", source: `function a(x:integer):integer{return x+10;} function b(x:integer):integer{return x+100;} let f=true?a:b; function change():integer{f=b;return 1;} let result=f(change()); let next=f(1);`, want: map[string]any{"result": float64(11), "next": float64(101)}},
	{name: "index target before assignment effects", source: `let a=[1,2]; let old=a; let b=[8,9]; let i=0; let stored=(a[i]=(i=1)); let changed=(a[0]=((a=b)==b?9:0)); let first=old[0]; let second=b[0];`, want: map[string]any{"stored": float64(1), "changed": float64(9), "first": float64(9), "second": float64(8)}},
	{name: "property target before assignment effects", source: `class A{let n:integer=0;} let a=new A(); let old=a; let b=new A(); let stored=(a.n=((a=b)==b?3:0)); let first=old.n; let second=b.n;`, want: map[string]any{"stored": float64(3), "first": float64(3), "second": float64(0)}},
	{name: "widened assignment result", source: `let x:float=0; let y=(x=3); let result=y/2;`, want: map[string]any{"x": float64(3), "y": float64(3), "result": 1.5}},
	{name: "fresh invocation cells", source: `let count=0; function f(flag:boolean):integer{let x:integer;if(flag){x=5;}count=count+1;return x;} let first=f(true);let second=f(false);`, want: map[string]any{"count": float64(2), "first": float64(5)}, fault: "undefined"},
	{name: "fresh captured cells", source: `let count=0; function f(flag:boolean):integer{let x:integer;function read():integer{return x;}if(flag){x=5;}count=count+1;return read();}let first=f(true);let second=f(false);`, want: map[string]any{"count": float64(2), "first": float64(5)}, fault: "undefined"},
	{name: "null implicit and indirect effects", source: `let count=0; function f(){count=count+1;}let alias=f;let n=true?alias():null;let xs=[null,alias()];print(n);print(xs);`, want: map[string]any{"count": float64(2), "n": nil}, output: "null\n[null, null]\n"},
	{name: "reference print", source: `class A{}let a=new A();function f(){}print(a);print(f);print([1,2]);`, output: "<object s1.A>\n<function s1.f>\n[1, 2]\n"},
	{name: "boolean condition guard", source: `let count=1; let x=1?(count=2):(count=3);`, want: map[string]any{"count": float64(1)}, fault: "nonboolean branch"},
	{name: "loop nested joins", source: `let n=0;let result=0;while(n<4){result=result+(n<2?(n==0?1:2):(n==2?3:4));n=n+1;}`, want: map[string]any{"n": float64(4), "result": float64(10)}},
	{name: "null constant and joined fields", source: `class A{const fixed=null;let chosen=true?null:null;}let a=new A();let first=a.fixed;let second=a.chosen;`, want: map[string]any{"first": nil, "second": nil}},
	{name: "undefined global initialized through call", source: `let pending:integer;function set(){pending=3;}set();let result=pending;`, want: map[string]any{"pending": float64(3), "result": float64(3)}},
	{name: "undefined capture initialized through call", source: `function outer():integer{let pending:integer;function set(){pending=4;}set();return pending;}let result=outer();`, want: map[string]any{"result": float64(4)}},
	{name: "standalone common nominal join", source: `class Base{}class Child:Base{}let child=new Child();let base=new Base();print(false?child:base);`, output: "<object s1.Base>\n"},
	{name: "null fields call global alias", source: `let count=0;function f(){count=count+1;}let alias=f;class Base{let n=alias();}class Child:Base{}let child=new Child();let n=child.n;`, want: map[string]any{"count": float64(1), "n": nil}},
}

func assertValuesEffects(t *testing.T, report CompilationReport, tt valuesEffectsCase) {
	t.Helper()
	if report.TAC == nil || len(report.Diagnostics) != 0 || report.Codegen == nil {
		t.Fatalf("compile: %+v", report.Diagnostics)
	}
	got, output, err := runFunctionFramesOutput(report.TAC)
	if (err == nil) != (tt.fault == "") || (err != nil && !strings.Contains(err.Error(), tt.fault)) {
		t.Fatalf("execution values=%v output=%q err=%v want fault=%q", got, output, err, tt.fault)
	}
	if output != tt.output {
		t.Errorf("output=%q want %q", output, tt.output)
	}
	for name, want := range tt.want {
		value, exists := got["s1."+name]
		if !exists || !reflect.DeepEqual(value, want) {
			t.Errorf("%s=%v exists=%t want %v", name, value, exists, want)
		}
	}
}

func TestValuesEffectsStorageAndImmutability(t *testing.T) {
	for _, source := range []string{`let pending:integer;`, `function f(){let pending:integer;} f();`, `function outer(){let pending:integer;function read():integer{return pending;}} outer();`} {
		t.Run(source, func(t *testing.T) {
			program, pd := frontend.Parse([]byte(source))
			scopes, sd := semantic.Analyze(program)
			if len(pd)+len(sd) != 0 {
				t.Fatalf("fixture: %v %v", pd, sd)
			}
			before, _ := json.Marshal([]any{program, scopes})
			generated, ds := tac.Generate(program, scopes)
			after, _ := json.Marshal([]any{program, scopes})
			if generated == nil || len(ds) != 0 || string(before) != string(after) {
				t.Fatalf("Generate mutates or rejects: %v", ds)
			}
			report := Compile([]byte(source))
			if report.Codegen == nil {
				t.Fatal(report.Diagnostics)
			}
			found := false
			for _, symbol := range report.Codegen.Symbols {
				if symbol.Symbol.Name == "pending" {
					found = true
					if len(symbol.Locations) == 0 {
						t.Fatal("reserved binding missing actual location")
					}
				}
			}
			if !found {
				t.Fatal("missing pending binding")
			}
			check := func(tape []tac.Instruction) {
				for _, i := range tape {
					if i.Target.Kind == tac.Symbol && strings.HasSuffix(i.Target.Value, ".pending") && i.Op != tac.Declare {
						t.Fatalf("fabricated initializer: %+v", i)
					}
				}
			}
			check(report.TAC.Instructions)
			for _, p := range report.TAC.Procedures {
				check(p.Instructions)
			}
		})
	}
}

func TestValuesEffectsCompile(t *testing.T) {
	for _, tt := range valuesEffectsCases {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			before, _ := json.Marshal(Analyze(source))
			report := Compile(source)
			assertValuesEffects(t, report, tt)
			after, _ := json.Marshal(Analyze(source))
			if string(source) != tt.source || string(before) != string(after) {
				t.Fatal("source or Analyze changed")
			}
			again, _ := json.Marshal(Compile(source))
			first, _ := json.Marshal(report)
			if string(again) != string(first) {
				t.Fatal("nondeterministic compilation")
			}
		})
	}
}

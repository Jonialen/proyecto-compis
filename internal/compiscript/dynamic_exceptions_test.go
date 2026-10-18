package compiscript

import (
	"encoding/json"
	"os"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

const dynamicExceptionsExample = `function inc(x):integer { return x+1; }
let result=inc(2);
let effects=0;
try {
  effects=effects+1;
  print(result);
  let bad=1/0;
  effects=100;
} catch(error) {
  print(error);
  effects=effects+10;
}
print(effects);
`

// These cases exercise the public compiler and the shared bounded frame oracle.
// TypeError in semantic snapshots is unknown metadata, not a new source type.
var dynamicExceptionsCases = []valuesEffectsCase{
	{name: "checked in example", source: dynamicExceptionsExample, want: map[string]any{"result": float64(3), "effects": float64(11)}, output: "3\ndivision by zero\n11\n"},
	{name: "untyped arithmetic", source: `function inc(x):integer{return x+1;}let a=inc(2);print(a);`, want: map[string]any{"a": float64(3)}, output: "3\n"},
	{name: "untyped identity result boundary", source: `function id(x):integer{return x;}let a=id(2);print(a);`, want: map[string]any{"a": float64(2)}, output: "2\n"},
	{name: "untyped callable argument", source: `function apply(f):integer{return f();}function one():integer{return 1;}let a=apply(one);print(a);`, want: map[string]any{"a": float64(1)}, output: "1\n"},
	{name: "unannotated undefined binding", source: `let x;x=2;let y=x+1;print(y);`, want: map[string]any{"x": float64(2), "y": float64(3)}, output: "3\n"},
	{name: "unknown numeric promotion", source: `function add(x,y):float{return x+y;}let a=add(2,0.5);print(a);`, want: map[string]any{"a": 2.5}, output: "2.5\n"},
	{name: "unknown string addition", source: `function add(x,y):string{return x+y;}let a=add("a","b");print(a);`, want: map[string]any{"a": "ab"}, output: "ab\n"},
	{name: "unknown capture", source: `function outer(x):integer{function read():integer{return x;}return read();}let a=outer(4);print(a);`, want: map[string]any{"a": float64(4)}, output: "4\n"},
	{name: "unknown recursion", source: `function sum(x,n:integer):integer{if(n==0){return x;}return sum(x+1,n-1);}let a=sum(2,3);print(a);`, want: map[string]any{"a": float64(5)}, output: "5\n"},
	{name: "unknown field argument", source: `class Box{let n:integer=7;}function read(x):integer{return x.n;}let a=read(new Box());print(a);`, want: map[string]any{"a": float64(7)}, output: "7\n"},
	{name: "unknown list argument", source: `function read(x):integer{return x[0];}let a=read([8]);print(a);`, want: map[string]any{"a": float64(8)}, output: "8\n"},
	{name: "unknown foreach", source: `function sum(xs):integer{let n=0;foreach(x in xs){n=n+x;}return n;}let a=sum([2,3]);print(a);`, want: map[string]any{"a": float64(5)}, output: "5\n"},
	{name: "unknown strict boolean", source: `function choose(flag):integer{if(flag){return 1;}return 2;}let a=choose(true);print(a);`, want: map[string]any{"a": float64(1)}, output: "1\n"},
	{name: "unknown lazy join", source: `function choose(flag,x,y):integer{return flag?x:y;}let a=choose(false,1,2);print(a);`, want: map[string]any{"a": float64(2)}, output: "2\n"},
	{name: "unknown switch selector and case", source: `function choose(flag,value):integer{switch(flag){case value:return 1;default:return 2;}}let a=choose(true,true);let b=choose(false,true);print(a);print(b);`, want: map[string]any{"a": float64(1), "b": float64(2)}, output: "1\n2\n"},
	{name: "unknown switch rejects nonboolean selector", source: `function choose(flag){switch(flag){default:print("wrong");}}let count=0;try{choose(1);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown switch rejects nonboolean case", source: `function choose(flag,value){switch(flag){case value:print("wrong");}}let count=0;try{choose(true,1);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unrelated nominal ternary retains actual identity", source: `class A{let n:integer=1;}class B{let n:integer=2;}let chosen=false?new A():new B();print(chosen.n);print(false?new A():new B());`, output: "2\n<object s1.B>\n"},
	{name: "empty generic list checked at typed boundary", source: `function count(xs:integer[]):integer{let n=0;foreach(x in xs){n=n+1;}return n;}function pass(xs):integer{return count(xs);}let a=pass([]);print(a);`, want: map[string]any{"a": float64(0)}, output: "0\n"},
	{name: "generic list preserves reference identity", source: `function pass(xs):integer[]{return xs;}let xs=[1];let ys=pass(xs);ys[0]=2;let a=xs[0];print(xs==ys);print(a);`, want: map[string]any{"a": float64(2)}, output: "true\n2\n"},
	{name: "unknown null result executes once", source: `let count=0;function run(){count=count+1;}function apply(f){let n=f();print(n);}apply(run);print(count);`, want: map[string]any{"count": float64(1)}, output: "null\n1\n"},
	{name: "try skips catch on success", source: `let count=0;try{count=1;}catch(e){count=2;print(e);}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "catch division and retain prefix", source: `let count=0;try{count=1;print("before");let bad=1/0;count=2;}catch(e){print(e);count=count+10;}print(count);`, want: map[string]any{"count": float64(11)}, output: "before\ndivision by zero\n11\n"},
	{name: "catch undefined", source: `let count=0;try{let x:integer;print(x);count=1;}catch(e){count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "catch null list", source: `let count=0;let xs:integer[]=null;try{print(xs[0]);count=1;}catch(e){count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "catch bounds", source: `let count=0;let xs=[1];let i=2;try{print(xs[i]);count=1;}catch(e){print(e);count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "list index out of bounds\n2\n"},
	{name: "catch noncallable", source: `let count=0;let x=1;try{x();count=1;}catch(e){count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "catch invalid dynamic operands", source: `let count=0;function add(x):integer{return x+1;}try{let bad=add(true);count=1;}catch(e){count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "catch invalid dynamic condition", source: `let count=0;function choose(x):integer{if(x){count=1;}return 2;}try{choose(1);count=3;}catch(e){count=count+10;}print(count);`, want: map[string]any{"count": float64(10)}, output: "10\n"},
	{name: "typed argument checked before body effects", source: `let count=0;function body(x:integer){count=100;}function pass(x){body(x);}try{pass("bad");count=1;}catch(e){count=count+2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "catch type guard before return cleanup", source: `function checked(x):integer{try{return x;}catch(e){return 7;}return 9;}let a=checked("bad");print(a);`, want: map[string]any{"a": float64(7)}, output: "7\n"},
	{name: "null result is not inferred", source: `let count=0;function checked(x){return x;}try{checked(3);count=1;}catch(e){count=2;}print(count);`, want: map[string]any{"count": float64(2)}, output: "2\n"},
	{name: "typed null through unknown null result", source: `function checked(x){return x;}let s:string=null;let value=checked(s);print(value);`, want: map[string]any{"value": nil}, output: "null\n"},
	{name: "unknown equality rejects incompatible actual tags", source: `function same(x,y):boolean{return x==y;}let count=0;try{print(same(1,"x"));}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "catch fault reaches outer handler", source: `let count=0;try{try{let bad=1/0;}catch(inner){count=1;let bad=2/0;count=2;}}catch(outer){count=count+10;}print(count);`, want: map[string]any{"count": float64(11)}, output: "11\n"},
	{name: "callee unwinds caller continuation", source: `let count=0;function fail(){count=1;let bad=1/0;count=2;}try{fail();count=3;}catch(e){count=count+10;}print(count);`, want: map[string]any{"count": float64(11)}, output: "11\n"},
	{name: "recursive unwind", source: `let count=0;function fail(n:integer){count=count+1;if(n==0){let bad=1/0;}else{fail(n-1);}count=100;}try{fail(3);}catch(e){count=count+10;}print(count);`, want: map[string]any{"count": float64(14)}, output: "14\n"},
	{name: "return removes departed handler", source: `let count=0;function done():integer{try{return 1;}catch(e){count=100;}return 2;}try{let a=done();let bad=1/0;}catch(e){count=count+1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "break removes crossed handler", source: `let count=0;try{while(true){try{break;}catch(e){count=100;}}let bad=1/0;}catch(e){count=count+1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "continue removes crossed handler", source: `let count=0;let n=0;try{while(n<2){n=n+1;try{continue;}catch(e){count=100;}}let bad=1/0;}catch(e){count=count+1;}print(count);`, want: map[string]any{"n": float64(2), "count": float64(1)}, output: "1\n"},
	{name: "break within try keeps handler", source: `let count=0;try{while(true){break;}let bad=1/0;}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "catch exception reference equality", source: `try{let bad=1/0;}catch(e){let alias=e;print(e==alias);print(e);}`, output: "true\ndivision by zero\n"},
	{name: "catch binding retained by escaped closure", source: `function zero():boolean{return false;}let read=zero;try{let bad=1/0;}catch(e){function same():boolean{return e==e;}read=same;}let a=read();print(a);`, want: map[string]any{"a": true}, output: "true\n"},
	{name: "abandoned callee retains escaped cells", source: `function zero():integer{return 0;}let read=zero;function fail(){let n=2;function saved():integer{return n;}read=saved;n=3;let bad=1/0;}try{fail();}catch(e){}let a=read();print(a);`, want: map[string]any{"a": float64(3)}, output: "3\n"},
	{name: "closure does not inherit defining handler", source: `let count=0;function zero():integer{return 0;}let run=zero;try{function fail():integer{let bad=1/0;return 0;}run=fail;}catch(e){count=100;}try{run();}catch(e){count=count+1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "old typed behavior", source: `function add(x:integer):integer{return x+1;}let a=add(2);let xs=[a];let same=xs==xs;print(a);print(same);`, want: map[string]any{"a": float64(3), "same": true}, output: "3\ntrue\n"},
	{name: "function reference equality", source: `function f():integer{return 1;}function g():integer{return 1;}function same(x,y):boolean{return x==y;}let alias=f;let fs=[f];print(same(f,alias));print(same(f,g));print(same(fs[0],f));`, output: "true\nfalse\ntrue\n"},
	{name: "distinct caught exception identities", source: `try{let bad=1/0;}catch(first){try{let bad=1/0;}catch(second){print(first==second);print(first==first);}}`, output: "false\ntrue\n"},
	{name: "unknown unary and relational operations", source: `function work(x,flag):float{print(!flag);print(x<3);return -x/2;}let a=work(2,false);print(a);`, want: map[string]any{"a": float64(-1)}, output: "true\ntrue\n-1\n"},
	{name: "unknown short circuit suppresses fault", source: `function work(x):boolean{return x&&(1/0>0);}let a=work(false);print(a);`, want: map[string]any{"a": false}, output: "false\n"},
	{name: "unknown heterogeneous ternary", source: `function choose(flag){let value=flag?2:"text";print(value);}choose(true);choose(false);`, output: "2\ntext\n"},
	{name: "unknown invalid member", source: `function read(x):integer{return x.n;}let count=0;try{read(1);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown missing member", source: `class A{}function read(x):integer{return x.n;}let count=0;try{read(new A());}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown invalid index receiver", source: `function read(x):integer{return x[0];}let count=0;try{read(1);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown invalid index value", source: `function read(x,i):integer{return x[i];}let count=0;try{read([1],true);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown invalid foreach", source: `function read(x){foreach(v in x){print(v);}}let count=0;try{read(1);}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown member mutation and bound method", source: `class A{let n:integer=1;function add(x):integer{n=n+x;return n;}}function update(x):integer{x.n=3;return x.add(2);}let box=new A();let a=update(box);print(a);print(box.n);`, want: map[string]any{"a": float64(5)}, output: "5\n5\n"},
	{name: "untyped constructor parameter", source: `class A{let n:integer;function constructor(x){n=x;}}let box=new A(4);print(box.n);`, output: "4\n"},
	{name: "dynamic field assignment yields widened stored value", source: `class A{let n:float=0;}function integerOnly(x:integer){print("wrong");}function set(box){let value=(box.n=1);integerOnly(value);}let count=0;let box=new A();try{set(box);}catch(e){count=1;}print(box.n);print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n1\n"},
	{name: "unknown index mutation preserves alias", source: `function put(xs,value){xs[0]=value;}let xs=[1];put(xs,4);print(xs[0]);`, output: "4\n"},
	{name: "unknown index mutation enforces allocation type", source: `function put(xs,value){xs[0]=value;}let xs=[1];let count=0;try{put(xs,true);}catch(e){count=1;}print(xs[0]);print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n1\n"},
	{name: "unknown nominal boundary before effects", source: `class A{}let count=0;function body(x:A){count=100;}function pass(x){body(x);}{class A{}try{pass(new A());}catch(e){count=count+1;}}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown full signature retained through storage", source: `let count=0;function a(x:integer):integer{return x;}function b(x:string):integer{count=100;return 2;}let selected=a;function replace(x){selected=x;}replace(b);try{selected(1);}catch(e){count=count+1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "unknown call checks actual nominal signature", source: `class A{}let count=0;function body(x:A):integer{count=100;return 1;}function apply(f,x):integer{return f(x);}{class A{}try{apply(body,new A());}catch(e){count=count+1;}}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "generic empty list checked view preserves identity", source: `let saved;function keep(xs):integer[]{saved=xs;return xs;}let typed=keep([]);print(saved==typed);let count=0;foreach(v in typed){count=count+1;}print(count);`, want: map[string]any{"count": float64(0)}, output: "true\n0\n"},
	{name: "contextual list rejects wrong unknown contents", source: `function make(x):integer[]{return [x];}let count=0;try{make("bad");}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "foreach switch handler transfers", source: `let count=0;try{foreach(v in [1,2,3]){try{switch(v==2){case true:continue;default:count=count+v;}}catch(e){count=100;}if(v==3){break;}}let bad=1/0;}catch(e){count=count+10;}print(count);`, want: map[string]any{"count": float64(14)}, output: "14\n"},
	{name: "continue inside try retains handler", source: `let count=0;try{let n=0;while(n<2){n=n+1;continue;}let bad=1/0;}catch(e){count=1;}print(count);`, want: map[string]any{"count": float64(1)}, output: "1\n"},
	{name: "old typed covariant alias fault timing", source: `class A{}class B:A{let n:integer=3;}let b=[new B()];let a:A[]=b;a[0]=new A();let effects=0;let x=b[0].n;`, want: map[string]any{"effects": float64(0)}, fault: "nominal object type mismatch"},
}

func TestDynamicExceptionsExampleSource(t *testing.T) {
	source, err := os.ReadFile("../../testdata/compiscript/tac/dynamic-exceptions.cps")
	if err != nil || string(source) != dynamicExceptionsExample {
		t.Fatalf("checked in example differs: %v", err)
	}
}

func assertDynamicExceptionsAccepted(t *testing.T, source string) {
	t.Helper()
	program, parseDiagnostics := frontend.Parse([]byte(source))
	_, semanticDiagnostics := semantic.Analyze(program)
	if len(parseDiagnostics)+len(semanticDiagnostics) != 0 {
		t.Fatalf("not an accepted-source RED fixture: parse=%+v semantic=%+v", parseDiagnostics, semanticDiagnostics)
	}
}

func TestDynamicExceptionsAcceptedSources(t *testing.T) {
	for _, tt := range dynamicExceptionsCases {
		t.Run(tt.name, func(t *testing.T) {
			assertDynamicExceptionsAccepted(t, tt.source)
		})
	}
}

func TestDynamicExceptionsCompile(t *testing.T) {
	for _, tt := range dynamicExceptionsCases {
		t.Run(tt.name, func(t *testing.T) {
			assertDynamicExceptionsAccepted(t, tt.source)
			source := []byte(tt.source)
			program, _ := frontend.Parse(source)
			scopes, _ := semantic.Analyze(program)
			inputs, err := json.Marshal([]any{program, scopes})
			if err != nil {
				t.Fatal(err)
			}
			generated, diagnostics := tac.Generate(program, scopes)
			afterInputs, err := json.Marshal([]any{program, scopes})
			if err != nil || string(inputs) != string(afterInputs) || generated == nil || len(diagnostics) != 0 {
				t.Fatalf("Generate changed or rejected inputs: %v", diagnostics)
			}
			before, err := json.Marshal(Analyze(source))
			if err != nil {
				t.Fatal(err)
			}
			report := Compile(source)
			assertValuesEffects(t, report, tt)
			after, err := json.Marshal(Analyze(source))
			if err != nil || string(before) != string(after) || string(source) != tt.source {
				t.Fatal("source or Analyze changed")
			}
			first, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			again, err := json.Marshal(Compile(source))
			if err != nil || string(first) != string(again) {
				t.Fatal("nondeterministic compilation")
			}
		})
	}
}

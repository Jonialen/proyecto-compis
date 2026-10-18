package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

var collectionCases = []struct {
	name, source string
	want         map[string]any
}{
	{"B1 nested indexed signature under shadowing", `class A{let n:integer=7;}let original=new A();function identity(a:A[][]):A[][]{return a;}let fs=[[identity]];let x=0;{class A{let n:integer=50;}let row=fs[0];let fn=row[0];let alias=fn;let a=alias([[original]]);x=a[0][0].n;}let y=fs[0][0]([[original]])[0][0].n;`, map[string]any{"x": float64(7), "y": float64(7)}},
	{"B1 foreach function signature", `class A{let n:integer=9;}function make():A{return new A();}let fs=[make];let x=0;foreach(fn in fs){let alias=fn;let a=alias();x=x+a.n;}`, map[string]any{"x": float64(9)}},
	{"nominal homogeneous ancestor in either order", `class Base{let n:integer=2;}class Child:Base{}let a=[new Child(),new Base()];let b=[new Base(),new Child()];let x=0;foreach(v in [new Child(),new Base()]){x=x+v.n;}let y=a[1].n+b[1].n;`, map[string]any{"x": float64(4), "y": float64(4)}},
	{"index assignment value and grouping", `let a=[1,2];let x=(a[0]=7);((a[1]=x+1));let y=a[0]+a[1];`, map[string]any{"x": float64(7), "y": float64(15)}},
	{"uncontextual nested numeric inference", `let x:float=0;foreach(row in [[1],[2.5]]){foreach(v in row){x=x+v;}}`, map[string]any{"x": float64(3.5)}},
	{"empty allocations and zero trips", `let a:integer[]=[];let b:integer[]=[];let different=a!=b;let x=0;foreach(v in a){x=x+1;}`, map[string]any{"different": true, "x": float64(0)}},
	{"null list result", `function empty():integer[]{return null;}let a=empty();let x=a==null;`, map[string]any{"x": true}},
	{"list view widening", `let a=[1];let b:float[]=a;function read(a:float[]):float{return a[0];}let x=b[0]+read(a);`, map[string]any{"x": float64(2)}},
	{"read source once and retain reference", `let a=[4];let b=[8];let selected=a;let calls=0;function source():integer[]{calls=calls+1;return selected;}function index():integer{selected=b;return 0;}let x=source()[index()];`, map[string]any{"x": float64(4), "calls": float64(1)}},
	{"indexed for post", `let a=[0];for(;a[0]<3;a[0]=a[0]+1){}let x=a[0];`, map[string]any{"x": float64(3)}},
	{"foreach return and caller state", `function first(a:integer[]):integer{foreach(v in a){return v;}return 0;}let x=first([4])+first([7]);`, map[string]any{"x": float64(11)}},
	{"aliases and identity", `let a=[1,2];let b=a;b[0]=9;let x=a[0];let same=a==b;let other=a!=[9,2];`, map[string]any{"x": float64(9), "same": true, "other": true}},
	{"empty contextual nested ragged", `let a:integer[][]=[[],[1,2],[]];a[0]=[7];let x=a[0][0];let y=a[1][1];let count=0;foreach(row in a){foreach(n in row){count=count+n;}}`, map[string]any{"x": float64(7), "y": float64(2), "count": float64(10)}},
	{"inferred trailing empty", `let a=[[1],[]];let x=a[0][0];let n=0;foreach(row in a){foreach(v in row){n=n+v;}}`, map[string]any{"x": float64(1), "n": float64(1)}},
	{"typed calls results recursion", `function id(a:integer[],n:integer):integer[]{if(n==0)return a;else return id(a,n-1);}let a=[2];let f=id;let b=f(a,3);b[0]=8;let x=a[0];`, map[string]any{"x": float64(8)}},
	{"contextual empty arguments results fields", `function empty():integer[]{return [];}function count(a:integer[]):integer{let n=0;foreach(v in a){n=n+1;}return n;}class Box{let a:integer[]=[];function put(a:integer[]){this.a=a;}}let box=new Box();box.put(empty());let x=count([]);let y=count(box.a);`, map[string]any{"x": float64(0), "y": float64(0)}},
	{"target index RHS frozen", `let a=[1,2];let b=[3,4];let selected=a;let i=0;let trace=0;function index():integer{trace=trace*10+1;selected=b;return i;}function value():integer{trace=trace*10+2;i=1;return 9;}selected[index()]=value();let x=a[0];let y=a[1];let z=b[0];`, map[string]any{"x": float64(9), "y": float64(2), "z": float64(3), "trace": float64(12)}},
	{"index symbol frozen across RHS", `let a=[1,2];let i=0;function change():integer{i=1;return 7;}a[i]=change();let x=a[0];let y=a[1];`, map[string]any{"x": float64(7), "y": float64(2)}},
	{"literal elements source order", `let n=1;function change():integer{n=9;return 2;}let a=[n,change(),n];let x=a[0];let y=a[1];let z=a[2];`, map[string]any{"x": float64(1), "y": float64(2), "z": float64(9)}},
	{"foreach reference retained mutable binding", `let a=[1,2,3];let original=a;let calls=0;function source():integer[]{calls=calls+1;return a;}let sum=0;foreach(v in source()){a=[9];v=v+10;sum=sum+v;original[1]=4;}let x=original[1];`, map[string]any{"sum": float64(38), "calls": float64(1), "x": float64(4)}},
	{"fresh captured iteration cells", `function zero():integer{return 0;}let first=zero;let second=zero;foreach(v in [2,5]){function next():integer{v=v+1;return v;}if(v==2)first=next;else second=next;}let x=first();let y=second();let z=first();`, map[string]any{"x": float64(3), "y": float64(6), "z": float64(4)}},
	{"nested transfer and post", `let total=0;let posts=0;for(let i=0;i<2;i=i+1){foreach(v in [1,2,3]){if(v==2)continue;foreach(w in [4,5]){if(w==5)break;total=total+v*w;}if(v==3)break;}posts=posts+1;}let x=total;`, map[string]any{"x": float64(32), "posts": float64(2)}},
	{"foreach switch continue", `let x=0;foreach(v in [1,2,3]){switch(v==2){case true:continue;default:x=x+v;}x=x+10;}`, map[string]any{"x": float64(24)}},
	{"dynamic allocation and caller temporaries", `function make(n:integer):integer[]{return [n];}let a=make(1);let b=make(2);a[0]=7;let distinct=a!=b;function sum(n:integer):integer{let a=[n];if(n==0)return a[0];else return a[0]+sum(n-1)+a[0];}let x=sum(3);let y=b[0];`, map[string]any{"distinct": true, "x": float64(12), "y": float64(2)}},
	{"nominal list origins under shadowing", `class A{let n:integer=3;function read():integer{return n;}}let a=[new A()];function get():A[]{return a;}let f=get;let x=0;{class A{let n:integer=20;function read():integer{return n;}}let b=a;let c=f();b[0].n=8;x=c[0].read();}let y=a[0].n;`, map[string]any{"x": float64(8), "y": float64(8)}},
	{"object list fields and iteration", `class A{let n:integer=3;}class Box{let a:A[];function constructor(a:A[]){this.a=a;}function read():integer{return a[0].n;}}let box=new Box([new A()]);foreach(v in box.a){v.n=7;}let x=box.read();`, map[string]any{"x": float64(7)}},
	{"float list widening and null", `let a:float[]=[1,2.5];a[0]=3;let x=a[0]+a[1];let b:integer[]=null;let absent=b==null;`, map[string]any{"x": float64(5.5), "absent": true}},
	{"preserve numeric closures objects loops", `let n=0;function bump():integer{n=n+1;return n;}let f=bump;class Box{let n:integer=4;function read():integer{return n;}}let b=new Box();for(let i=0;i<2;i=i+1){n=n+b.read();}let x=f();`, map[string]any{"x": float64(9)}},
}

func TestCollectionExecution(t *testing.T) {
	for _, tt := range collectionCases {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			analysis := Analyze(input)
			r := Compile(input)
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("Compile: %+v", r.Diagnostics)
			}
			values, err := runFunctionFrames(r.TAC)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range tt.want {
				if values["s1."+name] != want {
					t.Errorf("%s=%v want %v", name, values["s1."+name], want)
				}
			}
			if !bytes.Equal(input, before) || !reflect.DeepEqual(analysis.Scopes, r.Scopes) || !reflect.DeepEqual(r, Compile(input)) {
				t.Fatal("mutated input/scopes or nondeterministic compile")
			}
			program, pd := frontend.Parse(input)
			scopes, sd := semantic.Analyze(program)
			snapshot, _ := json.Marshal([]any{program, scopes})
			p, ds := tac.Generate(program, scopes)
			after, _ := json.Marshal([]any{program, scopes})
			if len(pd)+len(sd)+len(ds) != 0 || !reflect.DeepEqual(p, r.TAC) || !bytes.Equal(snapshot, after) {
				t.Fatal("Generate changed frontend snapshots or output")
			}
		})
	}
}

var collectionRejections = []string{
	`const a=[1];a[0]=2;`, `let a=[1];a[0]=true;`, `let a=[1];let x=a[1];`,
	`let a=[1];let x=a[-1];`, `let a=[1];let x=a[true];`, `let a=[];`,
	`let a=[1,"x"];`, `foreach(v in 1){}`, `foreach(v in [1]){}let x=v;`,
	`let a=[1];let x=a.length;`, `let a=[1];if(false){let x=1%2;}`,
	`let a=[1];function bad(){print(a[0]);}`,
}

func TestCollectionAtomicRejection(t *testing.T) {
	for _, source := range collectionRejections {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := append([]byte(nil), input...)
			a := Analyze(input)
			r := Compile(input)
			if r.TAC != nil || len(r.Diagnostics) == 0 || !bytes.Equal(input, before) || !reflect.DeepEqual(a.Scopes, r.Scopes) || !reflect.DeepEqual(r, Compile(input)) {
				t.Fatalf("non-atomic rejection: %+v", r)
			}
		})
	}
}

var collectionFaults = []struct {
	name, source, message string
	effects               map[string]any
}{
	{"B1 indexed nominal mismatch before effects", `let effects=0;class A{}function make():A{return new A();}let fs=[make];{class A{}function other():A{effects=effects+1;return new A();}fs[0]=other;}let f=fs[0];let a=f();`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
	{"B1 indexed nested shape mismatch before effects", `let effects=0;class A{}function make(a:A[][]):A[][]{return a;}function other(a:A[]):A[][]{effects=effects+1;return [a];}let fs=[make];fs[0]=other;let f=fs[0];let a=f([[new A()]]);`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
	{"upper dynamic bound", `let a=[1];let i=1;let effects=0;function next():integer{effects=effects+1;return 7;}a[i]=next();`, "list index out of bounds", map[string]any{"effects": float64(1)}},
	{"negative dynamic bound", `let a=[1];let i=0-1;let x=a[i];`, "list index out of bounds", nil},
	{"null iteration", `let a:integer[]=null;let effects=0;foreach(v in a){effects=effects+1;}`, "null list", map[string]any{"effects": float64(0)}},
	{"null indexing", `let a:integer[]=null;let i=0;let x=a[i];`, "null list", nil},
	{"indirect element kind", `let effects=0;function a(x:integer[]):integer{return 1;}function b(x:boolean[]):integer{effects=effects+1;return 2;}let f=a;f=b;let x=f([1]);`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
	{"indirect nested shape", `let effects=0;function a(x:integer[][]):integer{return 1;}function b(x:integer[]):integer{effects=effects+1;return 2;}let f=a;f=b;let x=f([[1]]);`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
	{"indirect result shape", `let effects=0;function a():integer[]{return [1];}function b():boolean[]{effects=effects+1;return [true];}let f=a;f=b;let x=f();`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
	{"indirect nominal list", `let effects=0;class A{}function a(x:A[]):integer{return 1;}let f=a;{class A{}function b(x:A[]):integer{effects=effects+1;return 2;}f=b;}let x=f([new A()]);`, "indirect call signature mismatch", map[string]any{"effects": float64(0)}},
}

func TestCollectionRuntimeGuards(t *testing.T) {
	for _, tt := range collectionFaults {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, tt.source)
			values, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("values=%v err=%v want %s", values, err, tt.message)
			}
			for name, want := range tt.effects {
				if values["s1."+name] != want {
					t.Fatalf("effects %s=%v want %v", name, values["s1."+name], want)
				}
			}
		})
	}
}

func TestCollectionNominalStorageGuards(t *testing.T) {
	for _, source := range []string{
		`class A{}let a=[new A()];{class A{}let b:A[]=a;}`,
		`class A{}let a=[new A()];{class A{}a[0]=new A();}`,
		`class A{}let a=[[new A()]];{class A{}let b:A[][]=a;}`,
	} {
		t.Run(source, func(t *testing.T) {
			r := Compile([]byte(source))
			if r.TAC != nil || len(r.Diagnostics) == 0 {
				t.Fatalf("nominal storage weakened: %+v", r)
			}
		})
	}
}

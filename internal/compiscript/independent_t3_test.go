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
	"reflect"
	"testing"
)

// S4/S5/S6: independently derived compositions, using public compilation and CLI JSON.
func TestIndependentT3Values(t *testing.T) {
	cli := buildClosureCLI(t)
	cases := []struct {
		name, source string
		want         map[string]any
	}{
		{"string and boolean lists", `let a=["a","b"];let flags=[true,false];let x="";foreach(v in a){x=x+v;}let y=flags[0]&&!flags[1];`, map[string]any{"x": "ab", "y": true}},
		{"closure list elements", `let n=1;function f(x:integer):integer{n=n+x;return n;}let fs=[f,f];let g=fs[1];let x=g(3);let y=fs[0](2);`, map[string]any{"x": float64(4), "y": float64(6)}},
		{"function list nominal result", `class A{let n:integer=7;}function make():A[]{return [new A()];}let fs=[make];let x=0;{class A{let n:integer=50;}let g=fs[0];let a=g();x=a[0].n;}`, map[string]any{"x": float64(7)}},
		{"leading empty inferred ragged", `let a=[[],[1,2]];let x=0;foreach(row in a){foreach(v in row){x=x+v;}}`, map[string]any{"x": float64(3)}},
		{"deep contextual empties", `function make():integer[][][]{return [[[]],[],[[3,4]]];}let a=make();a[0][0]=[7];let x=0;foreach(plane in a){foreach(row in plane){foreach(n in row){x=x+n;}}}`, map[string]any{"x": float64(14)}},
		{"nested alias replacement", `let row=[2,3];let a=[row,row];a[0][1]=8;let x=a[1][1];a[0]=[9];let y=row[0]+a[0][0]+a[1][0];`, map[string]any{"x": float64(8), "y": float64(13)}},
		{"target index rhs all effectful", `let a=[[1],[2]];let saved=a;let trace=0;let i=0;function source():integer[][]{trace=trace*10+1;return a;}function index():integer{trace=trace*10+2;a=[[9]];return i;}function rhs():integer[]{trace=trace*10+3;i=1;return [7];}source()[index()]=rhs();let x=saved[0][0];let y=saved[1][0];`, map[string]any{"x": float64(7), "y": float64(2), "trace": float64(123)}},
		{"nested read retains inner target", `let a=[[4]];let old=a;function mutate():integer{a[0]=[9];return 0;}let x=a[0][mutate()];let y=old[0][0];`, map[string]any{"x": float64(4), "y": float64(9)}},
		{"argument freeze and alias effects", `let a=[1];function replace():integer{a=[7];return 2;}function pair(x:integer[],n:integer):integer{x[0]=x[0]+n;return x[0];}let old=a;let x=pair(a,replace());let y=a[0]+old[0];`, map[string]any{"x": float64(3), "y": float64(10)}},
		{"field result reference", `class Box{let rows:integer[][]=[[]];function put(x:integer[][]):integer[][]{rows=x;return rows;}}let b=new Box();let a=[[5]];let c=b.put(a);c[0][0]=8;let x=b.rows[0][0]+a[0][0];`, map[string]any{"x": float64(16)}},
		{"foreach retained row and list mutation", `let a=[[1],[2],[3]];let saved=a;let x=0;foreach(row in a){a=[[99]];saved[2]=[8];row[0]=row[0]+10;x=x+row[0];}let y=saved[2][0];`, map[string]any{"x": float64(41), "y": float64(8)}},
		{"fresh iteration and nested captures", `function zero():integer{return 0;}let f=zero;let g=zero;foreach(row in [[1],[4]]){let offset=row[0];function next():integer{offset=offset+row[0];return offset;}if(offset==1)f=next;else g=next;}let x=f()+g()+f();`, map[string]any{"x": float64(13)}},
		{"reentrant foreach temporaries", `function sum(n:integer):integer{let s=0;foreach(v in [n,n+1]){if(n>0)s=s+sum(n-1);s=s+v;}return s;}let x=sum(3);`, map[string]any{"x": float64(37)}},
		{"mixed transfer stacks", `let x=0;for(let i=0;i<2;i=i+1){foreach(v in [1,2,3]){switch(v==2){case true:continue;default:x=x+1;}let j=0;while(j<2){j=j+1;if(j==1)continue;x=x+v;break;}if(v==3)break;}x=x+10;}`, map[string]any{"x": float64(32)}},
		{"nested nominal origin", `class A{let n:integer=6;function read():integer{return n;}}let rows=[[new A()]];function get():A[][]{return rows;}let f=get;let x=0;{class A{let n:integer=50;function read():integer{return n;}}let copy=rows;let other=f();copy[0][0].n=9;foreach(row in other){foreach(v in row){x=x+v.read();}}}`, map[string]any{"x": float64(9)}},
		{"nested class field origin", `class A{let n:integer=7;}class Holder{let rows:A[][]=[[new A()]];function get():A[][]{return rows;}}let box=new Holder();let x=0;{class A{let n:integer=50;}let rows=box.get();x=rows[0][0].n;}`, map[string]any{"x": float64(7)}},
		{"empty foreach no body effects", `let a:integer[][]=[[],[]];let x=0;foreach(row in a){foreach(v in row){x=x+100;}x=x+1;}let b:integer[]=[];let different=b!=a[0];`, map[string]any{"x": float64(2), "different": true}},
		{"nested widened read", `let a=[[2]];let b:float[][]=a;function read(x:float[][]):float{return x[0][0];}let x=read(b)+b[0][0];`, map[string]any{"x": float64(4)}},
		{"recursive literal element liveness", `function build(n:integer):integer[]{if(n==0)return [1,2];let a=build(n-1);return [a[0]+n,build(n-1)[1],a[1]+n];}let a=build(3);let x=a[0]+a[1]+a[2];`, map[string]any{"x": float64(14)}},
		{"list null element", `class A{let n:integer=1;}let a:A[]=[null,new A()];let x=a[0]==null;let y=a[1].n;`, map[string]any{"x": true, "y": float64(1)}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			hash := sha256.Sum256(input)
			a := Analyze(input)
			if len(a.Diagnostics) != 0 {
				t.Fatalf("Analyze: %+v", a.Diagnostics)
			}
			r := Compile(input)
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("Compile: %+v", r.Diagnostics)
			}
			got, err := runFunctionFrames(r.TAC)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.want {
				if got["s1."+k] != v {
					t.Errorf("%s=%v want %v", k, got["s1."+k], v)
				}
			}
			if hash != sha256.Sum256(input) || !reflect.DeepEqual(a.Scopes, r.Scopes) || !reflect.DeepEqual(r, Compile(input)) {
				t.Fatal("mutation/nondeterminism")
			}
			tree, pd := frontend.Parse(input)
			scopes, sd := semantic.Analyze(tree)
			before, _ := json.Marshal([]any{tree, scopes})
			ir, ds := tac.Generate(tree, scopes)
			after, _ := json.Marshal([]any{tree, scopes})
			if len(pd)+len(sd)+len(ds) != 0 || !bytes.Equal(before, after) || !reflect.DeepEqual(ir, r.TAC) {
				t.Fatal("AST/scopes mutation or nonidentical Generate")
			}
			assertClosureCLI(t, cli, tt.source, tt.want)
		})
	}
}

// S4/S5/S6: semantic rejection remains atomic, including skipped code.
func TestIndependentT3Rejections(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, source := range []string{
		`const a=[[1]];a[0][0]=2;`,
		`let a:integer[][]=[[]];a[0]=[true];`,
		`let a=[1,2];let x=a[2];`,
		`let a=[1];let x=a[0.5];`,
		`let a:integer[]=[];foreach(v in a){}let x=v;`,
		`let a=[1];if(false){let x=a.length;}`,
		`let a=[1];if(false){try{print(a[missing]);}catch(e){}}`,
		`class A{}let a=[[new A()]];{class A{}let b:A[][]=a;}`,
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			h := sha256.Sum256(input)
			a := Analyze(input)
			r := Compile(input)
			if r.TAC != nil || len(r.Diagnostics) == 0 || h != sha256.Sum256(input) || !reflect.DeepEqual(a.Scopes, r.Scopes) {
				t.Fatalf("non-atomic: %+v", r)
			}
			if len(a.Diagnostics) == 0 {
				tree, _ := frontend.Parse(input)
				scopes, _ := semantic.Analyze(tree)
				before, _ := json.Marshal([]any{tree, scopes})
				p, ds := tac.Generate(tree, scopes)
				after, _ := json.Marshal([]any{tree, scopes})
				if p != nil || len(ds) == 0 || !bytes.Equal(before, after) {
					t.Fatalf("Generate rejection/mutation: %v %v", p, ds)
				}
			}
			assertClosureCLIRejection(t, cli, source)
		})
	}
}

// S4/S6: these are oracle runtime faults, not Analyze restrictions.
func TestIndependentT3Faults(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct {
		name, source, err string
		effects           float64
	}{
		{"dynamic upper after rhs", `let a=[1];let i=1;let effects=0;function rhs():integer{effects=effects+1;return 8;}a[i]=rhs();`, "list index out of bounds", 1},
		{"negative nested", `let a=[[1]];let i=0-1;let effects=0;let x=a[0][i];`, "list index out of bounds", 0},
		{"null nested iteration", `let a:integer[][]=[null];let effects=0;foreach(row in a){foreach(v in row){effects=effects+1;}}`, "null list", 0},
		{"indirect result rank", `let effects=0;function a():integer[][]{return [[1]];}function b():integer[]{effects=effects+1;return [2];}let f=a;f=b;let x=f();`, "indirect call signature mismatch", 0},
		{"indirect nominal nested param", `let effects=0;class A{}function a(x:A[][]):integer{return 1;}let f=a;{class A{}function b(x:A[][]):integer{effects=effects+1;return 2;}f=b;}let x=f([[new A()]]);`, "indirect call signature mismatch", 0},
		{"covariant alias guard", `class A{}class B:A{let n:integer=3;}let b=[new B()];let a:A[]=b;a[0]=new A();let effects=0;let x=b[0].n;`, "nominal object type mismatch", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := Analyze([]byte(tt.source))
			r := Compile([]byte(tt.source))
			if len(a.Diagnostics) != 0 || r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("compile: %v %v", a.Diagnostics, r.Diagnostics)
			}
			values, err := runFunctionFrames(r.TAC)
			if err == nil || err.Error() != tt.err || values["s1.effects"] != tt.effects {
				t.Fatalf("values=%v err=%v want %q effects=%v", values, err, tt.err, tt.effects)
			}
			path := filepath.Join(t.TempDir(), "fault.cps")
			if err := os.WriteFile(path, []byte(tt.source), 0600); err != nil {
				t.Fatal(err)
			}
			before := sha256.Sum256([]byte(tt.source))
			cmd := exec.Command(cli, "-format=json", path)
			var out, stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("exit=%v stderr=%q", err, stderr.String())
			}
			encoded, _ := json.MarshalIndent(r, "", "  ")
			if out.String() != string(encoded)+"\n" {
				t.Fatal("CLI JSON mismatch")
			}
			var report CompilationReport
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			values, err = runFunctionFrames(report.TAC)
			if err == nil || err.Error() != tt.err || values["s1.effects"] != tt.effects {
				t.Fatalf("CLI JSON values=%v err=%v", values, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(after) != before {
				t.Fatal("source mutation")
			}

		})
	}
}

// S4/S5/S6 B1: nominal signatures must survive function-valued list elements.
func TestIndependentT3NominalFunctionList(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range []struct{ name, source string }{
		{"control direct alias", `class A{}function make():A{return new A();}let f=make;let a=f();`},
		{"minimal extracted alias", `class A{}function make():A{return new A();}let fs=[make];let f=fs[0];let a=f();`},
		{"nominal parameter", `class A{}function take(a:A):integer{return 7;}let fs=[take];let f=fs[0];let x=f(new A());`},
		{"nested result", `class A{}function make():A[][]{return [[new A()]];}let fs=[make];let f=fs[0];let a=f();`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			h := sha256.Sum256(input)
			a := Analyze(input)
			r := Compile(input)
			if len(a.Diagnostics) != 0 || len(r.Diagnostics) != 0 || r.TAC == nil {
				t.Fatalf("Analyze=%v Compile=%v", a.Diagnostics, r.Diagnostics)
			}
			if h != sha256.Sum256(input) {
				t.Fatal("source mutation")
			}
			t.Run("Compile execution", func(t *testing.T) {
				if values, err := runFunctionFrames(r.TAC); err != nil {
					t.Fatalf("values=%v fault=%v", values, err)
				}
			})
			t.Run("CLI JSON execution", func(t *testing.T) { assertClosureCLI(t, cli, tt.source, nil) })
		})
	}
}

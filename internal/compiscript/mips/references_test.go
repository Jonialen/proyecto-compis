package mips_test

import (
	"strings"
	"testing"

	"genanalex/internal/compiscript"
	"genanalex/internal/compiscript/mips"
	"genanalex/internal/compiscript/tac"
)

func TestMIPSReferences(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"objects and overrides", `class Base{let n:integer=2;function read():integer{return n;}}class Child:Base{function read():integer{return this.n+8;}}function id(x:Base):Base{return x;}let a=id(new Child());let b=a;b.n=4;print(a.read());print(a==b);print(a);`, "12\ntrue\n<object s1.Child>\n"},
		{"retained local class", `function zero():integer{return 0;}let saved=zero;function install(n:integer){class Local{let value:integer=n;function read():integer{return this.value+n;}}saved=new Local().read;}install(3);let old=saved;install(8);print(old());print(saved());`, "6\n16\n"},
		{"nested lists and aliases", `let a:integer[][]=[[],[1,2],[]];a[0]=[7];let alias=a;alias[1][0]=8;print(a);let total=0;foreach(row in a){foreach(v in row){total=total+v;}}print(total);print(a==alias);print(a!=[[],[8,2],[]]);`, "[[7], [8, 2], []]\n17\ntrue\ntrue\n"},
		{"widened list views", `let a=[1];let b:float[]=a;function read(xs:float[]):float{return xs[0];}print(b[0]+read(a));print(a);`, "2\n[1]\n"},
		{"all initializers before inherited constructor", `class Base{let first:integer=1;function constructor(){this.set();}function set(){}}class Child:Base{let later:integer=7;function set(){this.later=9;}}print(new Child().later);let trace=0;function mark(n:integer):integer{trace=trace*10+n;return n;}class A{let a:integer=mark(1);function constructor(x:integer){mark(8);}}class B:A{let b:integer=mark(2);function constructor(x:integer){mark(x);}}let b=new B(mark(3));print(trace);`, "9\n3123\n"},
		{"base and derived retained environments", `let value=5;class Base{function base():integer{return value;}}function zero():integer{return 0;}let saved=zero;function install(n:integer){class Child:Base{function read():integer{return this.base()+n;}}saved=new Child().read;}install(3);let old=saved;install(8);print(old());print(saved());`, "8\n13\n"},
		{"bound tuple identity", `class A{function f():integer{return 1;}}function same(x,y):boolean{return x==y;}let a=new A();let b=new A();print(same(a.f,a.f));print(same(a.f,b.f));`, "true\nfalse\n"},
		{"frozen collection evaluations", `let n=1;function change():integer{n=9;return 2;}print([n,change(),n]);let a=[1,2];let b=[3,4];let selected=a;let i=0;function index():integer{selected=b;return i;}function value():integer{i=1;return 9;}selected[index()]=value();print(a);print(b);`, "[1, 2, 9]\n[9, 2]\n[3, 4]\n"},
		{"bounded print budget", "print([" + strings.Repeat("0,", 69) + "0]);", "[" + strings.Repeat("0, ", 62) + "0]\n"},
		{"bounded print depth", `print([[[[[[[[[1]]]]]]]]]);`, "[[[[[[[[...]]]]]]]]\n"},
		{"print bytes and functions", `function f():integer{return 1;}class A{}print(["á🌎","a\u0000b"]);print(f);print(new A());`, "[á🌎, a\x00b]\n<function s1.f>\n<object s1.A>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertReferenceExecution(t, tt.source, tt.want, 0)
		})
	}
}

// The helper still executes public source -> TAC -> assembly in real MARS.
func assertReferenceExecution(t *testing.T, source, want string, wantCode int) {
	t.Helper()
	out, stderr, code := execute(t, source)
	if out != want || stderr != "\n" || code != wantCode {
		t.Fatalf("stdout=%q stderr=%q exit=%d want=%q exit=%d", out, stderr, code, want, wantCode)
	}
}

func TestMIPSReferencesMalformed(t *testing.T) {
	for _, tt := range []struct {
		name string
		mutate func(*tac.Program)
	}{
		{"undefined parent", func(p *tac.Program) { p.Classes[0].Parent = "missing" }},
		{"cycle", func(p *tac.Program) { p.Classes[0].Parent = p.Classes[0].ID }},
		{"field offset", func(p *tac.Program) { p.Classes[0].Fields[0].Offset = 99 }},
		{"method", func(p *tac.Program) { p.Classes[0].Methods[0].Callee = "missing" }},
		{"bound member", func(p *tac.Program) {
			for n := range p.Instructions {
				if p.Instructions[n].Op == tac.BindMethod && p.Instructions[n].Callee == "" {
					p.Instructions[n].Member = "missing"
				}
			}
		}},
		{"instruction offset", func(p *tac.Program) {
			for n := range p.Instructions {
				if p.Instructions[n].Op == tac.LoadField {
					p.Instructions[n].Offset = nil
				}
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := compiscript.Compile([]byte(`class A{let n:integer=1;function read():integer{return n;}}let a=new A();print(a.n);print(a.read());`))
			if r.TAC == nil {
				t.Fatal(r.Diagnostics)
			}
			tt.mutate(r.TAC)
			assembly, err := mips.Generate(r.TAC)
			if assembly != "" || err == nil {
				t.Fatalf("non atomic: assembly bytes=%d err=%v", len(assembly), err)
			}
		})
	}
}

func TestMIPSReferencesUnusedMetadata(t *testing.T) {
	r := compiscript.Compile([]byte(`class Unused{}print(1);`))
	if r.TAC == nil {
		t.Fatal(r.Diagnostics)
	}
	r.TAC.Classes[0].Initializer = "missing"
	assembly, err := mips.Generate(r.TAC)
	if assembly != "" || err == nil {
		t.Fatalf("unused malformed class: assembly bytes=%d err=%v", len(assembly), err)
	}
}

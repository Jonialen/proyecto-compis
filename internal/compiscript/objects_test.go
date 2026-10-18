package compiscript

import (
	"bytes"
	"encoding/json"
	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
	"reflect"
	"strings"
	"testing"
)

const objectInitializerOrder = `class Base { let first:integer=1; function constructor(){this.set();} function set(){} } class Child:Base { let later:integer=7; function set(){this.later=9;} } let child=new Child(); let observed=child.later;`

var objectCases = []struct {
	name, source string
	want         map[string]any
}{
	{"unqualified object field method", `class Box{function read():integer{return 7;}}class Holder{let value:Box;function constructor(v:Box){this.value=v;}function read():integer{return value.read();}}let holder=new Holder(new Box());let x=holder.read();let y=holder.value.read();`, map[string]any{"x": float64(7), "y": float64(7)}},
	{"shadowed inferred nominal alias minimal", `class A{}let a=new A();{class A{}let b=a;}`, map[string]any{}},
	{"shadowed inferred nominal origins", `class A{let n:integer=1;function read():integer{return this.n;}}class Holder{let value:A;function constructor(value:A){this.value=value;}}let a=new A();let holder=new Holder(a);function get():A{return a;}let factory=get;let x=0;let y=0;let z=0;let distinct=false;{class A{let n:integer=20;function read():integer{return this.n;}}let local=new A();let b=a;const c=holder.value;let d=get();let e=factory();b.n=9;x=c.read();y=d.read();z=e.read();distinct=b!=local;}let original=a.n;`, map[string]any{"x": float64(9), "y": float64(9), "z": float64(9), "original": float64(9), "distinct": true}},
	{"string storage calls and comparison", `function greet(name:string):string{return "Hi "+name;}let text=greet("Ada");let yes=text=="Hi Ada";let no=text!="Hi Ada";`, map[string]any{"text": "Hi Ada", "yes": true, "no": false}},
	{"all initializers before inherited constructor", objectInitializerOrder, map[string]any{"observed": float64(9)}},
	{"identity and shared fields", `class Box{let n:integer; function constructor(n:integer){this.n=n;}} let a=new Box(2); let b=new Box(3); let alias=a; alias.n=8; let x=a.n; let y=b.n; let same=a==alias; let different=a!=b;`, map[string]any{"x": float64(8), "y": float64(3), "same": true, "different": true}},
	{"subtype override strings", `class Animal{let name:string; function constructor(name:string){this.name=name;} function sound():string{return "noise";} function describe():string{return this.name+":"+this.sound();}} class Dog:Animal{function sound():string{return "woof";}} function identity(a:Animal):Animal{return a;} let dog:Animal=identity(new Dog("Rex")); let text=dog.describe(); let equal=text=="Rex:woof";`, map[string]any{"text": "Rex:woof", "equal": true}},
	{"unqualified fields and initializers", `class Box{let a:integer=2; let b:integer=a+3; function bump():integer{a=a+1;return a+b;} } let box=new Box(); let x=box.bump(); let y=box.a;`, map[string]any{"x": float64(8), "y": float64(3)}},
	{"constructor effects and no automatic ancestors", `let trace=0; function mark(n:integer):integer{trace=trace*10+n;return n;} class Base{let a:integer=mark(1); function constructor(n:integer){trace=trace*10+8;}} class Child:Base{let b:integer=mark(2); function constructor(n:integer){trace=trace*10+n;}} let child=new Child(mark(3)); let observed=trace;`, map[string]any{"observed": float64(3123)}},
	{"escaped this and unqualified field", `function zero():integer{return 0;} let saved=zero; class Box{let n:integer=4; function install(){function next():integer{this.n=this.n+1;return n;} saved=next;}} function install(){let box=new Box();box.install();} install();let x=saved();let y=saved();`, map[string]any{"x": float64(5), "y": float64(6)}},
	{"bound receiver selected before arguments", `class Box{let n:integer;function constructor(n:integer){this.n=n;}function add(v:integer):integer{return this.n+v;}} let a=new Box(10);let b=new Box(20);let chosen=a;let method=a.add;function change():integer{chosen=b;method=b.add;return 1;}let x=chosen.add(change());let y=method(change());let z=method(2);`, map[string]any{"x": float64(11), "y": float64(21), "z": float64(22)}},
	{"field receiver survives RHS", `class Box{let n:integer=0;}let a=new Box();let b=new Box();let chosen=a;function change():integer{chosen=b;return 9;}chosen.n=change();let x=a.n;let y=b.n;`, map[string]any{"x": float64(9), "y": float64(0)}},
	{"receiver once and recursive typed result", `class Box{let n:integer=3;}let box=new Box();let calls=0;function get():Box{calls=calls+1;return box;}function descend(b:Box,n:integer):Box{if(n==0)return b;else return descend(b,n-1);}get().n=get().n+1;let answer=descend(box,4);let x=answer.n;`, map[string]any{"x": float64(4), "calls": float64(2)}},
	{"local class escaped method", `function zero():integer{return 0;}let saved=zero;function install(n:integer){class Local{let value:integer=n;function read():integer{return this.value+n;}}let local=new Local();saved=local.read;}install(3);let old=saved;install(8);let x=old();let y=saved();`, map[string]any{"x": float64(6), "y": float64(16)}},
	{"inherited member does not shadow lexical global", `let n=5;class Base{let n:integer=2;}class Child:Base{function read():integer{return n;}}let child=new Child();let x=child.read();`, map[string]any{"x": float64(5)}},
	{"immutable initializer and float widening", `class Box{const n:integer=2;let f:float=1;function read():float{return this.n+this.f;}}let box=new Box();let x=box.read();`, map[string]any{"x": float64(3)}},
	{"constructor snapshots arguments across initialization", `let n=2;function change():integer{n=9;return 1;}class Box{let first:integer=change();let n:integer;function constructor(value:integer){this.n=value;}}let box=new Box(n);let x=box.n;let after=n;`, map[string]any{"x": float64(2), "after": float64(9)}},
	{"unqualified method and field alias subtyping", `class Base{let n:integer=1;function read():integer{return this.n;}}class Child:Base{function read():integer{return 8;}function twice():integer{return read()+read();}}class Holder{let value:Base;function constructor(value:Base){this.value=value;}}let child=new Child();let holder=new Holder(child);let x=child.twice();let y=holder.value.read();`, map[string]any{"x": float64(16), "y": float64(8)}},
	{"method recursion and initialization order", `class Node{let a:integer=2;let b:integer=this.a+3;function sum(n:integer):integer{if(n==0)return b;else return a+this.sum(n-1);}}let node=new Node();let x=node.sum(3);`, map[string]any{"x": float64(11)}},
	{"null class and string storage", `class Box{}let value:Box=null;let a=value==null;let text:string=null;let b=text==null;`, map[string]any{"a": true, "b": true}},
	{"typed null results", `class Box{}function empty():Box{return null;}function text():string{return null;}let a=empty();let b=text();let x=a==null;let y=b==null;`, map[string]any{"x": true, "y": true}},
}

func TestObjectInferredNominalMetadata(t *testing.T) {
	for _, source := range []string{
		`class A{}let a=new A();{class A{}let b=a;}`,
		`class A{}let a=new A();function get():A{return a;}{class A{}let b=get();}`,
		`class A{}let a=new A();function check():integer{class A{}let b=a;return 1;}let x=check();`,
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := append([]byte(nil), input...)
			program, parseDiagnostics := frontend.Parse(input)
			scopes, diagnostics := semantic.Analyze(program)
			if len(parseDiagnostics)+len(diagnostics) != 0 {
				t.Fatalf("fixture: %v %v", parseDiagnostics, diagnostics)
			}
			original, _ := json.Marshal([]any{program, scopes})
			p, errors := tac.Generate(program, scopes)
			if p == nil || len(errors) != 0 {
				t.Fatalf("Generate: %+v", errors)
			}
			var outer, alias string
			for _, env := range p.Environments {
				for _, binding := range env.Bindings {
					if binding.Value == "s1.a" {
						outer = binding.Class
					}
					if strings.HasSuffix(binding.Value, ".b") {
						alias = binding.Class
					}
				}
			}
			if outer != "s1.A" || alias != outer {
				t.Fatalf("nominal identity a=%q b=%q", outer, alias)
			}
			for _, procedure := range p.Procedures {
				for _, slot := range procedure.Frame.Slots {
					if slot.Operand != nil && strings.HasSuffix(slot.Operand.Value, ".b") && slot.Operand.Class != outer {
						t.Fatalf("frame alias identity: %+v", slot)
					}
				}
			}
			again, againErrors := tac.Generate(program, scopes)
			after, _ := json.Marshal([]any{program, scopes})
			if !reflect.DeepEqual(p, again) || !reflect.DeepEqual(errors, againErrors) || !bytes.Equal(original, after) || !bytes.Equal(input, before) {
				t.Fatal("mutated inputs or nondeterministic metadata")
			}
		})
	}
	for _, source := range []string{
		`class A{}let a=new A();{class A{}let b:A=a;}`,
		`class A{}let a=new A();{class A{}let b=new A();b=a;}`,
	} {
		r := Compile([]byte(source))
		if r.TAC != nil || len(r.Diagnostics) == 0 {
			t.Fatalf("explicit nominal storage weakened: %+v", r)
		}
	}
}

func TestObjectMetadataAndOracleFaults(t *testing.T) {
	const source = `class Base{let a:integer=1;function read():integer{return this.a;}}class Child:Base{let b:integer=2;function read():integer{return this.b;}}let child=new Child();let x=child.read();let y=child.a;`
	for _, tt := range []struct {
		name    string
		mutate  func(*tac.Program)
		message string
	}{
		{"undefined allocation class", func(p *tac.Program) {
			for j := range p.Instructions {
				if p.Instructions[j].Op == tac.NewObject {
					p.Instructions[j].Class = "missing"
					break
				}
			}
		}, "undefined object class"},
		{"undefined parent class", func(p *tac.Program) { p.Classes[1].Parent = "missing" }, "undefined base class"},
		{"cyclic parent", func(p *tac.Program) { p.Classes[0].Parent = p.Classes[0].ID }, "cyclic class layout"},
		{"invalid layout offset", func(p *tac.Program) { p.Classes[0].Fields[0].Offset = 9 }, "invalid field offset/owner"},
		{"undefined method", func(p *tac.Program) { p.Classes[0].Methods[0].Callee = "missing" }, "invalid method metadata"},
		{"undefined initializer", func(p *tac.Program) { p.Classes[0].Initializer = "missing" }, "undefined class initializer"},
		{"field instruction offset", func(p *tac.Program) {
			for j := range p.Instructions {
				if p.Instructions[j].Op == tac.LoadField {
					offset := 999
					p.Instructions[j].Offset = &offset
					break
				}
			}
		}, "field offset out of bounds"},
		{"field instruction identity", func(p *tac.Program) {
			for j := range p.Instructions {
				if p.Instructions[j].Op == tac.LoadField {
					p.Instructions[j].Member = "missing"
					break
				}
			}
		}, "invalid field identity"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, source)
			tt.mutate(p)
			_, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("err=%v want %s", err, tt.message)
			}
		})
	}
	p := compileConditions(t, source)
	if len(p.Classes) != 2 || len(p.Classes[1].Fields) != 2 || p.Classes[1].Fields[0].ID != p.Classes[0].Fields[0].ID || p.Classes[1].Fields[1].Offset != 1 {
		t.Fatalf("unstable inherited layout: %+v", p.Classes)
	}
	for _, c := range p.Classes {
		if c.Span.Start.Line == 0 || c.Initializer == "" || c.LexicalScope == 0 {
			t.Fatalf("incomplete class metadata: %+v", c)
		}
		for _, field := range c.Fields {
			if field.Span.Start.Line == 0 || field.Owner == "" {
				t.Fatalf("incomplete field metadata: %+v", field)
			}
		}
	}
	legacy := compileConditions(t, `function f(n:integer):integer{return n+1;}let x=f(2);`)
	encoded, _ := json.Marshal(legacy)
	for _, name := range []string{`"classes"`, `"class"`, `"receiver"`, `"resultClass"`, `"typeClasses"`} {
		if bytes.Contains(encoded, []byte(name)) {
			t.Fatalf("legacy optional field present: %s", name)
		}
	}
}

func TestObjectExecution(t *testing.T) {
	for _, tt := range objectCases {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			a := Analyze(input)
			if len(a.Diagnostics) != 0 {
				t.Fatalf("analysis: %+v", a.Diagnostics)
			}
			r := Compile(input)
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", r.Diagnostics)
			}
			got, err := runFunctionFrames(r.TAC)
			if err != nil {
				t.Fatalf("execution: %v\n%s", err, r.TAC)
			}
			for name, want := range tt.want {
				if got["s1."+name] != want {
					t.Errorf("%s=%v want %v", name, got["s1."+name], want)
				}
			}
			first, _ := json.Marshal(r)
			second, _ := json.Marshal(Compile(input))
			if !bytes.Equal(first, second) || !bytes.Equal(input, before) || !reflect.DeepEqual(a.Scopes, r.Scopes) {
				t.Fatal("nondeterministic output or mutated source/scopes")
			}
		})
	}
}

// Arguments precede base-to-derived field initialization; only the selected
// constructor runs, even when ancestors declare their own constructors.
func TestObjectThreeLevelPolicyA(t *testing.T) {
	const source = `let log=0;function mark(n:integer):integer{log=log*10+n;return n;}class A{let a:integer=mark(1);let b:integer=mark(2);function constructor(x:integer){mark(8);}}class B:A{let c:integer=mark(3);function constructor(x:integer){mark(9);}}class C:B{let d:integer=mark(4);function constructor(x:integer){mark(x);}}let c=new C(mark(5));let trace=log;`
	report := Compile([]byte(source))
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		t.Fatalf("compile: %+v", report.Diagnostics)
	}
	values, err := runFunctionFrames(report.TAC)
	if err != nil || values["s1.trace"] != float64(512345) {
		t.Fatalf("trace=%v want=512345 err=%v", values["s1.trace"], err)
	}
	assertClosureCLI(t, buildClosureCLI(t), source, map[string]any{"trace": float64(512345)})
}

func TestObjectAtomicRejections(t *testing.T) {
	for _, source := range []string{
		`class Unused{function bad(){try{missing;}catch(e){}}}`,
		`class Unused{let bad=missing;}`,
		`class Box{const n:integer=1;}let box=new Box();box.n=2;`,
		`class Box{}let box=new Box();let x=box.missing;`,
		`class Box{function constructor(n:integer){}}let box=new Box(true);`,
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := append([]byte(nil), input...)
			a := Analyze(input)
			r := Compile(input)
			if r.TAC != nil || len(r.Diagnostics) == 0 || !bytes.Equal(input, before) || !reflect.DeepEqual(a.Scopes, r.Scopes) {
				t.Fatalf("non-atomic rejection: %+v", r)
			}
		})
	}
}

func TestObjectRuntimeGuards(t *testing.T) {
	for _, tt := range []struct{ name, source, message string }{
		{"undefined field", `class Box{let n:integer;}let box=new Box();let x=box.n;`, "undefined field"},
		{"null receiver", `class Box{let n:integer=1;}let box:Box=null;let x=box.n;`, "null receiver"},
		{"nominal indirect parameter", `class A{}class B{}function a(x:A):integer{return 1;}function b(x:B):integer{return 2;}let alias=a;alias=b;let x=alias(new A());`, "indirect call signature mismatch"},
		{"nominal indirect result", `class A{}class B{}function a():A{return new A();}function b():B{return new B();}let alias=a;alias=b;let x=alias();`, "indirect call signature mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, tt.source)
			_, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("err=%v want %s", err, tt.message)
			}
		})
	}
}

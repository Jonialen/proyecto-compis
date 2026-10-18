package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Expected outcomes were also checked with an independent metadata-indexed cell
// interpreter. These cases retain that evidence through the suite's frame VM.
func TestCompileFunctionInvocationRegressions(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"forward chain", "let x=chain(3); function chain(a:integer):integer { return inc(a)+inc(a+1); } function inc(a:integer):integer { a=a+1; return a; } let y=chain(4);", map[string]any{"s1.x": float64(9), "s1.y": float64(11)}},
		{"live caller temporaries", "function id(a:integer):integer { let x=a*2; return x-1; } let x=id(3)+(id(4)*(id(5)+id(6)));", map[string]any{"s1.x": float64(145)}},
		{"global value snapshots", "let g=1; function bump():integer { g=g+1; return g; } function pair(a:integer,b:integer):integer { return a*10+b; } let x=g+bump(); let y=pair(g,bump()); let z=(g+1)+(bump()*2); let c=g<bump();", map[string]any{"s1.c": true, "s1.g": float64(5), "s1.x": float64(3), "s1.y": float64(23), "s1.z": float64(12)}},
		{"caller locals", "function calc(a:integer):integer { let v=a+1; return (v*2)+(inc(a)*3); } function inc(x:integer):integer { x=x+1; return x; } let x=calc(2); let y=calc(5);", map[string]any{"s1.x": float64(15), "s1.y": float64(30)}},
		{"parameters by value", "let g=8; function f(a:integer):integer { a=a+2; return a; } let x=f(g); let y=f(g);", map[string]any{"s1.g": float64(8), "s1.x": float64(10), "s1.y": float64(10)}},
		{"argument and result widening", "function f(a:float,b:integer,c:boolean):float { if(c) return a+b; else return 0; } let x=f(2,3,true); let y=f(4,5,false);", map[string]any{"s1.x": float64(5), "s1.y": float64(0)}},
		{"short circuit effects", "let g=0; function hit():boolean { g=g+1; return true; } let a=false&&hit(); let b=true||hit(); let c=true&&hit(); let d=false||hit();", map[string]any{"s1.a": false, "s1.b": true, "s1.c": true, "s1.d": true, "s1.g": float64(2)}},
		{"switch selector snapshot", "let b=true; let g=0; function flip():boolean { b=false; g=g+1; return false; } switch(b) { case flip(): g=50; default: g=g+10; }", map[string]any{"s1.b": false, "s1.g": float64(11)}},
		{"null return forms", "let g=0; function a() { g=g+1; return; } function b() { g=g+2; return (null); } function c() { g=g+4; } a(); (b()); c();", map[string]any{"s1.g": float64(7)}},
		{"transitive global initialization", "function a():integer { return b(); } let g=4; function b():integer { return g; } let x=a();", map[string]any{"s1.g": float64(4), "s1.x": float64(4)}},
		{"initializer shadowing", "let x=7; function f(a:integer):integer { let x=x+1; { const k=2; let a=a+k; x=x+a; } return x; } let z=f(3);", map[string]any{"s1.x": float64(7), "s1.z": float64(13)}},
		{"function control flow", "function f(n:integer):integer { let x=0; for(let i=0;i<n;i=i+1) { if(i==1) continue; switch(i==3) { case true: x=x+10; default: x=x+i; } } while(x<15) { x=x+1; } do { x=x+1; } while(false); return x; } let x=f(4); let y=f(0);", map[string]any{"s1.x": float64(16), "s1.y": float64(16)}},
		{"dynamic temporary types", "function f(n:integer):float { let x=n+2; let b=x>0; if(b) { x=x+1; } let y:float=x/2; return y; } let z=f(3);", map[string]any{"s1.z": float64(3)}},
		{"zero argument value", "function f(n:integer):integer { return n; } let z=f(0);", map[string]any{"s1.z": float64(0)}},
		{"empty program", "", map[string]any{}},
		{"unused function", "function f(n:integer):integer { const k=3; if(false) { let hidden=4; } return n+k; }", map[string]any{}},
		{"calls in control tests", "function test(n:integer):boolean { return n<3; } function next(n:integer):integer { return n+1; } let x=0; while(test(x)) { x=next(x); } switch(test(x)) { case test(0): x=9; default: x=next(x); }", map[string]any{"s1.x": float64(4)}},
		{"argument order exactly once", "let g=0; function hit():integer { g=g+1; return g; } function pair(a:float,b:integer):float { return a*10+b; } let x=pair(hit(),hit());", map[string]any{"s1.g": float64(2), "s1.x": float64(12)}},
		{"boolean parameter isolation", "let b=true; function flip(b:boolean):boolean { b=!b; return b; } let x=flip(b); let y=flip(b);", map[string]any{"s1.b": true, "s1.x": false, "s1.y": false}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			report := Compile(input)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report.Diagnostics)
			}
			got, err := runFunctionFrames(report.TAC)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got=%v want=%v err=%v", got, tt.want, err)
			}
			first, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			second, err := json.Marshal(Compile(input))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) || !bytes.Equal(input, before) || !reflect.DeepEqual(report.Scopes, Analyze(input).Scopes) {
				t.Fatal("nondeterminism or input/scope mutation")
			}
		})
	}
}

func TestCompileFunctionAtomicRejectionRegressions(t *testing.T) {
	for _, tt := range []struct {
		name, source, message string
	}{
		{"self recursion", "function f():integer { return f(); }", "recursive call cycles are unsupported"},
		{"mutual recursion", "function f():integer { return g(); } function g():integer { return f(); }", "recursive call cycles are unsupported"},
		{"nested declaration", "function f() { function g() {} }", "nested functions"},
		{"alias", "function f():integer { return 1; } let a=f; let x=a();", ""},
		{"shadow", "function f():integer { return 1; } { let f=2; f(); }", ""},
		{"untyped parameter", "function f(x):integer { return 1; }", "unsupported function signature"},
		{"string parameter", "function f(x:string):integer { return 1; }", "unsupported function signature"},
		{"unused unsupported body", "function f() { print(1); }", "unsupported statement"},
		{"null declaration", "function f() {} let x=f();", "only integer, float and boolean declarations are supported"},
		{"null call return", "function f() {} function g() { return f(); }", "null-result procedures only return null or no value"},
		{"early read", "let x=f(); let g=2; function f():integer { return g; }", "call requires initialized global binding s1.g"},
		{"early transitive read", "function a():integer { return b(); } let x=a(); let g=2; function b():integer { return g; }", "call requires initialized global binding s1.g"},
		{"early write", "f(); let g=2; function f() { g=3; }", "call requires initialized global binding s1.g"},
		{"declaration visibility", "function f():integer { return g; } let g=2; let x=f();", ""},
		{"argument type", "function f(a:integer):integer { return a; } let x=f(true);", ""},
		{"missing return", "function f():integer { let x=2; }", ""},
		{"duplicate parameter", "function f(a:integer,a:integer):integer { return a; }", ""},
		{"unused null declaration", "function f() {} function g() { let x=f(); }", "only integer, float and boolean declarations are supported"},
		{"null operand", "function f() {} let x=f()==null;", ""},
		{"same name alias", "function f():integer { return 1; } { let f=f; let x=f(); }", ""},
		{"parameter shadows callee", "function f():integer { return 1; } function g(f:integer):integer { return f(); }", ""},
		{"conservative dependency", "let x=f(); let g=2; function f():integer { if(false) { return g; } return 1; }", "call requires initialized global binding s1.g"},
		{"syntax error", "function f( {", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) == 0 || !bytes.Equal(input, before) {
				t.Fatalf("non-atomic rejection: %+v", report)
			}
			if tt.message != "" {
				for _, diagnostic := range report.Diagnostics {
					if strings.Contains(diagnostic.Message, tt.message) {
						return
					}
				}
				t.Fatalf("missing %q: %+v", tt.message, report.Diagnostics)
			}
		})
	}
}

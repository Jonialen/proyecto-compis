package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

const escapingCounter = `function zero():integer{return 0;} let saved=zero; function install(seed:integer){let count=seed; function next():integer{count=count+1;return count;} saved=next;} install(10); let first=saved(); let second=saved();`

func TestClosureExecution(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"escaping mutable counter", escapingCounter, map[string]any{"first": float64(11), "second": float64(12)}},
		{"distinct invocations", `function zero():integer{return 0;} let saved=zero; function install(seed:integer){let count=seed; function next():integer{count=count+1;return count;} saved=next;} install(10); let a=saved; install(20); let b=saved; let x=a(); let y=b(); let z=a();`, map[string]any{"x": float64(11), "y": float64(21), "z": float64(12)}},
		{"sibling shared cell", `function zero():integer{return 0;} let inc=zero; let read=zero; function install(){let count=3; function next():integer{count=count+1;return count;} function get():integer{return count;} inc=next; read=get;} install(); let x=inc(); let y=read();`, map[string]any{"x": float64(4), "y": float64(4)}},
		{"grandparent", `function zero():integer{return 0;} let saved=zero; function install(seed:integer){let n=seed; function middle(){function inner():integer{n=n+1;return n;} saved=inner;} middle();} install(8); let x=saved(); let y=saved();`, map[string]any{"x": float64(9), "y": float64(10)}},
		{"hoisted nested mutual recursion", `function outer(n:integer):integer{let seed=7; let x=even(n); function even(n:integer):integer{if(n==0)return seed;else return odd(n-1);} function odd(n:integer):integer{if(n==0)return 0;else return even(n-1);} return x;} let x=outer(4); let y=outer(3);`, map[string]any{"x": float64(7), "y": float64(0)}},
		{"recursive owner frames", `function outer(n:integer):integer{let seed=n; function read():integer{return seed;} if(n==0)return read();else return read()+outer(n-1)+read();} let x=outer(4);`, map[string]any{"x": float64(20)}},
		{"shadow initializer", `function outer():integer{let x=3; {let x=x+1; function read():integer{return x;} return read();}} let x=outer();`, map[string]any{"x": float64(4)}},
		{"block reentry", `function zero():integer{return 0;} let a=zero; let b=zero; let i=0; while(i<2){let n=i; function read():integer{return n;} if(i==0)a=read;else b=read; i=i+1;} let x=a(); let y=b();`, map[string]any{"x": float64(0), "y": float64(1)}},
		{"callee selected before arguments", `function first(n:integer):integer{return n+10;} function second(n:integer):integer{return n+20;} let alias=first; function change():integer{alias=second;return 1;} let x=alias(change()); let y=alias(2);`, map[string]any{"x": float64(11), "y": float64(22)}},
		{"alias arguments frozen", `let n=1; function bump():integer{n=n+1;return n;} function pair(a:integer,b:integer):integer{return a*10+b;} let alias=pair; let x=alias(n,bump());`, map[string]any{"x": float64(12)}},
		{"grouped and shadowed alias", `function f():integer{return 4;} {let f=f; let alias=(f); let x=alias();} let x=f();`, map[string]any{"x": float64(4)}},
		{"null alias", `let n=0; function f(){n=n+1;} let alias=f; alias();`, map[string]any{"n": float64(1)}},
		{"hoisted alias initialized before call", `function outer():integer{let alias=read; let n=8; function read():integer{return n;} return alias();} let x=outer();`, map[string]any{"x": float64(8)}},
		{"captured parameter", `function zero():integer{return 0;} let saved=zero; function install(n:integer){function next():integer{n=n+1;return n;} saved=next;} install(4); let x=saved(); let y=saved();`, map[string]any{"x": float64(5), "y": float64(6)}},
		{"for block reentry", `function zero():integer{return 0;} let a=zero; let b=zero; for(let i=0;i<2;i=i+1){let n=i; function read():integer{return n;} if(i==0)a=read;else b=read;} let x=a(); let y=b();`, map[string]any{"x": float64(0), "y": float64(1)}},
		{"discarded function value", `function f():integer{return 1;} f; let x=f();`, map[string]any{"x": float64(1)}},
		{"unused nested declaration", `function f(){function g(){return;} return;}`, map[string]any{}},
		{"switch sequence hoisting", `let x=0; switch(true){case true: x=f(); function f():integer{return 4;} default:x=9;}`, map[string]any{"x": float64(4)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			analysis := Analyze(input)
			if len(analysis.Diagnostics) != 0 {
				t.Fatalf("analysis: %+v", analysis.Diagnostics)
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
			a, _ := json.Marshal(r)
			b, _ := json.Marshal(Compile(input))
			if !bytes.Equal(a, b) || !bytes.Equal(input, before) || !reflect.DeepEqual(r.Scopes, analysis.Scopes) {
				t.Fatal("source/scopes changed or nondeterministic output")
			}
		})
	}
}

func TestClosureAtomicRejections(t *testing.T) {
	for _, tt := range []struct{ name, source, message string }{
		{"early captured read", `function outer():integer{let x=read(); let n=8; function read():integer{return n;} return x;} let x=outer();`, "call requires initialized captured binding"},
		{"early captured write", `function outer(){set(); let n=8; function set(){n=9;}} outer();`, "call requires initialized captured binding"},
		{"uncalled nested unsupported body", `function outer(){function bad(){try{}catch(e){}}}`, "unsupported statement"},
		{"unreachable nested unsupported body", `function outer(){if(false){function bad(){try{}catch(e){}}}}`, "unsupported statement"},
		{"nested signature", `function outer(){function bad(n){return;}}`, "unsupported function signature"},
		{"alias initializer unsupported graph", `function f():integer{return 1;} let alias=f; function unused(){try{}catch(e){}} let x=alias();`, "unsupported statement"},
		{"alias unknown result expression", `function f(){} let alias=f; function g(x){}`, "unsupported function signature"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source)
			before := append([]byte(nil), input...)
			a := Analyze(input)
			if len(a.Diagnostics) != 0 {
				t.Fatalf("fixture analysis: %+v", a.Diagnostics)
			}
			r := Compile(input)
			if r.TAC != nil || len(r.Diagnostics) != 1 || !strings.Contains(r.Diagnostics[0].Message, tt.message) || !bytes.Equal(input, before) || !reflect.DeepEqual(a.Scopes, r.Scopes) {
				t.Fatalf("non-atomic rejection: %+v", r)
			}
		})
	}
}

func TestClosureDynamicCallGuards(t *testing.T) {
	for _, tt := range []struct{ name, source, message string }{
		{"early global read", `let alias=read; let x=alias(); let n=8; function read():integer{return n;}`, "call requires initialized global binding s1.n"},
		{"early global write", `let alias=set; alias(); let n=8; function set(){n=9;}`, "call requires initialized global binding s1.n"},
		{"early transitive indirect global", `let alias=read; function invoke():integer{return alias();} let x=invoke(); let n=8; function read():integer{return n;}`, "call requires initialized global binding s1.n"},
		{"early captured read", `function outer():integer{let alias=read; let x=alias(); let n=8; function read():integer{return n;} return x;} let x=outer();`, "call requires initialized captured binding"},
		{"early captured write", `function outer(){let alias=set; alias(); let n=8; function set(){n=9;}} outer();`, "call requires initialized captured binding"},
		{"assigned incompatible arguments", `function a(n:integer):integer{return n;} function b(n:boolean):integer{if(n)return 1;else return 0;} let alias=a; alias=b; let x=alias(4);`, "indirect call signature mismatch"},
		{"assigned incompatible result", `function a():integer{return 1;} function b():boolean{return true;} let alias=a; alias=b; let x=alias();`, "indirect call signature mismatch"},
		{"assigned incompatible arity", `function a():integer{return 1;} function b(n:integer):integer{return n;} let alias=a; alias=b; let x=alias();`, "indirect call signature mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, tt.source)
			got, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("values=%v error=%v want %s", got, err, tt.message)
			}
		})
	}
}

func TestClosureOracleRejectsMalformedIR(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mutate  func(*tac.Program)
		message string
	}{
		{"missing environment layout", func(p *tac.Program) { p.Instructions[0].Environment = 999 }, "undefined environment layout"},
		{"cyclic parent", func(p *tac.Program) { p.Environments[0].ParentID = p.Environments[0].ScopeID }, "cyclic environment layout"},
		{"foreign cell", func(p *tac.Program) {
			p.Instructions[1].Target = tac.Operand{Kind: tac.Symbol, Value: "s1.missing", Type: model.TypeFunction}
		}, "environment binding out of bounds"},
		{"undefined closure procedure", func(p *tac.Program) { p.Instructions[1].Callee = "missing" }, "undefined closure procedure"},
		{"undefined closure environment", func(p *tac.Program) { p.Instructions[1].Environment = 999 }, "undefined closure environment"},
		{"missing signature", func(p *tac.Program) {
			for i := range p.Instructions {
				if p.Instructions[i].Op == tac.CallIndirect {
					p.Instructions[i].Signature = nil
				}
			}
		}, "indirect call signature mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, `function f():integer{return 1;} let alias=f; let x=alias();`)
			tt.mutate(p)
			_, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("err=%v want %s", err, tt.message)
			}
		})
	}
}

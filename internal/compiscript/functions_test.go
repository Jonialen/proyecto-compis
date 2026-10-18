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

func TestCompileDirectFunctions(t *testing.T) {
	for _, source := range []string{
		`let x=f(2); function f(a:integer):integer{return a+1;}`,
		`function f(a:integer):float{a=a+1; return a;} let x=f(2)+f(3);`,
		`function f(){return;} f(); function g(){return null;} g(); function h(){} h();`,
		`function f(a:boolean):integer{if(a){return 1;}else{return 2;}} let x=f(true);`,
		`function f():integer{let x=0; while(x<2){x=x+1;} switch(true){case true:return x; default:return 0;}} let x=f();`,
		`function first():integer{return later();} let g=7; function later():integer{return g;} let x=first();`,
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := append([]byte(nil), input...)
			r := Compile(input)
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", r.Diagnostics)
			}
			if !bytes.Equal(input, before) || !reflect.DeepEqual(r.Scopes, Analyze(input).Scopes) {
				t.Fatal("source or scopes changed")
			}
			a, _ := json.Marshal(r)
			b, _ := json.Marshal(Compile(input))
			if !bytes.Equal(a, b) {
				t.Fatal("nondeterministic report")
			}
		})
	}
}

func TestFunctionFrameExecution(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"forward repeated chained", `let x=outer(2)+outer(3); function outer(n:integer):integer{return twice(n)+twice(n+1);} function twice(n:integer):integer{let a=n*2; return a;}`, map[string]any{"x": float64(24)}},
		{"mutable by value", `let a=2; function f(a:integer):integer{a=a+1; return a;} let x=f(a)+f(a);`, map[string]any{"a": float64(2), "x": float64(6)}},
		{"mutable boolean by value", `let a=true; function f(a:boolean):boolean{a=!a;return a;} let x=f(a);`, map[string]any{"a": true, "x": false}},
		{"division and unary comparisons", `function half(a:integer):float{return -a/2;} function test(a:integer):boolean{return (-a<=0)&&(a>=1)&&(a!=0);} let x=half(5); let y=test(2);`, map[string]any{"x": float64(-2.5), "y": true}},
		{"caller live values", `function f(n:integer):integer{return (n+1)*(n+2);} let x=(1+2)*(f(2)+f(f(1)));`, map[string]any{"x": float64(204)}},
		{"formal and result widening", `function f(a:float,b:integer):float{b=b+1;return a+b;} function g():float{return 2;} let x=f(1,2)+g();`, map[string]any{"x": float64(6)}},
		{"binary freeze", `let g=1; function bump():integer{g=g+1;return g;} let x=g+bump();`, map[string]any{"g": float64(2), "x": float64(3)}},
		{"nested binary freeze", `let g=1; function bump():integer{g=g+1;return g;} let x=g+(g+bump());`, map[string]any{"g": float64(2), "x": float64(4)}},
		{"comparison freeze", `let g=1; function bump():integer{g=g+1;return g;} let x=g<bump(); let y=g==bump();`, map[string]any{"g": float64(3), "x": true, "y": false}},
		{"argument freeze", `let g=1; function bump():integer{g=g+1;return g;} function pair(a:integer,b:integer):integer{return a*10+b;} let x=pair(g,bump());`, map[string]any{"g": float64(2), "x": float64(12)}},
		{"ordered arguments once", `let g=0; function bump():integer{g=g+1;return g;} function pair(a:integer,b:integer):integer{return a*10+b;} let x=pair(bump(),bump());`, map[string]any{"g": float64(2), "x": float64(12)}},
		{"promoted argument freeze", `let g=1; function bump():float{g=g+1;return g;} function pair(a:float,b:float):float{return a*10+b;} let x=pair(g,bump());`, map[string]any{"g": float64(2), "x": float64(12)}},
		{"switch selector freeze", `let flag=true; let x=0; function flip():boolean{flag=false;return true;} switch(flag){case flip():x=1; default:x=2;}`, map[string]any{"flag": false, "x": float64(1)}},
		{"short circuit calls", `let g=0; function hit():boolean{g=g+1;return true;} let a=false&&hit(); let b=true||hit(); let c=true&&hit(); let x=false||hit();`, map[string]any{"g": float64(2), "a": false, "b": true, "c": true, "x": true}},
		{"null returns", `let g=0; function a(){g=g+1;return;} function b(){g=g+1;return null;} function c(){g=g+1;} a(); (b()); c();`, map[string]any{"g": float64(3)}},
		{"control flow", `function f(n:integer):integer{let sum=0; for(let i=0;i<n;i=i+1){const k=2; if(i==1)continue; {let v=i*k; sum=sum+v;} if(i==3)break;} do{sum=sum+1;}while(false); while(sum<10){sum=sum+1;} switch(n>1){case true:return sum; default:return 0;}} let x=f(5); let y=f(1);`, map[string]any{"x": float64(11), "y": float64(0)}},
		{"if returns", `function f(n:boolean):integer{if(n){return 3;}else{return 4;}} let x=f(true)+f(false);`, map[string]any{"x": float64(7)}},
		{"transitive late global", `function a():integer{return b();} let g=7; function b():integer{return g;} let x=a();`, map[string]any{"g": float64(7), "x": float64(7)}},
		{"global write", `let g=1; function put(v:integer){g=v;} put(9); let x=g;`, map[string]any{"g": float64(9), "x": float64(9)}},
		{"global constant", `const g=3; function f():integer{return g;} let x=f();`, map[string]any{"g": float64(3), "x": float64(3)}},
		{"statement scalar result", `let g=0; function f():integer{g=g+1;return g;} f(); let x=f();`, map[string]any{"g": float64(2), "x": float64(2)}},
		{"case scope ownership", `function f():integer{let x=0; switch(true){case true:const a=3; x=a; default:x=9;} return x;} let x=f();`, map[string]any{"x": float64(3)}},
		{"shadow initializer", `let g=2; function f(g:integer):integer{{let g=g+1; g=g+1;} return g;} let x=f(g);`, map[string]any{"g": float64(2), "x": float64(2)}},
		{"loop and switch calls", `function test(n:integer):boolean{return n<3;} function next(n:integer):integer{return n+1;} let x=0; while(test(x)){x=next(x);} switch(test(x)){case test(0):x=9; default:x=next(x);}`, map[string]any{"x": float64(4)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, tt.source)
			values, err := runFunctionFrames(p)
			if err != nil {
				t.Fatalf("execution: %v\n%s", err, p.String())
			}
			for name, want := range tt.want {
				if got := values["s1."+name]; got != want {
					t.Errorf("%s=%v want %v; globals=%v", name, got, want, values)
				}
			}
			for name := range values {
				if !strings.HasPrefix(name, "s1.") {
					t.Errorf("callee-owned value escaped: %s", name)
				}
			}
		})
	}
}

func TestFunctionFrameMetadata(t *testing.T) {
	p := compileConditions(t, `function f(z:integer,a:boolean):float{const c=2; {let b=z+c;} for(let i=0;i<1;i=i+1){let q=i;} let t=z+1; let flag=!a; return t;} let x=f(2,true);`)
	if len(p.Procedures) != 1 {
		t.Fatal("missing procedure")
	}
	f := p.Procedures[0]
	if f.Label != "s1.f" || f.ScopeID != 2 || f.ResultKind != model.TypeFloat || len(f.Parameters) != 2 || f.Parameters[0].Value != "s2.z" || f.Parameters[1].Value != "s2.a" {
		t.Fatalf("identity/formals: %+v", f)
	}
	wantRoles := []string{"continuation", "controlLink", "globalAccessLink", "result", "parameter", "parameter", "local", "local", "local", "local", "local", "local"}
	wantNames := []string{"s2.z", "s2.a", "s2.c", "s3.b", "s4.i", "s5.q", "s2.t", "s2.flag"}
	if f.Frame.Cells != len(wantRoles)+f.PeakTemporaries || len(f.Frame.Slots) != f.Frame.Cells {
		t.Fatalf("frame cells: %+v", f.Frame)
	}
	for i, slot := range f.Frame.Slots {
		if slot.Offset != i || slot.Size != 1 {
			t.Fatalf("cell index/size: %+v", slot)
		}
		if i < len(wantRoles) && slot.Role != wantRoles[i] {
			t.Fatalf("role[%d]=%s", i, slot.Role)
		}
		if i >= 4 && i < len(wantRoles) && (slot.Operand == nil || slot.Operand.Value != wantNames[i-4] || slot.Dynamic) {
			t.Fatalf("owned binding[%d]: %+v", i, slot)
		}
		if i >= len(wantRoles) && (slot.Role != "temporary" || !slot.Dynamic || slot.Operand == nil || slot.Operand.Kind != tac.Temporary || slot.Operand.Type != "") {
			t.Fatalf("dynamic temporary: %+v", slot)
		}
	}
	seen := map[string]map[model.TypeKind]bool{}
	for _, i := range f.Instructions {
		if i.Target.Kind == tac.Temporary {
			if seen[i.Target.Value] == nil {
				seen[i.Target.Value] = map[model.TypeKind]bool{}
			}
			seen[i.Target.Value][i.Target.Type] = true
		}
	}
	changed := false
	for _, kinds := range seen {
		if len(kinds) > 1 {
			changed = true
		}
	}
	if !changed {
		t.Fatal("fixture must exercise type-changing temporary reuse")
	}
	if _, err := runFunctionFrames(p); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(p)
	if err != nil || !bytes.Contains(data, []byte(`"procedures":[{"label":"s1.f","scopeId":2,"span":`)) || !bytes.Contains(data, []byte(`"offset":0,"size":1,"role":"continuation"`)) || !bytes.Contains(data, []byte(`"dynamic":true`)) {
		t.Fatalf("layout JSON: %s %v", data, err)
	}
	for _, instruction := range f.Instructions {
		if instruction.Span.Start.Line < 1 || instruction.Span.End.Offset <= instruction.Span.Start.Offset {
			t.Fatalf("missing source span: %+v", instruction)
		}
	}
}

func TestFunctionTextAndOptionalIRFields(t *testing.T) {
	p := compileConditions(t, `let x=f(2); function f(a:integer):integer{return a+1;}`)
	want := "main:\n" +
		"t0:integer = call s1.f(2:integer)\ns1.x:integer = t0:integer\n" +
		"procedure s1.f(s2.a:integer) -> integer [scope=2 frame=6 cells]:\n" +
		"  cell 0 continuation\n  cell 1 controlLink\n  cell 2 globalAccessLink\n  cell 3 result\n" +
		"  cell 4 parameter s2.a:integer\n  cell 5 temporary t0 dynamic\n" +
		"t0:integer = s2.a:integer + 1:integer\nreturn t0:integer\nend s1.f\n"
	if got := p.String(); got != want {
		t.Fatalf("text:\n%s\nwant:\n%s", got, want)
	}
	data, err := json.Marshal(p.Instructions[0])
	wantCall := `{"op":"call","target":{"kind":"temporary","value":"t0","type":"integer"},"left":{"kind":"","value":"","type":""},"span":{"start":{"offset":6,"line":1,"column":7},"end":{"offset":10,"line":1,"column":11}},"callee":"s1.f","arguments":[{"kind":"literal","value":"2","type":"integer"}]}`
	if err != nil || string(data) != wantCall {
		t.Fatalf("call JSON=%s err=%v", data, err)
	}
	data, err = json.Marshal(p.Procedures[0].Frame)
	wantFrame := `{"cells":6,"slots":[{"offset":0,"size":1,"role":"continuation"},{"offset":1,"size":1,"role":"controlLink"},{"offset":2,"size":1,"role":"globalAccessLink"},{"offset":3,"size":1,"role":"result"},{"offset":4,"size":1,"role":"parameter","operand":{"kind":"symbol","value":"s2.a","type":"integer"}},{"offset":5,"size":1,"role":"temporary","operand":{"kind":"temporary","value":"t0","type":""},"dynamic":true}]}`
	if err != nil || string(data) != wantFrame {
		t.Fatalf("frame JSON=%s err=%v", data, err)
	}
	legacy := compileConditions(t, `let x=1;`)
	data, err = json.Marshal(legacy)
	if err != nil || bytes.Contains(data, []byte(`"procedures"`)) || bytes.Contains(data, []byte(`"callee"`)) || bytes.Contains(data, []byte(`"returnValue"`)) || bytes.Contains(data, []byte(`"arguments"`)) {
		t.Fatalf("new metadata leaked to old IR: %s %v", data, err)
	}
	// Null-result procedures have no scalar result cell or invented default.
	null := compileConditions(t, `function f(){} f();`)
	if null.Procedures[0].Frame.Cells != 3 || null.Procedures[0].ResultKind != model.TypeNull || len(null.Procedures[0].Instructions) != 1 || null.Procedures[0].Instructions[0].ReturnValue != nil || null.Instructions[0].Target.Kind != "" {
		t.Fatalf("null frame: %+v", null)
	}
}

func TestFunctionTransitiveGlobalMetadata(t *testing.T) {
	p := compileConditions(t, `let z=1; let a=2; function first():integer{return second();} function second():integer{z=z+1;return z+a;} let x=first();`)
	want := []tac.Operand{{Kind: tac.Symbol, Value: "s1.a", Type: model.TypeInteger}, {Kind: tac.Symbol, Value: "s1.z", Type: model.TypeInteger}}
	for _, procedure := range p.Procedures {
		if !reflect.DeepEqual(procedure.Globals, want) {
			t.Fatalf("%s globals=%+v want %+v", procedure.Label, procedure.Globals, want)
		}
	}
	values, err := runFunctionFrames(p)
	if err != nil || values["s1.x"] != float64(4) {
		t.Fatalf("values=%v err=%v", values, err)
	}
}

func TestFunctionGlobalInitializationAndVisibility(t *testing.T) {
	for _, source := range []string{
		`let x=f(); let g=1; function f():integer{return g;}`,
		`f(); let g=1; function f(){g=2;}`,
		`function a():integer{return b();} let x=a(); let g=1; function b():integer{return g;}`,
		`function a():integer{return g;} let g=1; let x=a();`,
		`let g=0; switch(true){case true:let later=1;} function f():integer{return later;} let x=f();`,
		`function f():integer{return 1;} {const f=2; f=3;}`,
		`function f(a:integer):integer{switch(true){case true:let a=1; default:return a;} return a;}`,
		`function f():boolean{return true;} let x=true||((true?[]:[])==null);`,
		`function f():integer{return g();} function g():integer{return 1;} function unused(){try{missing;}catch(e){}}`,
	} {
		t.Run(source, func(t *testing.T) {
			analysis := Analyze([]byte(source))
			r := Compile([]byte(source))
			if r.TAC != nil || len(r.Diagnostics) == 0 {
				t.Fatalf("expected atomic rejection: %+v", r)
			}
			if len(analysis.Diagnostics) > 0 && !reflect.DeepEqual(r.Diagnostics, analysis.Diagnostics) {
				t.Fatalf("semantic errors changed: %+v vs %+v", r.Diagnostics, analysis.Diagnostics)
			}
		})
	}
}

func TestFunctionCapabilityDiagnostics(t *testing.T) {
	for _, tt := range []struct{ source, rejected, message string }{
		{`let x=f(1);let g=1;function f(a):integer{return g;}`, `f(1);`, "call requires initialized global binding s1.g"},
		{`function f(){}g(1);let x=1;function g(a){x=2;}`, `g(1);`, "call requires initialized global binding s1.x"},
		{`let x=f(); let g=1; function f():integer{return g;}`, `f();`, "call requires initialized global binding s1.g"},
		{`function a():integer{return b();} let x=a(); let g=1; function b():integer{return g;}`, `a();`, "call requires initialized global binding s1.g"},
		{`f(); let g=1; function f(){g=2;}`, `f();`, "call requires initialized global binding s1.g"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			analysis := Analyze([]byte(tt.source))
			if len(analysis.Diagnostics) != 0 {
				t.Fatalf("fixture: %+v", analysis.Diagnostics)
			}
			r := Compile([]byte(tt.source))
			start := strings.Index(tt.source, tt.rejected)
			length := len(tt.rejected)
			// Call-expression spans exclude the statement semicolon or outer equality.
			if strings.HasSuffix(tt.rejected, ";") {
				length--
			}
			want := model.Diagnostics{{Code: "TAC_UNSUPPORTED", Phase: model.Phase("codegen"), Message: tt.message, Span: compileSpan(start, start+length)}}
			if r.TAC != nil || !reflect.DeepEqual(r.Diagnostics, want) || !reflect.DeepEqual(r.Scopes, analysis.Scopes) {
				t.Fatalf("report=%+v want diagnostics=%+v", r, want)
			}
		})
	}
}

func TestFunctionOracleRejectsInvalidFrames(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*tac.Program)
		want   string
	}{
		{"foreign local", func(p *tac.Program) {
			p.Procedures[0].Instructions[0].ReturnValue = &tac.Operand{Kind: tac.Symbol, Value: "s99.x", Type: model.TypeInteger}
		}, "incorrect frame ownership"},
		{"undefined local", func(p *tac.Program) {
			p.Procedures[0].Instructions[0].ReturnValue = &tac.Operand{Kind: tac.Temporary, Value: "t0", Type: model.TypeInteger}
			p.Procedures[0].Frame.Slots = append(p.Procedures[0].Frame.Slots, tac.FrameSlot{Offset: len(p.Procedures[0].Frame.Slots), Size: 1, Role: "temporary", Operand: &tac.Operand{Kind: tac.Temporary, Value: "t0"}, Dynamic: true})
			p.Procedures[0].Frame.Cells++
		}, "undefined slot"},
		{"undefined label", func(p *tac.Program) {
			p.Procedures[0].Instructions = append([]tac.Instruction{{Op: tac.Jump, Label: "missing"}}, p.Procedures[0].Instructions...)
		}, "undefined label"},
		{"bad offset", func(p *tac.Program) { p.Procedures[0].Frame.Slots[0].Offset = 9 }, "invalid cell offset"},
		{"undefined callee", func(p *tac.Program) { p.Instructions[0].Callee = "missing" }, "undefined procedure"},
		{"bounded execution", func(p *tac.Program) {
			p.Procedures[0].Instructions = []tac.Instruction{{Op: tac.Label, Label: "loop"}, {Op: tac.Jump, Label: "loop"}}
		}, "instruction budget exceeded"},
		{"bounded depth", func(p *tac.Program) { p.Procedures[0].Instructions = []tac.Instruction{{Op: tac.Call, Callee: "s1.f"}} }, "call depth budget exceeded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := compileConditions(t, `function f():integer{return 1;} let x=f();`)
			tt.mutate(p)
			_, err := runFunctionFrames(p)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err=%v want %s", err, tt.want)
			}
		})
	}
}

func TestCompileFunctionRejectionsAreAtomic(t *testing.T) {
	for _, source := range []string{
		`f(1);let g=1;function f(a){g=2;return;}`,
		`f(["x"]);let g=1;function f(a:string[]){try{print(a[0]);g=2;}catch(e){}}`,
		`f();let g=1;function f(){try{g=2;}catch(e){}}`,
		`f();let x;function f(){x=2;return;}`,
		`g();let x;function f(a){x=2;return;} function g(){return f(null);}`,
		`let x=f(); let g=1; function f():integer{return g;}`,
		`function f():integer{return g();} let x=f(); let a=1; function g():integer{return a;}`,
	} {
		t.Run(source, func(t *testing.T) {
			if diagnostics := Analyze([]byte(source)).Diagnostics; len(diagnostics) != 0 {
				t.Fatalf("capability fixture must pass analysis: %+v", diagnostics)
			}
			r := Compile([]byte(source))
			if r.TAC != nil || len(r.Diagnostics) == 0 {
				t.Fatalf("expected atomic rejection: %+v", r)
			}
			if !reflect.DeepEqual(r.Scopes, Analyze([]byte(source)).Scopes) {
				t.Fatal("scopes changed")
			}
		})
	}
}

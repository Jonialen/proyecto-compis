package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
	"genanalex/internal/compiscript/tac"
)

// These compositions were also executed by an independent environment-cell
// interpreter; keep their observable behavior at the public compiler boundary.
func TestClosureCompositions(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"shared for header", "function zero():integer{return 0;} let saved=zero; let other=zero;for(let i=0;i<3;i=i+1){function get():integer{return i;}if(i==0)saved=get;else other=get;}let a=saved();let b=other();", map[string]any{"s1.a": float64(3), "s1.b": float64(3)}},
		{"callee selected before arguments", "function outer():integer{let n=1;function a(x:integer):integer{return n+x;}function b(x:integer):integer{return n+x+100;}let chosen=a;function change():integer{chosen=b;n=9;return 2;}return chosen(change());}let x=outer();", map[string]any{"s1.x": float64(11)}},
		{"transitive grandparent capture", "function zero():integer{return 0;} let saved=zero; function install(n:integer){function mid(){function leaf():integer{return n;}function bridge():integer{return leaf();}saved=bridge;}mid();}install(17);let x=saved();", map[string]any{"s1.x": float64(17)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.source)
			before := append([]byte(nil), source...)
			tree, diagnostics := frontend.Parse(source)
			if len(diagnostics) != 0 {
				t.Fatal(diagnostics)
			}
			scopes, diagnostics := semantic.Analyze(tree)
			if len(diagnostics) != 0 {
				t.Fatal(diagnostics)
			}
			encode := func(value any) []byte {
				t.Helper()
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			astBefore, scopesBefore := encode(tree), encode(scopes)
			generated, diagnostics := tac.Generate(tree, scopes)
			if generated == nil || len(diagnostics) != 0 {
				t.Fatalf("Generate: %+v", diagnostics)
			}
			report := Compile(source)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("Compile: %+v", report.Diagnostics)
			}
			values, err := runFunctionFrames(report.TAC)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range tt.want {
				if values[name] != want {
					t.Errorf("%s=%v want=%v", name, values[name], want)
				}
			}
			if !bytes.Equal(before, source) || !bytes.Equal(astBefore, encode(tree)) || !bytes.Equal(scopesBefore, encode(scopes)) ||
				!bytes.Equal(encode(generated), encode(report.TAC)) || !bytes.Equal(encode(report.TAC), encode(Compile(source).TAC)) ||
				!reflect.DeepEqual(Analyze(source).Scopes, report.Scopes) {
				t.Fatal("mutation, nondeterminism, or public API mismatch")
			}
		})
	}
}

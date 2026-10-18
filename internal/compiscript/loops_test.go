package compiscript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestLoopExecution(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         map[string]any
	}{
		{"while zero", "let x=0; while(false){x=1;}", map[string]any{"s1.x": float64(0)}},
		{"while one", "let x=0; while(x<1){x=x+1;}", map[string]any{"s1.x": float64(1)}},
		{"while continue", "let x=0; let y=0; while(x<5){x=x+1; if(x<3)continue; y=y+1;}", map[string]any{"s1.x": float64(5), "s1.y": float64(3)}},
		{"do once", "let x=0; do{x=x+1;}while(false);", map[string]any{"s1.x": float64(1)}},
		{"do continue condition", "let x=0; do{x=x+1; continue;}while(x<3);", map[string]any{"s1.x": float64(3)}},
		{"do outer condition", "let x=0; do{let x=x+1;}while(x!=0);", map[string]any{"s1.x": float64(0), "s2.x": float64(1)}},
		{"for continue post", "let x=0; let y=0; for(;x<4;x=x+1){if(x<2)continue; y=y+1;}", map[string]any{"s1.x": float64(4), "s1.y": float64(2)}},
		{"for break skips post", "let x=0; for(;;x=x+1){if(x==2)break;}", map[string]any{"s1.x": float64(2)}},
		{"for initializer once", "let x=9; let y=0; for(x=0;x<3;x=x+1){y=y+x;}", map[string]any{"s1.x": float64(3), "s1.y": float64(3)}},
		{"for shadow initializer", "let x=2; let y:float=0; for(let x=x;x<5;(x=x+1)){let x=x+1; y=y+x;} x=x+1;", map[string]any{"s1.x": float64(3), "s1.y": float64(12), "s2.x": float64(5), "s3.x": float64(5)}},
		{"mixed nearest transfers", "let x=0; let y=0; while(x<3){x=x+1; for(let i=0;i<4;i=i+1){if(i==1)continue; if(i==3)break; y=y+1;} if(x==2)continue; y=y+10;}", map[string]any{"s1.x": float64(3), "s1.y": float64(26)}},
		{"nested do break", "let x=0; let y=0; for(;x<3;x=x+1){do{y=y+1; break;}while(true);}", map[string]any{"s1.x": float64(3), "s1.y": float64(3)}},
		{"consecutive loop targets", "let x=0; while(true){x=x+1; break;} do{x=x+1; continue;}while(false); for(;;){x=x+1; break;} x=x+1;", map[string]any{"s1.x": float64(4)}},
		{"short circuit backedge", "let x=0; while(x<3 && (true || 1/0>0)){x=x+1;}", map[string]any{"s1.x": float64(3)}},
		{"short circuit do", "let x=0; do{x=x+1;}while(false && 1/0>0);", map[string]any{"s1.x": float64(1)}},
		{"short circuit for", "let x=0; for(;false && 1/0>0;x=x+1){}", map[string]any{"s1.x": float64(0)}},
		{"discard numeric post", "let x=0; for(;x<3;(1+2)*(3+4)){x=x+1;}", map[string]any{"s1.x": float64(3)}},
		{"discard boolean post", "let x=0; for(;x<3;false||true){x=x+1;}", map[string]any{"s1.x": float64(3)}},
		{"omitted condition boolean post", "let x=0; for(;;false||true){x=x+1; if(x==3)break;}", map[string]any{"s1.x": float64(3)}},
		{"float post widening", "let x:float=0; for(;x<3;x=x+1){}", map[string]any{"s1.x": float64(3)}},
		{"nested while targets", "let x=0; let y=0; while(x<3){x=x+1; while(true){y=y+1; break;} if(x<3)continue; y=y+10;}", map[string]any{"s1.x": float64(3), "s1.y": float64(13)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			program := compileConditions(t, tt.source)
			values, divisions, err := runConditions(program)
			if err != nil || divisions != 0 {
				t.Fatalf("values=%v divisions=%d err=%v", values, divisions, err)
			}
			for name, want := range tt.want {
				if values[name] != want {
					t.Errorf("%s=%v want %v (all=%v)", name, values[name], want, values)
				}
			}
		})
	}
}

func TestLoopOptionalForClauses(t *testing.T) {
	for _, init := range []string{";", "let i=0;", "i=0;"} {
		for _, condition := range []string{"", "i<3"} {
			for _, post := range []string{"", "i=i+1"} {
				body := "if(i>=3)break; y=y+1;"
				if post == "" {
					body += "i=i+1;"
				}
				source := fmt.Sprintf("let i=0; let y=0; for(%s%s;%s){%s}", init, condition, post, body)
				t.Run(source, func(t *testing.T) {
					values, _, err := runConditions(compileConditions(t, source))
					if err != nil || values["s1.y"] != float64(3) {
						t.Fatalf("values=%v err=%v", values, err)
					}
					wantOuter := float64(3)
					if init == "let i=0;" {
						wantOuter = 0
					}
					if values["s1.i"] != wantOuter {
						t.Fatalf("initializer binding: %v", values)
					}
				})
			}
		}
	}
}

func TestLoopStaticRejection(t *testing.T) {
	for _, source := range []string{
		"while(false){print(1);}", "do{print(1);}while(false);", "for(;false;){print(1);}",
		"for(;false;5%2){}", "let x=0; for(;false;x=(x=1)){}",
		"let x=0; for(;false;(x=1)+1){}", "let x=0; while(false){x=(x=1);}",
		"const x=0; for(;false;x=1){}", "for(;false;missing=1){}",
		"for(let i=0;false;i=i+1){} i=1;", "do{let x=true;}while(x);",
		"break;", "continue;", "while(false){} break;", "for(;;){break;} continue;",
		"foreach(x in [1,2]){print(x);}", "switch(true){case true:print(1);}",
		"let x=[1]; for(;false;x[0]=2%1){}", "while(1){}", "let x=0; for(;false;x=true){}",
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := bytes.Clone(input)
			analysis := Analyze(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) == 0 {
				t.Fatalf("expected atomic rejection: %+v", report)
			}
			if !bytes.Equal(input, before) || !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatal("rejection mutated inputs/scopes")
			}
			if len(analysis.Diagnostics) != 0 && !reflect.DeepEqual(report.Diagnostics, analysis.Diagnostics) {
				t.Fatal("semantic diagnostics changed")
			}
			first, _ := json.Marshal(report)
			second, _ := json.Marshal(Compile(input))
			if !bytes.Equal(first, second) {
				t.Fatal("rejection changed counters or diagnostics between calls")
			}
		})
	}
}

func TestLoopTemporaryPressure(t *testing.T) {
	// 200 iterations remain within the existing VM's 10,000-instruction bound.
	var peak int
	for _, count := range []int{1, 200} {
		source := fmt.Sprintf("let x=0; let y:float=0; while(x<%d){let z=(x+1)*(x+2); y=y+z/2; x=x+1;}", count)
		program := compileConditions(t, source)
		values, _, err := runConditions(program)
		if err != nil || values["s1.x"] != float64(count) {
			t.Fatalf("values=%v err=%v", values, err)
		}
		if count == 1 {
			peak = program.PeakTemporaries
		}
		if peak != program.PeakTemporaries || peak > 4 {
			t.Fatalf("peak=%d want=%d <=4", program.PeakTemporaries, peak)
		}
	}
}

func TestLoopHeaderAndPostTemporaries(t *testing.T) {
	// Each backedge recomputes a header with live logical operands after a post
	// and body that reuse the same temporary identities with different types.
	for _, count := range []int{1, 100} {
		source := fmt.Sprintf("let x=0; let y:float=0; for(;((x+1)<(%d+1))&&(true||false);(x=x+1)){let z=(x+2)*(x+3); y=y+z/2;}", count)
		program := compileConditions(t, source)
		values, _, err := runConditions(program)
		want := float64(0)
		for x := 0; x < count; x++ {
			want += float64((x+2)*(x+3)) / 2
		}
		if err != nil || values["s1.x"] != float64(count) || values["s1.y"] != want || program.PeakTemporaries > 4 {
			t.Fatalf("values=%v want y=%v peak=%d err=%v", values, want, program.PeakTemporaries, err)
		}
	}
}

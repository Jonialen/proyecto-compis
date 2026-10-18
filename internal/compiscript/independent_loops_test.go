package compiscript

import (
	"bytes"
	"fmt"
	"testing"

	"genanalex/internal/compiscript/ast"
	"genanalex/internal/compiscript/frontend"
)

func TestIndependentLoopForRoles(t *testing.T) {
	for _, init := range []string{";", "let i=0;", "i=0;"} {
		for _, condition := range []string{"", "i<3"} {
			for _, post := range []string{"", "i=i+1", "(i=i+1)", "false", "i+1"} {
				source := fmt.Sprintf("for(%s%s;%s){}", init, condition, post)
				t.Run(source, func(t *testing.T) {
					program, diagnostics := frontend.Parse([]byte(source))
					if len(diagnostics) != 0 {
						t.Fatal(diagnostics)
					}
					n := program.Statements[0].(ast.ForStmt)
					if (n.Init == nil) != (init == ";") || (n.Condition == nil) != (condition == "") || (n.Post == nil) != (post == "") {
						t.Fatalf("incorrect clause roles: %+v", n)
					}
				})
			}
		}
	}
}

func TestIndependentLoopContracts(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   float64
	}{
		{"let x=0;while(false){x=9;}", 0},
		{"let x=0;do{x=x+1;}while(false);", 1},
		{"let x=0;for(;;x=x+1){break;}", 0},
		{"let x=0;for(;;false){x=x+1;if(x==3)break;}", 3},
		{"let x=0;for(;x<3;((x=x+1))){}", 3},
		{"let x=0;for(let i=x+1;i<4;i=i+1){x=x+i;}", 6},
		{"let i=7;let x=0;for(let i=i-7;i<3;i=i+1){let i=i+10;x=x+i;}", 33},
		{"let x:float=0;for(let i=0;i<3;i=i+1){x=x+1/2;}", 1.5},
		{"let x=0;while(x<2&&(true||1/0>0)){x=x+1;}", 2},
	} {
		t.Run(tt.source, func(t *testing.T) {
			input := []byte(tt.source)
			before := bytes.Clone(input)
			report := Compile(input)
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report)
			}
			values, _, err := runConditions(report.TAC)
			if err != nil || values["s1.x"] != tt.want {
				t.Fatalf("values=%v error=%v, want x=%v", values, err, tt.want)
			}
			if !bytes.Equal(input, before) {
				t.Fatal("source mutated")
			}
		})
	}
}

func TestIndependentLoopAtomicErrors(t *testing.T) {
	for _, source := range []string{
		"break;", "continue;",
		"let x=0;while(false){print(x);}",
		"let x=0;for(;false;x=(x=1)){}",
		"let x=0;for(;false;(x=1)+2){}",
		"let x=0;while(false){(x=1);}",
		"for(let i=0;false;i=i+1){}let x=i;",
		"do{let x=1;}while(x>0);",
		"while(false){foreach(x in [1]){print(x);}}",
		"while(false){switch(1){case 1:break;}}",
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := bytes.Clone(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) == 0 {
				t.Fatalf("expected atomic rejection: %+v", report)
			}
			if !bytes.Equal(input, before) {
				t.Fatal("source mutated")
			}
		})
	}
}

package compiscript

import (
	"bytes"
	"fmt"
	"testing"
)

func TestIndependentConditionalContracts(t *testing.T) {
	for _, a := range []bool{false, true} {
		for _, b := range []bool{false, true} {
			t.Run(fmt.Sprintf("%t_%t", a, b), func(t *testing.T) {
				source := []byte(fmt.Sprintf("let result=((1+2)<4)==((%t || !%t) && (%t || !%t));", a, b, b, a))
				before := bytes.Clone(source)
				report := Compile(source)
				if report.TAC == nil || len(report.Diagnostics) != 0 {
					t.Fatalf("compile: %+v", report)
				}
				values, divisions, err := runConditions(report.TAC)
				want := (a || !b) && (b || !a)
				if err != nil || divisions != 0 || values["s1.result"] != want {
					t.Fatalf("values=%v divisions=%d error=%v, want result=%t", values, divisions, err, want)
				}
				if !bytes.Equal(source, before) {
					t.Fatal("source mutated")
				}
			})
		}
	}
	for _, source := range []string{
		"let result=false && (1/0>0);",
		"let result=!(true || (1/0>0));",
		"let x=10; if (true) {let x=x+2;} else {let x=x+4;} let result=x!=10;",
	} {
		t.Run(source, func(t *testing.T) {
			report := Compile([]byte(source))
			if report.TAC == nil || len(report.Diagnostics) != 0 {
				t.Fatalf("compile: %+v", report)
			}
			values, divisions, err := runConditions(report.TAC)
			if err != nil || divisions != 0 || values["s1.result"] != false {
				t.Fatalf("values=%v divisions=%d error=%v", values, divisions, err)
			}
		})
	}
}

func TestIndependentUnchosenSourceValidation(t *testing.T) {
	for _, source := range []string{
		"let result=false && missing;",
		"let result=true || ((true?[]:[])==null);",
		"let result=false && ((false?[]:[])==null);",
		"if (true) {} else {try{}catch(e){}}",
		"if (false) {foreach(x in [1,2]) {try{}catch(e){}}}",
		"if (false) {let x=missing;}",
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

package compiscript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"genanalex/internal/compiscript/tac"
)

func compileConditions(t *testing.T, source string) *tac.Program {
	t.Helper()
	input := []byte(source)
	before := sha256.Sum256(input)
	report := Compile(input)
	if report.TAC == nil || len(report.Diagnostics) != 0 {
		t.Fatalf("compile %q: %+v", source, report.Diagnostics)
	}
	if before != sha256.Sum256(input) || !reflect.DeepEqual(report.Scopes, Analyze(input).Scopes) {
		t.Fatal("source or semantic scopes changed")
	}
	first, _ := json.Marshal(report)
	second, _ := json.Marshal(Compile(input))
	if !bytes.Equal(first, second) {
		t.Fatal("nondeterministic report")
	}
	for _, instruction := range report.TAC.Instructions {
		if instruction.Span.Start.Line < 1 || instruction.Span.End.Offset <= instruction.Span.Start.Offset {
			t.Fatalf("invalid instruction span: %+v", instruction)
		}
	}
	return report.TAC
}

func TestConditionalScalarValues(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   any
	}{
		{"let x=true; x=!x;", false},
		{"const b=false; let x=!b;", true},
		{"let x=1<2;", true}, {"let x=2<=2;", true},
		{"let x=3>2.5;", true}, {"let x=2.0>=3;", false},
		{"let x=2==2.0;", true}, {"let x=2!=2.0;", false},
		{"let x=true==false;", false}, {"let x=true!=false;", true},
		{"let x=!(1+2>=4);", true},
	} {
		t.Run(tt.source, func(t *testing.T) {
			program := compileConditions(t, tt.source)
			values, _, err := runConditions(program)
			if err != nil || values["s1.x"] != tt.want {
				t.Fatalf("values=%v err=%v want x=%v", values, err, tt.want)
			}
			if strings.Contains(tt.source, "2.0") || strings.Contains(tt.source, "2.5") {
				if !strings.Contains(program.String(), "i2f") {
					t.Fatal("missing explicit widening")
				}
			}
		})
	}
}

func TestConditionalTruthTablesAndJoins(t *testing.T) {
	for _, a := range []bool{false, true} {
		for _, b := range []bool{false, true} {
			for _, tt := range []struct {
				expression string
				want       bool
			}{
				{"a&&b", a && b}, {"a||b", a || b},
				{"!(a&&(!b))||((!a)&&b)", !(a && !b) || (!a && b)},
				{"(a||b)==((a&&b)||((!a)&&(!b)))", (a || b) == ((a && b) || (!a && !b))},
			} {
				source := fmt.Sprintf("let a=%t; let b=%t; let x=%s;", a, b, tt.expression)
				t.Run(source, func(t *testing.T) {
					values, _, err := runConditions(compileConditions(t, source))
					if err != nil || values["s1.x"] != tt.want {
						t.Fatalf("values=%v err=%v want=%v", values, err, tt.want)
					}
				})
			}
		}
	}
}

func TestConditionalShortCircuitTraps(t *testing.T) {
	for _, tt := range []struct {
		expression string
		want       bool
		divisions  int
	}{
		{"false&&(1/0>0)", false, 0}, {"true||(1/0>0)", true, 0},
		{"true&&(1/0>0)", false, 1}, {"false||(1/0>0)", false, 1},
		{"!((false&&(1/0>0))||(!(2<3)))", true, 0},
		{"(1/2>0)&&(3/4>0)", true, 2},
		{"(1/0>0)==(2/1>0)", false, 1},
	} {
		t.Run(tt.expression, func(t *testing.T) {
			values, divisions, err := runConditions(compileConditions(t, "let x="+tt.expression+";"))
			trap := strings.Contains(tt.expression, "1/0") && tt.divisions == 1
			if divisions != tt.divisions || (trap && (err == nil || err.Error() != "division by zero")) || (!trap && (err != nil || values["s1.x"] != tt.want)) {
				t.Fatalf("values=%v divisions=%d err=%v", values, divisions, err)
			}
		})
	}
}

func TestConditionalIfScopes(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   any
	}{
		{"let x:float=0; if(true){x=1;}else{x=1/0;}", float64(1)},
		{"let x=0; if(false)x=1; else x=2;", float64(2)},
		{"let x=0; if(false)x=1;", float64(0)},
		{"let x=0; if(true){}else{x=2;}", float64(0)},
		{"let x=0; if(false)x=1; else if(true) {if(false)x=2;else x=3;} else x=4;", float64(3)},
		{"let x=true; if(true){let x=!x; x=true;}else{let x=!x;} x=!x;", false},
		{"let x:float=0; if(1<2)x=3; else x=4.5;", float64(3)},
	} {
		t.Run(tt.source, func(t *testing.T) {
			values, divisions, err := runConditions(compileConditions(t, tt.source))
			if err != nil || divisions != 0 || values["s1.x"] != tt.want {
				t.Fatalf("values=%v divisions=%d err=%v", values, divisions, err)
			}
		})
	}
}

func TestConditionalBranchBindings(t *testing.T) {
	for _, condition := range []bool{false, true} {
		source := fmt.Sprintf("let x=true; let y=true; if(%t){let x=!x; y=x;}else{let x=!x; y=x;}", condition)
		values, _, err := runConditions(compileConditions(t, source))
		selected, skipped := "s3.x", "s2.x"
		if condition {
			selected, skipped = skipped, selected
		}
		_, leaked := values[skipped]
		if err != nil || values["s1.x"] != true || values["s1.y"] != false || values[selected] != false || leaked {
			t.Fatalf("branch bindings: values=%v err=%v", values, err)
		}
	}
}

func TestConditionalStaticRejection(t *testing.T) {
	for _, source := range []string{
		`let x=false&&((true?[]:[])==null);`, `let x=true||((true?[]:[])==null);`,
		`if(false){try{missing;}catch(e){}}`, `if(true){}else{let s=[missing];}`,
		`if(false){foreach(x in [1,2]){try{missing;}catch(e){}}}`, `let x=true||(missing>0);`,
		`if(true){let x=1;} x=2;`, `let x=false&&(true+1>0);`,
		`let x=missing;`, `let x=true?1:missing;`, `if(false){let x=0; try{x=(x=missing);}catch(e){}}`,
		`if(false){switch(1){case 1: break;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := sha256.Sum256(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) == 0 {
				t.Fatalf("accepted invalid/unsupported source: %+v", report)
			}
			analysis := Analyze(input)
			if len(analysis.Diagnostics) > 0 && !reflect.DeepEqual(report.Diagnostics, analysis.Diagnostics) {
				t.Fatal("analysis errors not preserved")
			}
			if before != sha256.Sum256(input) || !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatal("rejection changed input/scopes")
			}
		})
	}
}

func TestConditionalRepeatedPressure(t *testing.T) {
	source := "let x=false;" + strings.Repeat("x=((1+2)<(3+4))==((false||true)&&((5+6)<(7+8)));", 100)
	program := compileConditions(t, source)
	values, _, err := runConditions(program)
	if err != nil || values["s1.x"] != true || program.PeakTemporaries > 6 {
		t.Fatalf("values=%v peak=%d err=%v", values, program.PeakTemporaries, err)
	}
}

package compiscript

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestSwitchExecution(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		want         float64
	}{
		{"first no fallthrough", "switch(true){case true:x=1; case false:x=2; default:x=3;}", 1},
		{"middle selector preserved", "switch(1<2){case false:x=1; case true:x=2; default:x=3;}", 2},
		{"default", "switch(true){case false:x=1; default:x=3;}", 3},
		{"no match", "switch(true){case false:x=1;}", 0},
		{"default only", "switch(false){default:x=3;}", 3},
		{"empty", "switch(true){}", 0},
		{"empty match exits", "switch(true){case true: case false:x=2; default:x=3;}", 0},
		{"explicit break", "switch(true){case true:x=1; break; default:x=3;}", 1},
		{"own local", "switch(true){case true:let y=3; x=y; default:x=2;}", 3},
		{"own constant", "switch(true){case true:const y=3; x=y;}", 3},
		{"own shadow initializer", "let y=3; {switch(true){case true:let y=y+1; x=y;}}", 4},
		{"real block", "let y=3; switch(true){case true:{let y=y+1; x=y;} default:x=y;} x=x+y;", 7},
		{"shared incoming", "let y=3; switch(false){case true:y=7; case false:x=y;}", 3},
		{"nested outer writes", "switch(true){case true:{x=1;} switch(false){case false:x=x+2; break;} x=x+3;}", 6},
		{"switch break in loop", "while(x<3){switch(true){case true:x=x+1; break;} x=x+1;}", 4},
		{"loop break in switch", "switch(true){case true:while(true){x=x+1; break;} x=x+2;}", 3},
		{"while continue nested switches", "while(x<3){x=x+1; switch(true){case true:switch(false){case false:continue;}} x=99;}", 3},
		{"do continue nested switches", "do{x=x+1; switch(true){case true:switch(false){case false:continue;}} x=99;}while(x<3);", 3},
		{"for continue nested switches", "for(;x<3;x=x+1){switch(true){case true:switch(false){case false:continue;}} x=99;}", 3},
		{"consecutive frames", "switch(true){case true:break;} while(true){x=x+1; break;} switch(false){default:x=x+1; break;} for(;;){x=x+1; break;}", 3},
		{"live selector complex rhs", "switch((1+2)<(3+4)){case ((2+3)*(4+5))<((1+2)*(2+3)):x=1; case (true||false)&&((5+6)<(7+8)):x=2;}", 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			program := compileConditions(t, "let x=0; "+tt.source)
			values, divisions, err := runConditions(program)
			if err != nil || divisions != 0 || values["s1.x"] != tt.want || program.PeakTemporaries > 7 {
				t.Fatalf("values=%v divisions=%d peak=%d err=%v want=%v", values, divisions, program.PeakTemporaries, err, tt.want)
			}
		})
	}
}

func TestSwitchOrderedEvaluation(t *testing.T) {
	for _, tt := range []struct {
		source    string
		divisions int
		trap      bool
	}{
		{"switch(2/1>0){case 1/1<0:x=1; case 2/1>0:x=2; case 1/0>0:x=3; default:x=4;}", 3, false},
		{"switch(true){case 1/0>0:x=1; case true:x=2;}", 1, true},
		{"switch(1/0>0){}", 1, true},
		{"switch(1/0>0){default:x=1;}", 1, true},
		{"switch(true){case true:x=2; default:x=1/0;}", 0, false},
		{"switch(2/1>0){} x=2;", 1, false},
		{"switch(2/1>0){default:x=2;}", 1, false},
		{"switch(true){case (1/1>0):x=2; case (2/1>0):x=3;}", 1, false},
	} {
		t.Run(tt.source, func(t *testing.T) {
			values, divisions, err := runConditions(compileConditions(t, "let x:float=0; "+tt.source))
			if divisions != tt.divisions || (tt.trap && (err == nil || err.Error() != "division by zero")) || (!tt.trap && (err != nil || values["s1.x"] != float64(2))) {
				t.Fatalf("values=%v divisions=%d err=%v", values, divisions, err)
			}
		})
	}
}

func TestSwitchAtomicRejection(t *testing.T) {
	for _, tt := range []struct{ source, code string }{
		{"switch(1){case 1:break;}", "SEM_CONDITION"},
		{"switch(true){case true:break; case true:break;}", "SEM_DUPLICATE_CASE"},
		{"switch(true){case 1:break;}", "SEM_CASE_TYPE"},
		{"switch(true){default:continue;}", "SEM_TRANSFER"},
		{"switch(true){case true:break; default:try{missing;}catch(e){}}", "SEM_UNRESOLVED"},
		{"switch(true){case true:break; case false:try{missing;}catch(e){}}", "SEM_UNRESOLVED"},
		{"switch(true){case true:break; case false:function f(a){missing;} break;}", "SEM_UNRESOLVED"},
		{"switch(true){case true:let a=1; case false:let b=a;}", "TAC_UNSUPPORTED"},
		{"switch(true){case true:let a=1;} let b=a;", "TAC_UNSUPPORTED"},
		{"switch(true){default:let a=1;} let b=a;", "TAC_UNSUPPORTED"},
		{"let a=9; {switch(false){case true:let a=1; case false:let b=a;}}", "TAC_UNSUPPORTED"},
		{"let a=9; {switch(true){case true:let a=1;} let b=a;}", "TAC_UNSUPPORTED"},
		{"let a=true; {switch(false){case true:let a=false; case a:break;}}", "TAC_UNSUPPORTED"},
		{"let a=9; {switch(false){case true:switch(true){case true:let a=1;} case false:let b=a;}}", "TAC_UNSUPPORTED"},
		{"switch(false){case true:let a=1; case false:a=2;}", "TAC_UNSUPPORTED"},
		{"switch(true){case true:const a=1;} let b=a;", "TAC_UNSUPPORTED"},
		{"let a=9; {switch(true){case true:switch(true){case true:let a=1;}} let b=a;}", "TAC_UNSUPPORTED"},
		{"let a=9; {switch(false){case true:let a=1; case false:{let b=a;}}}", "TAC_UNSUPPORTED"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			input := []byte(tt.source)
			before := bytes.Clone(input)
			analysis := Analyze(input)
			report := Compile(input)
			if report.TAC != nil || len(report.Diagnostics) == 0 || report.Diagnostics[0].Code != tt.code {
				t.Fatalf("expected %s atomic rejection: %+v", tt.code, report)
			}
			if !bytes.Equal(input, before) || !reflect.DeepEqual(report.Scopes, analysis.Scopes) || (len(analysis.Diagnostics) > 0 && !reflect.DeepEqual(report.Diagnostics, analysis.Diagnostics)) {
				t.Fatal("input or analysis changed")
			}
			if strings.Contains(tt.source, "let a=") && tt.code == "TAC_UNSUPPORTED" && !strings.Contains(report.Diagnostics[0].Message, "not guaranteed initialized") {
				t.Fatalf("missing explicit initialization restriction: %+v", report.Diagnostics)
			}
			first, _ := json.Marshal(report)
			second, _ := json.Marshal(Compile(input))
			if !bytes.Equal(first, second) {
				t.Fatal("rejection changed counters")
			}
		})
	}
}

func TestSwitchRepeatedPressure(t *testing.T) {
	unit := "switch((1+2)<(3+4)){case false:x=1; case (true||false):x=x+1;}"
	var peak int
	for _, count := range []int{1, 100} {
		program := compileConditions(t, "let x=0;"+strings.Repeat(unit, count))
		values, _, err := runConditions(program)
		if count == 1 {
			peak = program.PeakTemporaries
		}
		if err != nil || values["s1.x"] != float64(count) || program.PeakTemporaries != peak || peak > 5 {
			t.Fatalf("values=%v peak=%d err=%v", values, program.PeakTemporaries, err)
		}
	}
}

package compiscript

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestIndependentSwitchBlockedWrites(t *testing.T) {
	for _, source := range []string{
		"switch(true){default:let hidden=1;} hidden=2;",
		"switch(true){case 1<2:let hidden=1;} hidden=2;",
		"let hidden=9; {switch(false){case 1<2:let hidden=1; case 2<1:hidden=2;}}",
		"let hidden=9; {switch(true){case 1<2:switch(true){case 1<2:let hidden=1;}} hidden=2;}",
	} {
		t.Run(source, func(t *testing.T) {
			input := []byte(source)
			before := bytes.Clone(input)
			analysis := Analyze(input)
			report := Compile(input)
			if len(analysis.Diagnostics) != 0 || report.TAC != nil || len(report.Diagnostics) != 1 {
				t.Fatalf("analysis=%+v report=%+v", analysis.Diagnostics, report)
			}
			diagnostic := report.Diagnostics[0]
			if diagnostic.Code != "TAC_UNSUPPORTED" || diagnostic.Phase != "codegen" ||
				diagnostic.Message != "switch binding is not guaranteed initialized on this path" ||
				diagnostic.Span.Start.Offset != strings.LastIndex(source, "hidden") {
				t.Fatalf("diagnostic=%+v", diagnostic)
			}
			if !bytes.Equal(input, before) || !reflect.DeepEqual(report.Scopes, analysis.Scopes) {
				t.Fatal("source or semantic scope inventory changed")
			}
		})
	}
}

func TestIndependentSwitchChildDeclarations(t *testing.T) {
	for _, body := range []string{"if(true){let hidden=99;}", "while(false){let hidden=99;}"} {
		t.Run(body, func(t *testing.T) {
			source := "let x=0; let hidden=14; switch(false){case 1<2:" + body + "case 2<1:x=hidden;}"
			values, divisions, err := runConditions(compileConditions(t, source))
			if err != nil || divisions != 0 || values["s1.x"] != float64(14) {
				t.Fatalf("values=%v divisions=%d error=%v", values, divisions, err)
			}
		})
	}
}

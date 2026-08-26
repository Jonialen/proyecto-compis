package semantic_test

import (
	"strings"
	"testing"

	"genanalex/internal/compiscript/frontend"
	"genanalex/internal/compiscript/semantic"
)

func TestNamesTraversalAllControlForms(t *testing.T) {
	source := []byte(`while (missingWhile) { print(missingWhileBody); break; } do { print(missingDoBody); continue; } while (missingDo); for (let i: integer = 0; missingFor; i = missingPost) { print(missingForBody); } foreach (item in missingIterable) { print(missingForeachBody); } if (missingIf) { print(missingThen); } else { print(missingElse); } try { print(missingTry); } catch (err) { print(missingCatch); } switch (missingSwitch) { case 1: print(missingCase); default: print(missingDefault); } class Box { let field: integer = missingField; function method(): integer { return missingReturn; } }`)
	program, parseDiagnostics := frontend.Parse(source)
	if len(parseDiagnostics) != 0 {
		t.Fatalf("parse diagnostics: %+v", parseDiagnostics)
	}
	_, diagnostics := semantic.Analyze(program)
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == "unresolved name i" {
			t.Error("for initializer was not visible to post/body traversal")
		}
	}
	for _, name := range []string{"missingWhile", "missingWhileBody", "missingDoBody", "missingDo", "missingFor", "missingPost", "missingForBody", "missingIterable", "missingForeachBody", "missingIf", "missingThen", "missingElse", "missingTry", "missingCatch", "missingSwitch", "missingCase", "missingDefault", "missingField", "missingReturn"} {
		found := false
		for _, diagnostic := range diagnostics {
			found = found || strings.Contains(diagnostic.Message, name)
		}
		if !found {
			t.Errorf("no diagnostic reached %s", name)
		}
	}
}

package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDependencyTreeHasNoExternalOrGUIPackages(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, string(output))
	}

	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		pkg := strings.TrimSpace(line)
		if pkg == "" {
			continue
		}
		seen[pkg] = struct{}{}
		if pkg != "genanalex" && !strings.HasPrefix(pkg, "genanalex/") && !allowedFrontendDependency(pkg) {
			t.Fatalf("unexpected non-stdlib dependency %q found in dependency tree", pkg)
		}
		lower := strings.ToLower(pkg)
		if strings.Contains(lower, "fyne") || strings.Contains(lower, "gtk") || strings.Contains(lower, "qt") || strings.Contains(lower, "gio") || strings.Contains(lower, "webview") || strings.Contains(lower, "gui") {
			t.Fatalf("GUI-specific dependency %q found in dependency tree", pkg)
		}
	}

	if len(seen) == 0 {
		t.Fatal("dependency tree check saw no project packages, want at least the module packages")
	}

	imports := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
	output, err = imports.CombinedOutput()
	if err != nil {
		t.Fatalf("go list imports failed: %v\n%s", err, output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "genanalex/") {
			continue
		}
		for _, imported := range fields[1:] {
			if allowedFrontendDependency(imported) && !strings.HasPrefix(fields[0], "genanalex/internal/compiscript/frontend") {
				t.Fatalf("frontend dependency %q leaked into %q", imported, fields[0])
			}
		}
	}
}

func allowedFrontendDependency(pkg string) bool {
	return strings.HasPrefix(pkg, "github.com/antlr4-go/antlr/v4") || strings.HasPrefix(pkg, "golang.org/x/exp")
}

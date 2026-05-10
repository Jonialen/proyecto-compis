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
		if pkg != "genanalex" && !strings.HasPrefix(pkg, "genanalex/") {
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
}

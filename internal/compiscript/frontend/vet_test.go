package frontend

import (
	"os/exec"
	"testing"
)

func TestGeneratedParserPassesVet(t *testing.T) {
	if testing.Short() {
		t.Skip("generated parser verification invokes go vet")
	}
	// Force direct analysis: dependency-only vet cache entries can hide diagnostics.
	command := exec.Command("go", "vet", "-a", "./internal/compiscript/frontend/generated")
	command.Dir = repositoryRoot(t)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated parser vet failed: %v\n%s", err, output)
	}
}

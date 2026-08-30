package frontend

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestGenerationIsArgvSafeAndAtomic(t *testing.T) {
	root := repositoryRoot(t)
	space := filepath.Join(t.TempDir(), "paths with spaces")
	if err := os.MkdirAll(space, 0o755); err != nil {
		t.Fatal(err)
	}
	grammar := filepath.Join(space, "Compiscript grammar.g4")
	if err := os.WriteFile(grammar, []byte("grammar Compiscript; program: EOF;"), 0o644); err != nil {
		t.Fatal(err)
	}
	jar := filepath.Join(space, "antlr.jar")
	if err := os.WriteFile(jar, []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	checksum := filepath.Join(space, "antlr.sha256")
	writeChecksum(t, jar, checksum)

	for _, test := range []struct {
		name, java, gofmt string
		mutate            func()
		wantSuccess       bool
	}{
		{"spaced paths", fakeTool(t, space, "java-ok", fakeJava("package generated\n")), "gofmt", nil, true},
		{"missing java", filepath.Join(space, "missing-java"), "gofmt", nil, false},
		{"checksum mismatch", fakeTool(t, space, "java-unused", fakeJava("package generated\n")), "gofmt", func() { _ = os.WriteFile(checksum, []byte(strings.Repeat("0", 64)+"  antlr.jar\n"), 0o644) }, false},
		{"generation failure", fakeTool(t, space, "java-fails", "exit 7\n"), "gofmt", nil, false},
		{"formatting failure", fakeTool(t, space, "java-invalid", fakeJava("package generated\nfunc broken(\n")), fakeTool(t, space, "gofmt-fails", "exit 9\n"), nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			writeChecksum(t, jar, checksum)
			if test.mutate != nil {
				test.mutate()
			}
			output := filepath.Join(space, test.name, "generated output")
			if err := os.MkdirAll(output, 0o755); err != nil {
				t.Fatal(err)
			}
			old := []byte("previous output\n")
			if err := os.WriteFile(filepath.Join(output, "old.go"), old, 0o644); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(filepath.Join(root, "scripts", "generate-compiscript.sh"), "--grammar", grammar, "--output", output, "--jar", jar, "--checksum", checksum)
			command.Env = append(os.Environ(), "JAVA_BIN="+test.java, "GOFMT_BIN="+test.gofmt)
			run, err := command.CombinedOutput()
			if (err == nil) != test.wantSuccess {
				t.Fatalf("success=%t, output=%s, error=%v", test.wantSuccess, run, err)
			}
			if !test.wantSuccess {
				actual, readErr := os.ReadFile(filepath.Join(output, "old.go"))
				if readErr != nil || string(actual) != string(old) {
					t.Fatalf("failed generation changed output: %v %q", readErr, actual)
				}
			}
		})
	}
}

func TestGeneratedArtifactsArePortableAndReproducible(t *testing.T) {
	root := repositoryRoot(t)
	temporary := t.TempDir()
	workdirs := []string{filepath.Join(temporary, "first checkout path"), filepath.Join(temporary, "second checkout path")}
	outputs := []string{filepath.Join(temporary, "first output"), filepath.Join(temporary, "second output")}
	grammarSource, err := os.ReadFile(filepath.Join(root, "docs", "semestre2", "entrega1", "Compiscript.g4"))
	if err != nil {
		t.Fatal(err)
	}

	for i := range workdirs {
		if err := os.MkdirAll(workdirs[i], 0o755); err != nil {
			t.Fatal(err)
		}
		grammar := filepath.Join(workdirs[i], "Compiscript.g4")
		if err := os.WriteFile(grammar, grammarSource, 0o644); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(filepath.Join(root, "scripts", "generate-compiscript.sh"), "--grammar", grammar, "--output", outputs[i])
		command.Dir = workdirs[i]
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("generate from %q: %v\n%s", workdirs[i], err, output)
		}
	}

	first := generatedFiles(t, outputs[0])
	second := generatedFiles(t, outputs[1])
	if !reflect.DeepEqual(first, second) {
		t.Fatal("generation from distinct working directories produced different bytes")
	}
	committed := generatedFiles(t, filepath.Join(root, "internal", "compiscript", "frontend", "generated"))
	if !reflect.DeepEqual(first, committed) {
		t.Fatal("committed generated artifacts differ from portable regeneration")
	}

	absoluteHeader := regexp.MustCompile(`(?m)Code generated from (?:/|[A-Za-z]:[\\/])`)
	for set, files := range map[string]map[string][]byte{"committed": committed, "regenerated": first} {
		for name, contents := range files {
			for _, path := range append([]string{root}, workdirs...) {
				if bytes.Contains(contents, []byte(path)) {
					t.Errorf("%s %s contains checkout-specific path %q", set, name, path)
				}
			}
			if absoluteHeader.Match(contents) {
				t.Errorf("%s %s contains an absolute grammar path", set, name)
			}
		}
	}
}

func generatedFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[relative] = contents
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func fakeJava(goSource string) string {
	return "out=; while [ $# -gt 0 ]; do [ \"$1\" = -o ] && { shift; out=$1; }; shift; done\nmkdir -p \"$out\"\ncat > \"$out/compiscript.go\" <<'EOF'\n" + goSource + "EOF\n"
}

func fakeTool(t *testing.T, directory, name, body string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\nset -eu\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeChecksum(t *testing.T, jar, checksum string) {
	t.Helper()
	command := exec.Command("sha256sum", jar)
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checksum, output, 0o644); err != nil {
		t.Fatal(err)
	}
}

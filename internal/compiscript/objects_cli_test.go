package compiscript

import (
	"os"
	"testing"
)

func TestObjectCLIExecution(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, tt := range objectCases {
		t.Run(tt.name, func(t *testing.T) { assertClosureCLI(t, cli, tt.source, tt.want) })
	}
	example, err := os.ReadFile("../../testdata/compiscript/tac/objects.cps")
	if err != nil {
		t.Fatal(err)
	}
	assertClosureCLI(t, cli, string(example), map[string]any{"observed": float64(9), "first": float64(11), "second": float64(12), "independent": float64(21), "shared": float64(30), "label": "counter:ready"})
}

func TestObjectCLIAtomicRejection(t *testing.T) {
	cli := buildClosureCLI(t)
	for _, source := range []string{
		`class Unused{function bad(){print(1);}}`,
		`class Unused{let bad:integer[]=[1%2];}`,
		`class Box{const n:integer=1;}let box=new Box();box.n=2;`,
		`class Box{}let box=new Box();let x=box.missing;`,
		`class Box{function constructor(n:integer){}}let box=new Box(true);`,
	} {
		t.Run(source, func(t *testing.T) { assertClosureCLIRejection(t, cli, source) })
	}
}

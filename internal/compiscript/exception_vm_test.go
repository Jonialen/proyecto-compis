package compiscript

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"genanalex/internal/compiscript/tac"
)

// Only explicit language faults participate in source handlers. Metadata errors,
// budgets and Go failures remain ordinary errors; no host panic is recovered.
type testLanguageFault struct{ message string }

func (e *testLanguageFault) Error() string { return e.message }
func languageFaultf(format string, args ...any) error {
	return &testLanguageFault{message: fmt.Sprintf(format, args...)}
}

// A pointer gives each caught exception an opaque, stable reference identity.
type testException struct{ message string }
type testHandler struct {
	label        string
	environments map[int]*testEnvironment
}

func copyTestEnvironments(envs map[int]*testEnvironment) map[int]*testEnvironment {
	copy := make(map[int]*testEnvironment, len(envs))
	for id, env := range envs {
		copy[id] = env
	}
	return copy
}

func deliverTestFault(f *testFrame, err error) (*testFrame, bool) {
	var fault *testLanguageFault
	if !errors.As(err, &fault) {
		return f, false
	}
	for current := f; current != nil; {
		if n := len(current.handlers); n != 0 {
			handler := current.handlers[n-1]
			current.handlers = current.handlers[:n-1] // Removed before catch body.
			current.environments = handler.environments
			current.pending = &testException{message: fault.message}
			current.pc = current.labels[handler.label]
			return current, true
		}
		if current.procedure == nil {
			break
		}
		continuation, ok := current.cells[0].(frameContinuation)
		if !ok {
			return f, false
		}
		current = continuation.caller
	}
	return f, false
}

// Generic lists carry their original reference, not copied elements or mutated
// constraints. A typed use checks contents; known incompatible shapes stay strict.
func (f *testFrame) genericListFits(expected tac.Operand, list *testList) error {
	for _, value := range list.values {
		if _, err := f.checkedFrameValue(testListElement(expected), value); err != nil {
			return err
		}
	}
	return nil
}

func TestDynamicExceptionsFatalOracleFailures(t *testing.T) {
	for _, tt := range []struct {
		name, source, want, output string
		mutate                     func(*tac.Program)
	}{
		{name: "instruction budget", source: `try{while(true){}}catch(e){print("caught");}`, want: "instruction budget exceeded"},
		{name: "call depth budget", source: `function loop(){loop();}try{loop();}catch(e){print("caught");}`, want: "call depth budget exceeded"},
		{name: "undefined handler target", source: `try{print("before");}catch(e){print("caught");}`, want: "undefined label", mutate: func(p *tac.Program) { p.Instructions[0].Label = "missing" }},
		{name: "malformed opcode", source: `try{print("before");print(1);}catch(e){print("caught");}`, want: "unknown opcode", mutate: func(p *tac.Program) {
			for j := range p.Instructions {
				if p.Instructions[j].Op == tac.Print && p.Instructions[j].Left.Value == "1" {
					p.Instructions[j] = tac.Instruction{Op: "broken"}
					return
				}
			}
		}},
		{name: "malformed callee frame", source: `function f(){}try{f();}catch(e){print("caught");}`, want: "invalid frame size", mutate: func(p *tac.Program) { p.Procedures[0].Frame.Cells = -1 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := Compile([]byte(tt.source))
			if r.TAC == nil || len(r.Diagnostics) != 0 {
				t.Fatal(r.Diagnostics)
			}
			if tt.mutate != nil {
				tt.mutate(r.TAC)
			}
			_, out, err := runFunctionFramesOutput(r.TAC)
			var language *testLanguageFault
			if err == nil || !strings.Contains(err.Error(), tt.want) || errors.As(err, &language) || out != tt.output {
				t.Fatalf("output=%q err=%v want=%q output=%q", out, err, tt.want, tt.output)
			}
		})
	}
}

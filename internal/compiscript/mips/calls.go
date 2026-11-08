package mips

import (
	"fmt"

	"genanalex/internal/compiscript/tac"
)

func (g *generator) findProcedure(label string) *tac.Procedure {
	for n := range g.program.Procedures {
		if g.program.Procedures[n].Label == label {
			return &g.program.Procedures[n]
		}
	}
	return nil
}

// A dependency must have been declared, but need not already contain a value:
// a callee may initialize a write-only binding. Reads still reject its sentinel.
func (g *generator) requireBinding(o tac.Operand) error {
	if err := g.address(o, "$s3"); err != nil {
		return err
	}
	g.emit("    lw $s3, 0($s3)")
	g.emit("    beq $s3, $zero, rt_undefined")
	return nil
}

// The argument vector belongs to the caller; the callee copies it into its own
// frame before evaluating anything. There is no four-register argument limit.
func (g *generator) stageArguments(args []tac.Operand) (int, error) {
	size := (len(args)*4 + 7) &^ 7
	if size > 32760 {
		return 0, fmt.Errorf("MIPS argument vector exceeds offset range")
	}
	if size != 0 {
		g.emit("    addiu $sp, $sp, -%d", size)
	}
	for n, arg := range args {
		if err := g.load(arg, "$s3"); err != nil {
			return 0, err
		}
		g.emit("    sw $s3, %d($sp)", n*4)
	}
	return size, nil
}
func (g *generator) finishCall(i tac.Instruction, size int) error {
	if size != 0 {
		g.emit("    addiu $sp, $sp, %d", size)
	}
	if i.Target.Kind != "" {
		return g.store(i.Target, "$v0")
	}
	return nil
}
func (g *generator) directCall(i tac.Instruction) error {
	p := g.findProcedure(i.Callee)
	if p == nil {
		return fmt.Errorf("undefined procedure %q", i.Callee)
	}
	if len(i.Arguments) != len(p.Parameters) {
		return fmt.Errorf("argument count mismatch")
	}
	for _, global := range p.Globals {
		if err := g.requireBinding(global); err != nil {
			return err
		}
	}
	g.emit("    move $s7, $zero")
	if p.LexicalScope != 0 {
		g.environment(p.LexicalScope, "$s7")
		g.emit("    beq $s7, $zero, rt_undefined")
	}
	size, err := g.stageArguments(i.Arguments)
	if err != nil {
		return err
	}
	g.emit("    move $a0, $sp")
	g.emit("    move $a1, $s7")
	g.emit("    move $a2, $zero")
	g.emit("    jal %s", g.procedures[p.Label])
	return g.finishCall(i, size)
}

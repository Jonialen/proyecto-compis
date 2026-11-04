package mips

import (
	"fmt"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

func (g *generator) layout(id int) *tac.EnvironmentLayout {
	for n := range g.program.Environments {
		e := &g.program.Environments[n]
		if e.ScopeID == id {
			return e
		}
	}
	return nil
}
func operandScope(o tac.Operand) int {
	id, _ := strconv.Atoi(strings.TrimPrefix(strings.SplitN(o.Value, ".", 2)[0], "s"))
	return id
}
func (g *generator) validateEnvironments() error {
	seen := map[int]bool{}
	for _, e := range g.program.Environments {
		if e.ScopeID <= 0 || seen[e.ScopeID] {
			return fmt.Errorf("MIPS: duplicate/invalid environment")
		}
		seen[e.ScopeID] = true
		cells := map[string]bool{}
		for _, o := range e.Bindings {
			if o.Kind != tac.Symbol || operandScope(o) != e.ScopeID || cells[o.Value] {
				return fmt.Errorf("MIPS: invalid environment binding")
			}
			cells[o.Value] = true
		}
	}
	for _, e := range g.program.Environments {
		chain := map[int]bool{}
		for id := e.ScopeID; id != 0; {
			if chain[id] {
				return fmt.Errorf("MIPS: cyclic environment")
			}
			chain[id] = true
			p := g.layout(id)
			if p == nil {
				return fmt.Errorf("MIPS: undefined parent environment")
			}
			id = p.ParentID
		}
	}
	return nil
}
func (g *generator) environment(id int, reg string) {
	if id == 0 {
		g.emit("    move %s, $zero", reg)
		return
	}
	g.emit("    li $a0, %d", id)
	g.emit("    jal rt_find_environment")
	g.emit("    move %s, $v0", reg)
}
func (g *generator) environmentAddress(o tac.Operand, reg string) error {
	e := g.layout(operandScope(o))
	if e == nil {
		return fmt.Errorf("undefined operand environment: %s", o)
	}
	for n, b := range e.Bindings {
		if b.Value == o.Value {
			if 12+n*4 > 32760 {
				return fmt.Errorf("environment exceeds offset range")
			}
			g.environment(e.ScopeID, reg)
			g.emit("    beq %s, $zero, rt_undefined", reg)
			g.emit("    addiu %s, %s, %d", reg, reg, 12+n*4)
			return nil
		}
	}
	return fmt.Errorf("environment binding out of bounds: %s", o)
}
func (g *generator) closureInstruction(i tac.Instruction) error {
	switch i.Op {
	case tac.EnterEnvironment:
		e := g.layout(i.Environment)
		if e == nil {
			return fmt.Errorf("undefined environment layout")
		}
		g.environment(e.ParentID, "$s7")
		if e.ParentID != 0 {
			g.emit("    beq $s7, $zero, rt_undefined")
		}
		g.emit("    li $a0, %d", 12+len(e.Bindings)*4)
		g.emit("    jal rt_alloc")
		g.emit("    move $s2, $v0")
		g.emit("    li $t0, %d", e.ScopeID)
		g.emit("    sw $t0, 0($s2)")
		g.emit("    sw $s7, 4($s2)")
		g.emit("    sw $s1, 8($s2)")
		for n := range e.Bindings {
			g.emit("    sw $zero, %d($s2)", 12+n*4)
		}
		if g.procedure != nil && g.procedure.ScopeID == e.ScopeID {
			formals := append([]tac.Operand{}, g.procedure.Parameters...)
			if g.procedure.Receiver != nil {
				formals = append(formals, *g.procedure.Receiver)
			}
			for _, o := range formals {
				found := false
				for n, b := range e.Bindings {
					if b.Value == o.Value {
						g.emit("    lw $t0, %d($fp)", g.homes[key(o)])
						g.emit("    sw $t0, %d($s2)", 12+n*4)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("parameter missing from environment")
				}
			}
		}
		g.emit("    move $s1, $s2")
		return nil
	case tac.MakeClosure:
		p := g.findProcedure(i.Callee)
		if p == nil || p.LexicalScope != i.Environment || g.layout(i.Environment) == nil {
			return fmt.Errorf("invalid closure procedure/environment")
		}
		g.environment(i.Environment, "$s7")
		g.emit("    beq $s7, $zero, rt_undefined")
		g.emit("    li $a0, 16")
		g.emit("    jal rt_alloc")
		g.emit("    li $t0, 7")
		g.emit("    sw $t0, 0($v0)")
		g.emit("    la $t0, %s", g.procedures[p.Label])
		g.emit("    sw $t0, 4($v0)")
		g.emit("    sw $s7, 8($v0)")
		g.emit("    sw $zero, 12($v0)")
		return g.store(i.Target, "$v0")
	case tac.CallIndirect:
		return g.indirectCall(i)
	}
	return fmt.Errorf("invalid closure instruction")
}
func listShape(t model.Type) string {
	if t.Kind != model.TypeList || t.Element == nil {
		return ""
	}
	s := string(t.Element.Kind)
	if t.Element.Kind == model.TypeList {
		s += "/" + listShape(*t.Element)
	}
	return s
}
func signatureMatches(i tac.Instruction, p *tac.Procedure) bool {
	if i.Guarded {
		return len(i.Arguments) == len(p.Parameters)
	}
	s := i.Signature
	if s == nil || s.Kind != model.TypeFunction || s.Result == nil || s.Result.Kind != p.ResultKind || len(s.Params) != len(p.Parameters) || listShape(*s.Result) != p.ResultShape {
		return false
	}
	for n, formal := range p.Parameters {
		if formal.Type != s.Params[n].Kind || formal.Shape != listShape(s.Params[n]) {
			return false
		}
	}
	for n, formal := range p.Parameters {
		if len(i.TypeClasses) == 0 {
			if formal.Class != "" {
				return false
			}
		} else if len(i.TypeClasses) != len(p.Parameters)+1 || i.TypeClasses[n] != formal.Class {
			return false
		}
	}
	if len(i.TypeClasses) == 0 {
		if p.ResultClass != "" {
			return false
		}
	} else if len(i.TypeClasses) != len(p.Parameters)+1 || i.TypeClasses[len(p.Parameters)] != p.ResultClass {
		return false
	}
	return true
}
func (g *generator) indirectCall(i tac.Instruction) error {
	if !i.Guarded && i.Signature == nil {
		return fmt.Errorf("missing indirect call signature")
	}
	if err := g.load(i.Left, "$s7"); err != nil {
		return err
	}
	g.emit("    lw $t0, 0($s7)")
	g.emit("    bne $t0, 7, rt_function_error")
	size, err := g.stageArguments(i.Arguments)
	if err != nil {
		return err
	}
	ready := g.unique("call_ready")
	g.emit("    lw $t0, 4($s7)")
	for n := range g.program.Procedures {
		p := &g.program.Procedures[n]
		if signatureMatches(i, p) {
			g.emit("    la $t1, %s", g.procedures[p.Label])
			g.emit("    beq $t0, $t1, %s", ready)
		}
	}
	g.emit("    j rt_signature_error")
	g.emit("%s:", ready)
	g.emit("    move $a0, $sp")
	g.emit("    lw $a1, 8($s7)")
	g.emit("    lw $a2, 12($s7)")
	g.emit("    lw $t9, 4($s7)")
	g.emit("    jalr $t9")
	return g.finishCall(i, size)
}

const environmentRuntime = `
# Search latest frame-owned entries first, then the retained lexical chain.
rt_find_environment:
    move $t0, $s1
rt_find_local:
    beq $t0, $zero, rt_find_lexical_start
    lw $t1, 0($t0)
    beq $t1, $a0, rt_find_found
    lw $t0, 8($t0)
    j rt_find_local
rt_find_lexical_start:
    move $t0, $s0
rt_find_lexical:
    beq $t0, $zero, rt_find_found
    lw $t1, 0($t0)
    beq $t1, $a0, rt_find_found
    lw $t0, 4($t0)
    j rt_find_lexical
rt_find_found:
    move $v0, $t0
    jr $ra
rt_function_error:
    la $a0, rt_msg_function
    j rt_fault
rt_signature_error:
    la $a0, rt_msg_signature
    j rt_fault
.data
rt_msg_function: .asciiz "invalid function value"
rt_msg_signature: .asciiz "indirect call signature mismatch"
.text
`

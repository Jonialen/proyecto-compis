package mips

import (
	"fmt"

	"genanalex/internal/compiscript/tac"
)

func (g *generator) exceptionInstruction(i tac.Instruction) error {
	// Validate control-flow, not textual nesting: lowering emits additional pops
	// on return/break/continue edges, including unreachable cleanup afterwards.
	tape := g.program.Instructions
	if g.procedure != nil {
		tape = g.procedure.Instructions
	}
	if err := validateExceptionFlow(tape); err != nil {
		return err
	}
	switch i.Op {
	case tac.PushHandler:
		target, err := g.jumpLabel(i.Label)
		if err != nil {
			return err
		}
		// Heap record: previous/target/sp/fp/ra, s0..s7, pending. No
		// frame homes move, and allocation does not modify saved registers.
		g.emit("    move $t9, $ra")
		g.emit("    li $a0, 56")
		g.emit("    jal rt_alloc")
		g.emit("    lw $t0, rt_handler")
		g.emit("    sw $t0, 0($v0)")
		g.emit("    la $t0, %s", target)
		g.emit("    sw $t0, 4($v0)")
		g.emit("    sw $sp, 8($v0)")
		g.emit("    sw $fp, 12($v0)")
		g.emit("    sw $t9, 16($v0)")
		for n := 0; n < 8; n++ {
			g.emit("    sw $s%d, %d($v0)", n, 20+n*4)
		}
		g.emit("    lw $t0, rt_pending_exception")
		g.emit("    sw $t0, 52($v0)")
		g.emit("    sw $v0, rt_handler")
		g.emit("    move $ra, $t9")
		return nil
	case tac.PopHandler:
		g.emit("    lw $t0, rt_handler")
		g.emit("    beq $t0, $zero, rt_handler_invalid")
		g.emit("    lw $t1, 12($t0)")
		g.emit("    bne $t1, $fp, rt_handler_invalid")
		g.emit("    lw $t1, 52($t0)")
		g.emit("    sw $t1, rt_pending_exception")
		g.emit("    lw $t0, 0($t0)")
		g.emit("    sw $t0, rt_handler")
		return nil
	case tac.CatchException:
		g.emit("    lw $s3, rt_pending_exception")
		g.emit("    beq $s3, $zero, rt_handler_invalid")
		if err := g.store(i.Target, "$s3"); err != nil {
			return err
		}
		g.emit("    sw $zero, rt_pending_exception")
		return nil
	}
	return fmt.Errorf("invalid exception opcode %q", i.Op)
}

// Exception edges restore the push-time depth and supply one pending value.
// Reject malformed reachable cleanup atomically; it is not a language fault.
func validateExceptionFlow(tape []tac.Instruction) error {
	type state struct {
		pc, depth int
		pending   bool
	}
	labels := map[string]int{}
	for pc, i := range tape {
		if i.Op == tac.Label {
			labels[i.Label] = pc
		}
	}
	queue := []state{{}}
	seen := map[int]state{}
	for len(queue) != 0 {
		s := queue[0]
		queue = queue[1:]
		if s.pc == len(tape) {
			if s.depth != 0 || s.pending {
				return fmt.Errorf("unterminated exception state at end of procedure")
			}
			continue
		}
		if old, ok := seen[s.pc]; ok {
			if old.depth != s.depth || old.pending != s.pending {
				return fmt.Errorf("inconsistent exception state at instruction %d", s.pc)
			}
			continue
		}
		seen[s.pc] = s
		i := tape[s.pc]
		next := s
		next.pc++
		edge := func(pending bool) error {
			pc, ok := labels[i.Label]
			if !ok {
				return fmt.Errorf("undefined handler/control label %q", i.Label)
			}
			queue = append(queue, state{pc: pc, depth: s.depth, pending: pending})
			return nil
		}
		switch i.Op {
		case tac.PushHandler:
			if err := edge(true); err != nil {
				return err
			}
			next.depth++
		case tac.PopHandler:
			if s.depth == 0 {
				return fmt.Errorf("handler stack underflow at instruction %d", s.pc)
			}
			next.depth--
		case tac.CatchException:
			if !s.pending {
				return fmt.Errorf("catch without pending exception at instruction %d", s.pc)
			}
			next.pending = false
		case tac.Return:
			if s.depth != 0 {
				return fmt.Errorf("return with active handler at instruction %d", s.pc)
			}
			continue
		case tac.Jump, tac.BranchTrue:
			if err := edge(s.pending); err != nil {
				return err
			}
			if i.Op == tac.Jump {
				continue
			}
		}
		queue = append(queue, next)
	}
	return nil
}

const exceptionRuntime = `
.data
.align 2
rt_handler: .word 0
rt_pending_exception: .word 0
rt_msg_handler_invalid: .asciiz "invalid exception handler state"
.text
# Fault entry: a0 is stable NUL-terminated ASCII text. Never return to the
# faulting helper: its scratch stack and intervening frames are abandoned.
rt_raise:
    lw $t0, rt_handler
    beq $t0, $zero, rt_exception_uncaught
    addiu $sp, $sp, -16
    sw $a0, 0($sp)
    move $t0, $a0
rt_exception_length:
    lbu $t1, 0($t0)
    beq $t1, $zero, rt_exception_box
    addiu $t0, $t0, 1
    j rt_exception_length
rt_exception_box:
    subu $t0, $t0, $a0
    sw $t0, 4($sp)
    li $a0, 16
    jal rt_alloc
    li $t0, 9
    sw $t0, 0($v0)
    lw $t0, 0($sp)
    sw $t0, 4($v0)
    lw $t0, 4($sp)
    sw $t0, 8($v0)
    sw $zero, 12($v0)
    sw $v0, rt_pending_exception
    lw $t0, rt_handler
    lw $t1, 0($t0)
    sw $t1, rt_handler
    lw $t9, 4($t0)
    lw $sp, 8($t0)
    lw $fp, 12($t0)
    lw $ra, 16($t0)
    lw $s0, 20($t0)
    lw $s1, 24($t0)
    lw $s2, 28($t0)
    lw $s3, 32($t0)
    lw $s4, 36($t0)
    lw $s5, 40($t0)
    lw $s6, 44($t0)
    lw $s7, 48($t0)
    jr $t9
# Defensive invalid-IR path bypasses all source handlers.
rt_handler_invalid:
    la $a0, rt_msg_handler_invalid
rt_exception_uncaught:
    li $v0, 4
    syscall
    li $v0, 4
    la $a0, rt_newline
    syscall
    li $a0, 1
    li $v0, 17
    syscall
`

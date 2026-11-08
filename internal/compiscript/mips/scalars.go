package mips

import (
	"fmt"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// Static TAC must remain a typed contract. Runtime guards are reserved for
// deliberately guarded instructions, not recovery from corrupted compiler IR.
func validateScalar(i tac.Instruction) error {
	unary := i.Op == tac.Negate || i.Op == tac.IntegerToFloat || i.Op == tac.LogicalNot
	if unary && i.Right != nil || !unary && i.Right == nil {
		return fmt.Errorf("invalid unary/binary operands")
	}
	if i.Guarded {
		return nil
	}
	numeric := func(k model.TypeKind) bool { return k == model.TypeInteger || k == model.TypeFloat }
	if unary {
		switch i.Op {
		case tac.LogicalNot:
			if i.Left.Type == model.TypeBoolean && i.Target.Type == model.TypeBoolean {
				return nil
			}
		case tac.IntegerToFloat:
			if i.Left.Type == model.TypeInteger && i.Target.Type == model.TypeFloat {
				return nil
			}
		case tac.Negate:
			if numeric(i.Left.Type) && i.Target.Type == i.Left.Type {
				return nil
			}
		}
	} else if i.Op == tac.Equal || i.Op == tac.NotEqual {
		if i.Target.Type == model.TypeBoolean {
			return nil
		}
	} else if i.Op == tac.Add && i.Left.Type == model.TypeString &&
		i.Right.Type == model.TypeString && i.Target.Type == model.TypeString {
		return nil
	} else if numeric(i.Left.Type) && i.Left.Type == i.Right.Type {
		comparison := i.Op == tac.Less || i.Op == tac.LessEqual || i.Op == tac.Greater || i.Op == tac.GreaterEqual
		if comparison && i.Target.Type == model.TypeBoolean || !comparison && i.Target.Type == i.Left.Type {
			return nil
		}
	}
	return fmt.Errorf("invalid static scalar type contract: %s", i.Op)
}

func (g *generator) scalar(i tac.Instruction) error {
	if err := validateScalar(i); err != nil {
		return err
	}
	unary := i.Op == tac.Negate || i.Op == tac.IntegerToFloat || i.Op == tac.LogicalNot
	if err := g.load(i.Left, "$s3"); err != nil {
		return err
	}
	if i.Right != nil {
		if err := g.load(*i.Right, "$s4"); err != nil {
			return err
		}
	}
	if i.Op == tac.Equal || i.Op == tac.NotEqual {
		if i.Guarded {
			ready := g.unique("equality_tags")
			g.emit("    lw $t0, 0($s3)")
			g.emit("    lw $t1, 0($s4)")
			g.emit("    beq $t0, $t1, %s", ready)
			g.emit("    beq $t0, 8, %s", ready)
			g.emit("    beq $t1, 8, %s", ready)
			g.emit("    bgt $t0, 2, ref_binary_error")
			g.emit("    bgt $t1, 2, ref_binary_error")
			g.emit("%s:", ready)
		}
		g.emit("    move $a0, $s3")
		g.emit("    move $a1, $s4")
		g.emit("    jal rt_equal")
		if i.Op == tac.NotEqual {
			g.emit("    xori $v0, $v0, 1")
		}
		g.emit("    move $t0, $v0")
		g.boolResult("$t0")
		return g.store(i.Target, "$s5")
	}
	if i.Op == tac.LogicalNot {
		g.emit("    lw $t0, 0($s3)")
		if i.Guarded {
			valid := g.unique("negation_boolean")
			g.emit("    beq $t0, 3, %s", valid)
			g.fault("invalid boolean negation")
			g.emit("%s:", valid)
		} else {
			g.emit("    bne $t0, 3, rt_boolean_error")
		}
		g.emit("    lw $t0, 4($s3)")
		g.emit("    xori $t0, $t0, 1")
		g.boolResult("$t0")
		return g.store(i.Target, "$s5")
	}
	if i.Op == tac.Add && i.Guarded {
		numeric, done := g.unique("add_numeric"), g.unique("add_done")
		g.emit("    lw $t0, 0($s3)")
		g.emit("    bne $t0, 4, %s", numeric)
		g.emit("    lw $t0, 0($s4)")
		g.emit("    bne $t0, 4, %s", numeric)
		g.emit("    move $a0, $s3")
		g.emit("    move $a1, $s4")
		g.emit("    jal rt_concat")
		if err := g.store(i.Target, "$v0"); err != nil {
			return err
		}
		g.emit("    j %s", done)
		g.emit("%s:", numeric)
		if err := g.numericScalar(i, unary); err != nil {
			return err
		}
		g.emit("%s:", done)
		return nil
	}
	if i.Op == tac.Add && i.Target.Type == model.TypeString {
		g.emit("    move $a0, $s3")
		g.emit("    move $a1, $s4")
		g.emit("    jal rt_concat")
		return g.store(i.Target, "$v0")
	}
	return g.numericScalar(i, unary)
}

func (g *generator) numericScalar(i tac.Instruction, unary bool) error {
	g.emit("    lw $t0, 0($s3)")
	g.emit("    blt $t0, 1, rt_numeric_error")
	g.emit("    bgt $t0, 2, rt_numeric_error")
	g.emit("    l.d $f0, 8($s3)")
	if !unary {
		g.emit("    lw $t1, 0($s4)")
		g.emit("    blt $t1, 1, rt_numeric_error")
		g.emit("    bgt $t1, 2, rt_numeric_error")
		g.emit("    l.d $f2, 8($s4)")
	}
	switch i.Op {
	case tac.Add:
		g.emit("    add.d $f4, $f0, $f2")
	case tac.Subtract:
		g.emit("    sub.d $f4, $f0, $f2")
	case tac.Multiply:
		g.emit("    mul.d $f4, $f0, $f2")
	case tac.Divide, tac.Remainder:
		g.emit("    mtc1 $zero, $f6")
		g.emit("    cvt.d.w $f6, $f6")
		g.emit("    c.eq.d $f2, $f6")
		fault := "rt_division_error"
		if i.Op == tac.Remainder {
			fault = "rt_remainder_error"
		}
		g.emit("    bc1t %s", fault)
		g.emit("    div.d $f4, $f0, $f2")
		if i.Op == tac.Remainder {
			g.emit("    jal rt_fmod")
		}
	case tac.Negate:
		g.emit("    neg.d $f4, $f0")
	case tac.IntegerToFloat:
		g.emit("    mov.d $f4, $f0")
	case tac.Less, tac.LessEqual, tac.Greater, tac.GreaterEqual:
		a, b := "$f0", "$f2"
		if i.Op == tac.Greater || i.Op == tac.GreaterEqual {
			a, b = b, a
		}
		op := "c.lt.d"
		if i.Op == tac.LessEqual || i.Op == tac.GreaterEqual {
			op = "c.le.d"
		}
		yes, done := g.unique("true"), g.unique("compared")
		g.emit("    %s %s, %s", op, a, b)
		g.emit("    bc1t %s", yes)
		g.emit("    la $s5, rt_false")
		g.emit("    j %s", done)
		g.emit("%s:", yes)
		g.emit("    la $s5, rt_true")
		g.emit("%s:", done)
		return g.store(i.Target, "$s5")
	}
	tag := tagInteger
	if i.Target.Type == model.TypeFloat {
		tag = tagFloat
	}
	g.emit("    li $a0, 16")
	g.emit("    jal rt_alloc")
	g.emit("    li $t0, %d", tag)
	if i.Guarded {
		if i.Op == tac.Divide || i.Op == tac.IntegerToFloat {
			g.emit("    li $t0, 2")
		} else {
			g.emit("    lw $t0, 0($s3)")
			if !unary {
				g.emit("    lw $t1, 0($s4)")
				g.emit("    or $t0, $t0, $t1")
				// Integer/float tags are 1/2: either float makes the OR >=2.
				integer := g.unique("integer_result")
				g.emit("    beq $t0, 1, %s", integer)
				g.emit("    li $t0, 2")
				g.emit("%s:", integer)
			}
		}
	}
	g.emit("    sw $t0, 0($v0)")
	g.emit("    s.d $f4, 8($v0)")
	return g.store(i.Target, "$v0")
}
func (g *generator) boolResult(reg string) {
	yes, done := g.unique("bool_true"), g.unique("bool_done")
	g.emit("    bne %s, $zero, %s", reg, yes)
	g.emit("    la $s5, rt_false")
	g.emit("    j %s", done)
	g.emit("%s:", yes)
	g.emit("    la $s5, rt_true")
	g.emit("%s:", done)
}

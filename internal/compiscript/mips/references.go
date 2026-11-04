package mips

import (
	"fmt"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// Objects: tag, class descriptor, retained-environment vector, reserved; fields
// follow at 16. Lists: tag, element descriptor, length, reserved; elements at 16.
func (g *generator) class(id string) *tac.ClassLayout {
	for n := range g.program.Classes {
		if g.program.Classes[n].ID == id {
			return &g.program.Classes[n]
		}
	}
	return nil
}

func (g *generator) classNumber(id string) int {
	for n, c := range g.program.Classes {
		if c.ID == id {
			return n
		}
	}
	return -1
}

func (g *generator) classLabel(id string) string {
	return fmt.Sprintf("ref_class_%d", g.classNumber(id))
}

func (g *generator) subclass(actual, expected string) bool {
	seen := map[string]bool{}
	for actual != "" && !seen[actual] {
		if actual == expected {
			return true
		}
		seen[actual] = true
		c := g.class(actual)
		if c == nil {
			return false
		}
		actual = c.Parent
	}
	return false
}

// Reject malformed metadata during generation, never through a source handler.
func (g *generator) validateClasses() error {
	seen := map[string]bool{}
	for _, c := range g.program.Classes {
		if c.ID == "" || seen[c.ID] {
			return fmt.Errorf("invalid class identity")
		}
		seen[c.ID] = true
		chain := map[string]bool{}
		for id := c.ID; id != ""; {
			if chain[id] {
				return fmt.Errorf("cyclic class layout")
			}
			chain[id] = true
			p := g.class(id)
			if p == nil {
				return fmt.Errorf("undefined base class")
			}
			id = p.Parent
		}
		init := g.findProcedure(c.Initializer)
		if init == nil || init.Receiver == nil || init.Receiver.Class != c.ID ||
			init.LexicalScope != c.LexicalScope || g.layout(c.LexicalScope) == nil {
			return fmt.Errorf("undefined class initializer/environment")
		}
		if 16+4*(len(c.Fields)+len(g.program.Classes)) > 32760 {
			return fmt.Errorf("object layout exceeds offset range")
		}
		fields := map[string]bool{}
		for n, f := range c.Fields {
			if f.Offset != n || f.ID == "" || fields[f.ID] || !g.subclass(c.ID, f.Owner) {
				return fmt.Errorf("invalid field offset/owner")
			}
			fields[f.ID] = true
		}
		if p := g.class(c.Parent); p != nil {
			if len(p.Fields) > len(c.Fields) {
				return fmt.Errorf("invalid inherited fields")
			}
			for n, f := range p.Fields {
				if c.Fields[n].ID != f.ID || c.Fields[n].Offset != f.Offset {
					return fmt.Errorf("invalid inherited field offset")
				}
			}
		}
		methods := map[string]bool{}
		for _, m := range c.Methods {
			p := g.findProcedure(m.Callee)
			if methods[m.Name] || p == nil || p.Receiver == nil ||
				p.Receiver.Class != m.Owner || !g.subclass(c.ID, m.Owner) {
				return fmt.Errorf("invalid method metadata")
			}
			methods[m.Name] = true
		}
	}
	return nil
}

func element(o tac.Operand) tac.Operand {
	kind, rest, _ := strings.Cut(o.Shape, "/")
	if kind == "" {
		kind = string(model.TypeError)
	}
	return tac.Operand{Type: model.TypeKind(kind), Shape: rest, Class: o.Class}
}

func runtimeTag(k model.TypeKind) (int, error) {
	switch k {
	case model.TypeError:
		return 0, nil
	case model.TypeInteger:
		return tagInteger, nil
	case model.TypeFloat:
		return tagFloat, nil
	case model.TypeBoolean:
		return tagBoolean, nil
	case model.TypeString:
		return tagString, nil
	case model.TypeClass:
		return tagObject, nil
	case model.TypeList:
		return tagList, nil
	case model.TypeFunction:
		return tagFunction, nil
	case model.TypeNull:
		return tagNull, nil
	}
	return 0, fmt.Errorf("invalid value type %q", k)
}

// Descriptors are immutable pairs: tag (zero means unknown), child descriptor
// for a list, nominal class descriptor for an object, or zero for scalars.
func (g *generator) typeDescriptor(o tac.Operand) (string, error) {
	tag, err := runtimeTag(o.Type)
	if err != nil {
		return "", err
	}
	child := "0"
	if o.Type == model.TypeList {
		child, err = g.typeDescriptor(element(o))
		if err != nil {
			return "", err
		}
	}
	if o.Type == model.TypeClass {
		if g.class(o.Class) == nil {
			return "", fmt.Errorf("undefined nominal class %q", o.Class)
		}
		child = g.classLabel(o.Class)
	}
	label := g.unique("type")
	fmt.Fprintf(&g.data, ".align 2\n%s: .word %d, %s\n", label, tag, child)
	return label, nil
}

func (g *generator) checkValue(expected tac.Operand, reg string) error {
	if err := g.validateClasses(); err != nil {
		return err
	}
	if expected.Type == model.TypeError {
		return nil
	}
	desc, err := g.typeDescriptor(expected)
	if err != nil {
		return err
	}
	g.emit("    move $a0, %s", reg)
	g.emit("    la $a1, %s", desc)
	g.emit("    jal ref_check")
	g.emit("    move %s, $v0", reg)
	return nil
}

func (g *generator) fault(message string) {
	g.emit("    la $a0, %s", g.text(message))
	g.emit("    j rt_fault")
}

func (g *generator) requireReference(reg string, tag int, nullMessage, invalid string) {
	nonnull, ok := g.unique("nonnull"), g.unique("reference")
	g.emit("    lw $t0, 0(%s)", reg)
	g.emit("    bne $t0, 8, %s", nonnull)
	g.fault(nullMessage)
	g.emit("%s:", nonnull)
	g.emit("    beq $t0, %d, %s", tag, ok)
	g.fault(invalid)
	g.emit("%s:", ok)
}

func (g *generator) referenceInstruction(i tac.Instruction) error {
	if err := g.validateClasses(); err != nil {
		return err
	}
	switch i.Op {
	case tac.CheckValue:
		if err := g.load(i.Left, "$s3"); err != nil {
			return err
		}
		if err := g.checkValue(i.Target, "$s3"); err != nil {
			return err
		}
		return g.store(i.Target, "$s3")
	case tac.NewObject:
		return g.newObject(i)
	case tac.NewList:
		return g.newList(i)
	case tac.ListLength, tac.LoadIndex, tac.StoreIndex:
		return g.listInstruction(i)
	case tac.LoadField, tac.StoreField, tac.BindMethod, tac.DynamicMember, tac.DynamicStore:
		return g.objectInstruction(i)
	}
	return fmt.Errorf("invalid reference opcode %s", i.Op)
}

func (g *generator) newObject(i tac.Instruction) error {
	c := g.class(i.Class)
	if c == nil {
		return fmt.Errorf("undefined object class")
	}
	g.emit("    li $a0, %d", 16+4*len(c.Fields)+4*len(g.program.Classes))
	g.emit("    jal rt_alloc")
	g.emit("    move $s2, $v0")
	g.emit("    li $t0, 5")
	g.emit("    sw $t0, 0($s2)")
	g.emit("    la $t0, %s", g.classLabel(c.ID))
	g.emit("    sw $t0, 4($s2)")
	g.emit("    addiu $t0, $s2, %d", 16+4*len(c.Fields))
	g.emit("    sw $t0, 8($s2)")
	g.emit("    sw $zero, 12($s2)")
	for n := 0; n < len(c.Fields)+len(g.program.Classes); n++ {
		g.emit("    sw $zero, %d($s2)", 16+n*4)
	}
	// Capture each base's environment now, not when a method is later selected.
	for cur := c; cur != nil; cur = g.class(cur.Parent) {
		g.environment(cur.LexicalScope, "$s7")
		g.emit("    beq $s7, $zero, rt_undefined")
		g.emit("    lw $t0, 8($s2)")
		g.emit("    sw $s7, %d($t0)", g.classNumber(cur.ID)*4)
	}
	return g.store(i.Target, "$s2")
}

func (g *generator) newList(i tac.Instruction) error {
	if i.Target.Type != model.TypeList || len(i.Arguments) > 8180 {
		return fmt.Errorf("invalid list allocation")
	}
	desc, err := g.typeDescriptor(element(i.Target))
	if err != nil {
		return err
	}
	g.emit("    li $a0, %d", 16+len(i.Arguments)*4)
	g.emit("    jal rt_alloc")
	g.emit("    move $s2, $v0")
	g.emit("    li $t0, 6")
	g.emit("    sw $t0, 0($s2)")
	g.emit("    la $t0, %s", desc)
	g.emit("    sw $t0, 4($s2)")
	g.emit("    li $t0, %d", len(i.Arguments))
	g.emit("    sw $t0, 8($s2)")
	g.emit("    sw $zero, 12($s2)")
	for n, a := range i.Arguments {
		if err := g.load(a, "$s3"); err != nil {
			return err
		}
		if err := g.checkValue(element(i.Target), "$s3"); err != nil {
			return err
		}
		g.emit("    sw $s3, %d($s2)", 16+n*4)
	}
	return g.store(i.Target, "$s2")
}

func (g *generator) listInstruction(i tac.Instruction) error {
	if i.Op != tac.ListLength && i.Index == nil {
		return fmt.Errorf("missing list index")
	}
	if i.Op == tac.StoreIndex && i.Right == nil {
		return fmt.Errorf("missing list value")
	}
	if err := g.load(i.Left, "$s2"); err != nil {
		return err
	}
	g.requireReference("$s2", tagList, "null list", "invalid list reference")
	if i.Op == tac.ListLength {
		g.emit("    lw $t0, 8($s2)")
		g.emit("    mtc1 $t0, $f4")
		g.emit("    cvt.d.w $f4, $f4")
		g.emit("    li $a0, 16")
		g.emit("    jal rt_alloc")
		g.emit("    li $t0, 1")
		g.emit("    sw $t0, 0($v0)")
		g.emit("    s.d $f4, 8($v0)")
		return g.store(i.Target, "$v0")
	}
	if err := g.load(*i.Index, "$s3"); err != nil {
		return err
	}
	g.emit("    move $a0, $s2")
	g.emit("    move $a1, $s3")
	g.emit("    jal ref_index")
	g.emit("    move $s4, $v0")
	if i.Op == tac.LoadIndex {
		g.emit("    lw $s3, 0($s4)")
		if err := g.checkValue(i.Target, "$s3"); err != nil {
			return err
		}
		return g.store(i.Target, "$s3")
	}
	if err := g.load(*i.Right, "$s3"); err != nil {
		return err
	}
	if err := g.checkValue(element(i.Left), "$s3"); err != nil {
		return err
	}
	// A generic access cannot erase the original allocation's element contract.
	// Known covariant views deliberately retain the existing frontend policy.
	if i.Left.Type == model.TypeError || i.Left.Shape == "" ||
		strings.Contains(i.Left.Shape, string(model.TypeError)) {
		g.emit("    move $a0, $s3")
		g.emit("    lw $a1, 4($s2)")
		g.emit("    jal ref_check")
		g.emit("    move $s3, $v0")
	}
	g.emit("    sw $s3, 0($s4)")
	return nil
}

func fieldOperand(f tac.FieldLayout) tac.Operand {
	return tac.Operand{Type: f.Type.Kind, Shape: f.Shape, Class: f.Class}
}

func (g *generator) fieldAccess(i tac.Instruction, f tac.FieldLayout, dynamic bool) error {
	offset := 16 + 4*f.Offset
	if i.Op == tac.LoadField || i.Op == tac.DynamicMember {
		g.emit("    lw $s3, %d($s2)", offset)
		ok := g.unique("field_defined")
		g.emit("    bne $s3, $zero, %s", ok)
		g.fault("undefined field " + f.ID)
		g.emit("%s:", ok)
		return g.store(i.Target, "$s3")
	}
	if i.Right == nil {
		return fmt.Errorf("missing field value")
	}
	if !f.Mutable {
		if dynamic || g.procedure == nil || g.procedure.Label != g.class(f.Owner).Initializer {
			g.fault("immutable field")
		} else {
			ok := g.unique("field_unset")
			g.emit("    lw $t0, %d($s2)", offset)
			g.emit("    beq $t0, $zero, %s", ok)
			g.fault("immutable field already initialized")
			g.emit("%s:", ok)
		}
	}
	if err := g.load(*i.Right, "$s3"); err != nil {
		return err
	}
	if err := g.checkValue(fieldOperand(f), "$s3"); err != nil {
		return err
	}
	g.emit("    sw $s3, %d($s2)", offset)
	if dynamic {
		return g.store(i.Target, "$s3")
	}
	return nil
}

func (g *generator) bind(i tac.Instruction, c *tac.ClassLayout, label string) error {
	p := g.findProcedure(label)
	if p == nil || p.Receiver == nil {
		return fmt.Errorf("undefined object method")
	}
	owner := c
	for owner != nil && owner.LexicalScope != p.LexicalScope {
		owner = g.class(owner.Parent)
	}
	if owner == nil {
		return fmt.Errorf("undefined method environment")
	}
	g.emit("    lw $t0, 8($s2)")
	g.emit("    lw $s7, %d($t0)", g.classNumber(owner.ID)*4)
	g.emit("    li $a0, 16")
	g.emit("    jal rt_alloc")
	g.emit("    li $t0, 7")
	g.emit("    sw $t0, 0($v0)")
	g.emit("    la $t0, %s", g.procedures[label])
	g.emit("    sw $t0, 4($v0)")
	g.emit("    sw $s7, 8($v0)")
	g.emit("    sw $s2, 12($v0)")
	return g.store(i.Target, "$v0")
}

func (g *generator) objectInstruction(i tac.Instruction) error {
	dynamic := i.Op == tac.DynamicMember || i.Op == tac.DynamicStore
	c := g.class(i.Class)
	if !dynamic && c == nil {
		return fmt.Errorf("undefined object class")
	}
	if !dynamic && (i.Op == tac.LoadField || i.Op == tac.StoreField) {
		if i.Offset == nil || *i.Offset < 0 || *i.Offset >= len(c.Fields) {
			return fmt.Errorf("field offset out of bounds")
		}
		if c.Fields[*i.Offset].ID != i.Member {
			return fmt.Errorf("invalid field identity")
		}
	}
	if i.Op == tac.BindMethod {
		if i.Callee != "" && i.Callee != c.Initializer {
			return fmt.Errorf("invalid initializer selection")
		}
		if i.Callee == "" {
			found := false
			for _, method := range c.Methods {
				found = found || method.Name == i.Member
			}
			if !found {
				return fmt.Errorf("undefined object method %q", i.Member)
			}
		}
	}
	if err := g.load(i.Left, "$s2"); err != nil {
		return err
	}
	g.requireReference("$s2", tagObject, "null receiver", "invalid object receiver")
	if !dynamic {
		g.emit("    lw $a0, 4($s2)")
		g.emit("    la $a1, %s", g.classLabel(c.ID))
		g.emit("    jal ref_subclass")
		g.emit("    beq $v0, $zero, ref_object_error")
	}
	if i.Op == tac.LoadField || i.Op == tac.StoreField {
		return g.fieldAccess(i, c.Fields[*i.Offset], false)
	}
	if i.Op == tac.BindMethod && i.Callee != "" {
		return g.bind(i, c, i.Callee)
	}
	done := g.unique("member_done")
	g.emit("    lw $t2, 4($s2)")
	for n := range g.program.Classes {
		actual := &g.program.Classes[n]
		if !dynamic && !g.subclass(actual.ID, c.ID) {
			continue
		}
		next := g.unique("next_class")
		g.emit("    la $t0, %s", g.classLabel(actual.ID))
		g.emit("    bne $t2, $t0, %s", next)
		found := false
		if dynamic {
			for _, f := range actual.Fields {
				if f.Name == i.Member {
					if err := g.fieldAccess(i, f, true); err != nil {
						return err
					}
					found = true
					break
				}
			}
		}
		if !found && i.Op != tac.DynamicStore {
			for _, m := range actual.Methods {
				if m.Name == i.Member {
					if err := g.bind(i, actual, m.Callee); err != nil {
						return err
					}
					found = true
					break
				}
			}
		}
		if !found {
			g.fault("missing object member " + i.Member)
		}
		g.emit("    j %s", done)
		g.emit("%s:", next)
	}
	g.fault("missing object member " + i.Member)
	g.emit("%s:", done)
	return nil
}

func (g *generator) referenceSupport() string {
	for _, c := range g.program.Classes {
		parent := "0"
		if c.Parent != "" {
			parent = g.classLabel(c.Parent)
		}
		name := g.text("<object " + c.ID + ">")
		fmt.Fprintf(&g.data, ".align 2\n%s: .word %s, %s\n", g.classLabel(c.ID), parent, name)
	}
	var out strings.Builder
	out.WriteString(referenceRuntime)
	out.WriteString("\nref_function_text:\n    lw $t0, 4($a0)\n")
	for _, p := range g.program.Procedures {
		name := g.text("<function " + p.Label + ">")
		code := g.procedures[p.Label]
		fmt.Fprintf(&out, "    la $t1, %s\n    bne $t0, $t1, ref_print_next_%s\n", code, code)
		fmt.Fprintf(&out, "    la $a0, %s\n    j rt_print_text\nref_print_next_%s:\n", name, code)
	}
	out.WriteString("    j rt_type_error\n")
	return out.String()
}

const referenceRuntime = `
# a0 class descriptor, a1 expected class. Leaf, preserves saved registers.
ref_subclass:
    beq $a0, $a1, ref_yes
    beq $a0, $zero, ref_no
    lw $a0, 0($a0)
    j ref_subclass
ref_yes:
    li $v0, 1
    jr $ra
ref_no:
    move $v0, $zero
    jr $ra

# Descriptor covariance. No memory writes; class leaves walk nominal ancestry.
ref_fits:
    lw $t0, 0($a0)
    lw $t1, 0($a1)
    beq $t1, $zero, ref_yes
    beq $t0, $t1, ref_fits_same
    bne $t0, 1, ref_no
    bne $t1, 2, ref_no
    j ref_yes
ref_fits_same:
    beq $t0, 5, ref_fits_class
    bne $t0, 6, ref_yes
    lw $a0, 4($a0)
    lw $a1, 4($a1)
    j ref_fits
ref_fits_class:
    lw $a0, 4($a0)
    lw $a1, 4($a1)
    j ref_subclass
ref_generic:
    lw $t0, 0($a0)
    beq $t0, $zero, ref_yes
    bne $t0, 6, ref_no
    lw $a0, 4($a0)
    j ref_generic

# Check a0 boxed value against a1 descriptor. Result v0 is a possibly new
# float box. Recursive list checks never copy/mutate the allocation or elements.
ref_check:
    addiu $sp, $sp, -32
    sw $ra, 0($sp)
    sw $a0, 4($sp)
    sw $a1, 8($sp)
    lw $t0, 0($a1)
    beq $t0, $zero, ref_check_done
    lw $t1, 0($a0)
    beq $t0, $t1, ref_check_same
    beq $t1, 8, ref_check_null
    bne $t0, 2, rt_type_error
    bne $t1, 1, rt_type_error
    li $a0, 16
    jal rt_alloc
    li $t0, 2
    sw $t0, 0($v0)
    lw $t0, 4($sp)
    l.d $f0, 8($t0)
    s.d $f0, 8($v0)
    sw $v0, 4($sp)
    j ref_check_done
ref_check_null:
    beq $t0, 4, ref_check_done
    beq $t0, 5, ref_check_done
    beq $t0, 6, ref_check_done
    j rt_type_error
ref_check_same:
    beq $t0, 5, ref_check_class
    beq $t0, 6, ref_check_list
    j ref_check_done
ref_check_class:
    lw $a0, 4($a0)
    lw $a1, 4($a1)
    jal ref_subclass
    beq $v0, $zero, ref_nominal_error
    j ref_check_done
ref_check_list:
    lw $a0, 4($a1)
    jal ref_generic
    bne $v0, $zero, ref_check_done
    lw $t0, 4($sp)
    lw $a0, 4($t0)
    jal ref_generic
    bne $v0, $zero, ref_check_contents
    lw $t0, 4($sp)
    lw $a0, 4($t0)
    lw $t0, 8($sp)
    lw $a1, 4($t0)
    jal ref_fits
    beq $v0, $zero, ref_list_type_error
    j ref_check_done
ref_check_contents:
    sw $zero, 12($sp)
ref_check_loop:
    lw $t0, 4($sp)
    lw $t1, 12($sp)
    lw $t2, 8($t0)
    beq $t1, $t2, ref_check_done
    sll $t2, $t1, 2
    addu $t0, $t0, $t2
    lw $a0, 16($t0)
    lw $t0, 8($sp)
    lw $a1, 4($t0)
    jal ref_check
    lw $t0, 12($sp)
    addiu $t0, $t0, 1
    sw $t0, 12($sp)
    j ref_check_loop
ref_check_done:
    lw $v0, 4($sp)
    lw $ra, 0($sp)
    addiu $sp, $sp, 32
    jr $ra

# Validate the entire IEEE64 integer before narrowing. Modulo one checks
# integrality without a 32-bit conversion, including huge fractional values.
ref_index:
    lw $t0, 0($a1)
    bne $t0, 1, ref_index_error
    l.d $f0, 8($a1)
    mfc1 $t0, $f1
    li $t1, 0x7ff00000
    and $t0, $t0, $t1
    beq $t0, $t1, ref_index_error
    addiu $sp, $sp, -8
    sw $ra, 0($sp)
    li $t0, 1
    mtc1 $t0, $f2
    cvt.d.w $f2, $f2
    jal rt_fmod
    lw $ra, 0($sp)
    addiu $sp, $sp, 8
    mtc1 $zero, $f2
    cvt.d.w $f2, $f2
    c.eq.d $f4, $f2
    bc1f ref_index_error
    c.le.d $f2, $f0
    bc1f ref_bounds_error
    lw $t0, 8($a0)
    mtc1 $t0, $f2
    cvt.d.w $f2, $f2
    c.lt.d $f0, $f2
    bc1f ref_bounds_error
    trunc.w.d $f2, $f0
    cvt.d.w $f4, $f2
    c.eq.d $f0, $f4
    bc1f ref_index_error
    mfc1 $t0, $f2
    sll $t0, $t0, 2
    addu $v0, $a0, $t0
    addiu $v0, $v0, 16
    jr $ra

# Printing uses a shared per-print budget and a stack-linked ancestor chain.
# Only list ancestors participate in cycle detection; siblings remain printable.
rt_print_reference:
    li $t0, 64
    sw $t0, ref_budget
    move $a1, $zero
    move $a2, $zero
    j ref_render
ref_render:
    addiu $sp, $sp, -32
    sw $ra, 0($sp)
    sw $a0, 4($sp)
    sw $a1, 8($sp)
    sw $a2, 12($sp)
    lw $t0, ref_budget
    blez $t0, ref_render_ellipsis
    bge $a1, 8, ref_render_ellipsis
    addiu $t0, $t0, -1
    sw $t0, ref_budget
    lw $t0, 0($a0)
    beq $t0, 6, ref_render_list
    beq $t0, 5, ref_render_object
    beq $t0, 7, ref_render_function
    jal rt_print
    j ref_render_done
ref_render_object:
    lw $t0, 4($a0)
    lw $a0, 4($t0)
    jal rt_print_text
    j ref_render_done
ref_render_function:
    jal ref_function_text
    j ref_render_done
ref_render_ellipsis:
    la $a0, ref_ellipsis
    jal rt_print_text
    j ref_render_done
ref_render_list:
    move $t0, $a2
ref_render_ancestors:
    beq $t0, $zero, ref_render_open
    lw $t1, 4($t0)
    beq $t1, $a0, ref_render_cycle
    lw $t0, 12($t0)
    j ref_render_ancestors
ref_render_cycle:
    la $a0, ref_cycle
    jal rt_print_text
    j ref_render_done
ref_render_open:
    li $v0, 11
    li $a0, 91
    syscall
    sw $zero, 16($sp)
ref_render_elements:
    lw $t0, 4($sp)
    lw $t1, 16($sp)
    lw $t2, 8($t0)
    beq $t1, $t2, ref_render_close
    beq $t1, $zero, ref_render_element
    la $a0, ref_separator
    jal rt_print_text
ref_render_element:
    lw $t0, 4($sp)
    lw $t1, 16($sp)
    sll $t1, $t1, 2
    addu $t0, $t0, $t1
    lw $a0, 16($t0)
    lw $a1, 8($sp)
    addiu $a1, $a1, 1
    move $a2, $sp
    jal ref_render
    lw $t0, ref_budget
    blez $t0, ref_render_close
    lw $t0, 16($sp)
    addiu $t0, $t0, 1
    sw $t0, 16($sp)
    j ref_render_elements
ref_render_close:
    li $v0, 11
    li $a0, 93
    syscall
ref_render_done:
    lw $ra, 0($sp)
    addiu $sp, $sp, 32
    jr $ra
ref_nominal_error:
    la $a0, ref_msg_nominal
    j rt_fault
ref_list_type_error:
    la $a0, ref_msg_list_type
    j rt_fault
ref_object_error:
    la $a0, ref_msg_object
    j rt_fault
ref_index_error:
    la $a0, ref_msg_index
    j rt_fault
ref_bounds_error:
    la $a0, ref_msg_bounds
    j rt_fault
ref_binary_error:
    la $a0, ref_msg_binary
    j rt_fault
.data
.align 2
ref_budget: .word 0
ref_ellipsis: .asciiz "..."
ref_cycle: .asciiz "<cycle>"
ref_separator: .asciiz ", "
ref_msg_nominal: .asciiz "nominal object type mismatch"
ref_msg_list_type: .asciiz "list type mismatch"
ref_msg_object: .asciiz "invalid object receiver"
ref_msg_index: .asciiz "invalid list index"
ref_msg_bounds: .asciiz "list index out of bounds"
ref_msg_binary: .asciiz "invalid binary operand types"
.text
`

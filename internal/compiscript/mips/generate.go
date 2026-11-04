// Package mips lowers structured TAC into executable MARS MIPS32 assembly.
package mips

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"genanalex/internal/compiscript/model"
	"genanalex/internal/compiscript/tac"
)

// Tags are physical runtime values, independent of the types on reused TAC cells.
const (
	tagInteger   = 1
	tagFloat     = 2
	tagBoolean   = 3
	tagString    = 4
	tagObject    = 5
	tagList      = 6
	tagFunction  = 7
	tagNull      = 8
	tagException = 9
)
const frameHeader = 48

// Error locates invalid or unrepresentable TAC without returning partial assembly.
type Error struct {
	Procedure   string
	Instruction int
	Message     string
}

func (e *Error) Error() string {
	return fmt.Sprintf("MIPS %s instruction %d: %s", e.Procedure, e.Instruction, e.Message)
}

type generator struct {
	program    *tac.Program
	data, code strings.Builder
	literals   map[string]string
	globals    map[string]string
	procedures map[string]string
	serial     int
	current    string
	procedure  *tac.Procedure
	homes      map[string]int
	labels     map[string]string
	frameSize  int
	registers  registerPool
}

type registerPool struct{ used int }

// getReg leases scratch registers for one instruction. Every TAC value has a
// canonical memory home: writes spill immediately, so calls/edges lose no values.
func (p *registerPool) getReg() string {
	if p.used >= 8 {
		panic("MIPS scratch register budget exceeded")
	}
	r := fmt.Sprintf("$t%d", p.used)
	p.used++
	return r
}

// Generate validates and translates the complete program or returns no assembly.
func Generate(program *tac.Program) (string, error) {
	if program == nil {
		return "", fmt.Errorf("MIPS: nil TAC program")
	}
	g := &generator{program: program, literals: map[string]string{}, globals: map[string]string{}, procedures: map[string]string{}}
	for _, p := range program.Procedures {
		if _, ok := g.procedures[p.Label]; ok || p.Label == "" {
			return "", fmt.Errorf("MIPS: duplicate/empty procedure")
		}
		g.procedures[p.Label] = g.unique("procedure")
	}
	g.data.WriteString(".data\n.align 3\nrt_null: .word 8, 0, 0, 0\nrt_false: .word 3, 0, 0, 0\nrt_true: .word 3, 1, 0, 0\nrt_newline: .asciiz \"\\n\"\nrt_true_text: .asciiz \"true\"\nrt_false_text: .asciiz \"false\"\nrt_null_text: .asciiz \"null\"\n")
	// Main symbols and explicit procedure globals share canonical static cells.
	symbols := map[string]bool{}
	scan := func(tape []tac.Instruction) {
		for _, i := range tape {
			for _, o := range operands(i) {
				if o.Kind == tac.Symbol {
					symbols[o.Value] = true
				}
			}
		}
	}
	scan(program.Instructions)
	for _, p := range program.Procedures {
		for _, o := range p.Globals {
			symbols[o.Value] = true
		}
	}
	names := make([]string, 0, len(symbols))
	for n := range symbols {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		label := g.unique("global")
		g.globals[n] = label
		fmt.Fprintf(&g.data, ".align 2\n%s: .word 0\n", label)
	}
	g.code.WriteString(".text\n.globl main\n")
	if err := g.emitProcedure("main", nil, program.Instructions); err != nil {
		return "", err
	}
	for n := range program.Procedures {
		p := &program.Procedures[n]
		if err := g.emitProcedure(g.procedures[p.Label], p, p.Instructions); err != nil {
			return "", err
		}
	}
	g.code.WriteString(scalarRuntime)
	return g.data.String() + g.code.String(), nil
}
func operands(i tac.Instruction) []tac.Operand {
	out := []tac.Operand{i.Target, i.Left}
	if i.Right != nil {
		out = append(out, *i.Right)
	}
	out = append(out, i.Arguments...)
	if i.ReturnValue != nil {
		out = append(out, *i.ReturnValue)
	}
	if i.Index != nil {
		out = append(out, *i.Index)
	}
	if i.Receiver != nil {
		out = append(out, *i.Receiver)
	}
	return out
}
func key(o tac.Operand) string { return string(o.Kind) + ":" + o.Value }
func (g *generator) unique(prefix string) string {
	g.serial++
	return fmt.Sprintf("m_%s_%d", prefix, g.serial)
}
func (g *generator) emit(format string, args ...any) { fmt.Fprintf(&g.code, format+"\n", args...) }
func (g *generator) text(value string) string {
	label := g.unique("text")
	fmt.Fprintf(&g.data, "%s: .byte ", label)
	for _, b := range []byte(value) {
		fmt.Fprintf(&g.data, "%d,", int8(b))
	}
	g.data.WriteString("0\n")
	return label
}
func (g *generator) literal(o tac.Operand) (string, error) {
	k := string(o.Type) + ":" + o.Value
	if label, ok := g.literals[k]; ok {
		return label, nil
	}
	if o.Type == model.TypeNull {
		return "rt_null", nil
	}
	if o.Type == model.TypeBoolean {
		if o.Value == "true" {
			return "rt_true", nil
		}
		if o.Value == "false" {
			return "rt_false", nil
		}
		return "", fmt.Errorf("invalid boolean literal")
	}
	label := g.unique("literal")
	switch o.Type {
	case model.TypeInteger, model.TypeFloat:
		number, err := strconv.ParseFloat(o.Value, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return "", fmt.Errorf("invalid/non-finite numeric literal %q", o.Value)
		}
		tag := tagInteger
		if o.Type == model.TypeFloat {
			tag = tagFloat
		}
		fmt.Fprintf(&g.data, ".align 3\n%s: .word %d, 0\n.double %s\n", label, tag, strconv.FormatFloat(number, 'e', 17, 64))
	case model.TypeString:
		value, err := strconv.Unquote(o.Value)
		if err != nil {
			return "", err
		}
		text := g.text(value)
		fmt.Fprintf(&g.data, ".align 3\n%s: .word 4, %s, %d, 0\n", label, text, len([]byte(value)))
	default:
		return "", fmt.Errorf("invalid literal type %q", o.Type)
	}
	g.literals[k] = label
	return label, nil
}
func (g *generator) address(o tac.Operand, reg string) error {
	if offset, ok := g.homes[key(o)]; ok {
		g.emit("    addiu %s, $fp, %d", reg, offset)
		return nil
	}
	if o.Kind == tac.Symbol {
		if label, ok := g.globals[o.Value]; ok {
			g.emit("    la %s, %s", reg, label)
			return nil
		}
	}
	return fmt.Errorf("operand has no memory home: %s", o)
}
func (g *generator) load(o tac.Operand, reg string) error {
	if o.Kind == tac.Literal {
		label, err := g.literal(o)
		if err != nil {
			return err
		}
		g.emit("    la %s, %s", reg, label)
		return nil
	}
	if o.Kind != tac.Symbol && o.Kind != tac.Temporary {
		return fmt.Errorf("invalid source operand")
	}
	if err := g.address(o, reg); err != nil {
		return err
	}
	g.emit("    lw %s, 0(%s)", reg, reg)
	g.emit("    beq %s, $zero, rt_undefined", reg)
	return nil
}
func (g *generator) store(o tac.Operand, reg string) error {
	if o.Kind != tac.Symbol && o.Kind != tac.Temporary {
		return fmt.Errorf("invalid destination")
	}
	// Address lookup may call a runtime helper in the environment extension;
	// values survive in callee-saved scratch registers, never in $v0.
	g.emit("    move $s6, %s", reg)
	if err := g.address(o, "$v1"); err != nil {
		return err
	}
	g.emit("    sw $s6, 0($v1)")
	return nil
}
func (g *generator) emitProcedure(label string, p *tac.Procedure, tape []tac.Instruction) error {
	g.current = label
	g.procedure = p
	g.homes = map[string]int{}
	g.labels = map[string]string{}
	add := func(o tac.Operand) {
		if o.Kind != "" {
			if _, ok := g.homes[key(o)]; !ok {
				g.homes[key(o)] = frameHeader + len(g.homes)*4
			}
		}
	}
	if p != nil {
		for _, slot := range p.Frame.Slots {
			if slot.Operand != nil {
				add(*slot.Operand)
			}
		}
	}
	for _, i := range tape {
		if i.Op == tac.Label {
			if _, ok := g.labels[i.Label]; ok || i.Label == "" {
				return fmt.Errorf("MIPS: duplicate/empty label")
			}
			g.labels[i.Label] = g.unique("label")
		}
		for _, o := range operands(i) {
			if o.Kind == tac.Temporary {
				add(o)
			}
		}
	}
	g.frameSize = (frameHeader + len(g.homes)*4 + 7) &^ 7
	if g.frameSize > 32760 {
		return fmt.Errorf("MIPS: frame exceeds immediate offset range")
	}
	g.emit("%s:", label)
	g.emit("    addiu $sp, $sp, -%d", g.frameSize)
	g.emit("    sw $ra, 0($sp)")
	g.emit("    sw $fp, 4($sp)")
	for n := 0; n < 8; n++ {
		g.emit("    sw $s%d, %d($sp)", n, 8+n*4)
	}
	g.emit("    move $fp, $sp")
	g.emit("    move $s0, $a1")
	g.emit("    move $s1, $zero")
	for _, offset := range sortedOffsets(g.homes) {
		g.emit("    sw $zero, %d($fp)", offset)
	}
	if p != nil {
		for n, formal := range p.Parameters {
			g.emit("    lw $t0, %d($a0)", n*4)
			g.emit("    sw $t0, %d($fp)", g.homes[key(formal)])
		}
		if p.Receiver != nil {
			g.emit("    sw $a2, %d($fp)", g.homes[key(*p.Receiver)])
		}
	}
	for pc, i := range tape {
		g.registers.used = 0
		g.emit("    # TAC %d: %s", pc, i.Op)
		if err := g.instruction(i); err != nil {
			return &Error{label, pc, err.Error()}
		}
	}
	if p == nil {
		g.emit("    li $v0, 10")
		g.emit("    syscall")
	} else {
		g.emit("    j rt_type_error") // A typed procedure may not fall through.
		g.emit("%s_return:", label)
		g.emit("    move $sp, $fp")
		g.emit("    lw $ra, 0($sp)")
		g.emit("    lw $t9, 4($sp)")
		for n := 0; n < 8; n++ {
			g.emit("    lw $s%d, %d($sp)", n, 8+n*4)
		}
		g.emit("    addiu $sp, $sp, %d", g.frameSize)
		g.emit("    move $fp, $t9")
		g.emit("    jr $ra")
	}
	return nil
}
func sortedOffsets(homes map[string]int) []int {
	out := make([]int, 0, len(homes))
	for _, n := range homes {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}
func (g *generator) jumpLabel(label string) (string, error) {
	target, ok := g.labels[label]
	if !ok {
		return "", fmt.Errorf("undefined label %q", label)
	}
	return target, nil
}
func (g *generator) instruction(i tac.Instruction) error {
	switch i.Op {
	case tac.Label:
		label, err := g.jumpLabel(i.Label)
		if err != nil {
			return err
		}
		g.emit("%s:", label)
		return nil
	case tac.Jump:
		label, err := g.jumpLabel(i.Label)
		if err != nil {
			return err
		}
		g.emit("    j %s", label)
		return nil
	case tac.Declare:
		return g.store(i.Target, "$zero")
	case tac.BranchTrue:
		label, err := g.jumpLabel(i.Label)
		if err != nil {
			return err
		}
		if err = g.load(i.Left, "$s3"); err != nil {
			return err
		}
		r := g.registers.getReg()
		g.emit("    lw %s, 0($s3)", r)
		g.emit("    bne %s, 3, rt_boolean_error", r)
		g.emit("    lw %s, 4($s3)", r)
		g.emit("    bne %s, $zero, %s", r, label)
		return nil
	case tac.Print:
		if err := g.load(i.Left, "$a0"); err != nil {
			return err
		}
		g.emit("    jal rt_print")
		g.emit("    li $v0, 4")
		g.emit("    la $a0, rt_newline")
		g.emit("    syscall")
		return nil
	case tac.Call:
		return g.directCall(i)
	case tac.Return:
		if g.procedure == nil {
			return fmt.Errorf("return outside procedure")
		}
		if i.ReturnValue != nil {
			if err := g.load(*i.ReturnValue, "$v0"); err != nil {
				return err
			}
		} else {
			g.emit("    la $v0, rt_null")
		}
		g.emit("    j %s_return", g.current)
		return nil
	case tac.Copy:
		if err := g.load(i.Left, "$s3"); err != nil {
			return err
		}
		return g.store(i.Target, "$s3")
	case tac.Add, tac.Subtract, tac.Multiply, tac.Divide, tac.Remainder, tac.Negate, tac.IntegerToFloat, tac.LogicalNot, tac.Equal, tac.NotEqual, tac.Less, tac.LessEqual, tac.Greater, tac.GreaterEqual:
		return g.scalar(i)
	default:
		return fmt.Errorf("unsupported opcode %q", i.Op)
	}
}
func (g *generator) scalar(i tac.Instruction) error {
	unary := i.Op == tac.Negate || i.Op == tac.IntegerToFloat || i.Op == tac.LogicalNot
	if unary && i.Right != nil || !unary && i.Right == nil {
		return fmt.Errorf("invalid unary/binary operands")
	}
	if err := g.load(i.Left, "$s3"); err != nil {
		return err
	}
	if i.Right != nil {
		if err := g.load(*i.Right, "$s4"); err != nil {
			return err
		}
	}
	if i.Op == tac.Equal || i.Op == tac.NotEqual {
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
		g.emit("    bne $t0, 3, rt_boolean_error")
		g.emit("    lw $t0, 4($s3)")
		g.emit("    xori $t0, $t0, 1")
		g.boolResult("$t0")
		return g.store(i.Target, "$s5")
	}
	if i.Op == tac.Add && i.Target.Type == model.TypeString {
		g.emit("    move $a0, $s3")
		g.emit("    move $a1, $s4")
		g.emit("    jal rt_concat")
		return g.store(i.Target, "$v0")
	}
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

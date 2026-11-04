package mips

// Runtime leaf helpers preserve $s0-$s7 and $fp. They may clobber $a/$t/$v/FPU.
// Values are aligned immutable 16-byte boxes; references carry heap identities.
const scalarRuntime = `
# Aligned monotonic allocation. Heap ownership outlives activation records.
rt_alloc:
    addiu $a0, $a0, 7
    li $v0, 9
    syscall
    addiu $v0, $v0, 7
    li $t8, -8
    and $v0, $v0, $t8
    jr $ra

# Equality returns an unboxed boolean; numeric tags compare their double values.
rt_equal:
    beq $a0, $a1, rt_equal_yes
    lw $t0, 0($a0)
    lw $t1, 0($a1)
    beq $t0, 8, rt_equal_no
    beq $t1, 8, rt_equal_no
    bgt $t0, 2, rt_equal_other
    bgt $t1, 2, rt_equal_no
    l.d $f0, 8($a0)
    l.d $f2, 8($a1)
    c.eq.d $f0, $f2
    bc1t rt_equal_yes
    j rt_equal_no
rt_equal_other:
    bne $t0, $t1, rt_equal_no
    beq $t0, 3, rt_equal_bool
    beq $t0, 7, rt_equal_function
    bne $t0, 4, rt_equal_no
    lw $t4, 8($a0)
    lw $t5, 8($a1)
    bne $t4, $t5, rt_equal_no
    lw $t0, 4($a0)
    lw $t1, 4($a1)
rt_equal_chars:
    beq $t4, $zero, rt_equal_yes
    lbu $t2, 0($t0)
    lbu $t3, 0($t1)
    bne $t2, $t3, rt_equal_no
    addiu $t0, $t0, 1
    addiu $t1, $t1, 1
    addiu $t4, $t4, -1
    j rt_equal_chars
rt_equal_function:
    lw $t0, 4($a0)
    lw $t1, 4($a1)
    bne $t0, $t1, rt_equal_no
    lw $t0, 8($a0)
    lw $t1, 8($a1)
    bne $t0, $t1, rt_equal_no
    lw $t0, 12($a0)
    lw $t1, 12($a1)
    bne $t0, $t1, rt_equal_no
    j rt_equal_yes
rt_equal_bool:
    lw $t0, 4($a0)
    lw $t1, 4($a1)
    bne $t0, $t1, rt_equal_no
rt_equal_yes:
    li $v0, 1
    jr $ra
rt_equal_no:
    move $v0, $zero
    jr $ra

rt_concat:
    addiu $sp, $sp, -24
    sw $ra, 0($sp)
    sw $a0, 4($sp)
    sw $a1, 8($sp)
    lw $t0, 0($a0)
    bne $t0, 4, rt_string_error
    lw $t0, 0($a1)
    bne $t0, 4, rt_string_error
    lw $t0, 8($a0)
    lw $t1, 8($a1)
    addu $t2, $t0, $t1
    sw $t2, 16($sp)
    addiu $a0, $t2, 1
    jal rt_alloc
    sw $v0, 12($sp)
    move $t2, $v0
    lw $t0, 4($sp)
    lw $t4, 8($t0)
    lw $t0, 4($t0)
rt_concat_copy_a:
    beq $t4, $zero, rt_concat_start_b
    lbu $t3, 0($t0)
    sb $t3, 0($t2)
    addiu $t0, $t0, 1
    addiu $t2, $t2, 1
    addiu $t4, $t4, -1
    j rt_concat_copy_a
rt_concat_start_b:
    lw $t0, 8($sp)
    lw $t4, 8($t0)
    lw $t0, 4($t0)
rt_concat_copy_b:
    beq $t4, $zero, rt_concat_box
    lbu $t3, 0($t0)
    sb $t3, 0($t2)
    addiu $t0, $t0, 1
    addiu $t2, $t2, 1
    addiu $t4, $t4, -1
    j rt_concat_copy_b
rt_concat_box:
    sb $zero, 0($t2)
    li $a0, 16
    jal rt_alloc
    li $t0, 4
    sw $t0, 0($v0)
    lw $t0, 12($sp)
    sw $t0, 4($v0)
    lw $t0, 16($sp)
    sw $t0, 8($v0)
    lw $ra, 0($sp)
    addiu $sp, $sp, 24
    jr $ra

rt_print:
    lw $t0, 0($a0)
    ble $t0, 2, rt_print_number
    beq $t0, 3, rt_print_boolean
    beq $t0, 4, rt_print_string
    beq $t0, 9, rt_print_string
    beq $t0, 8, rt_print_null
    j rt_type_error
rt_print_string:
    lw $a1, 4($a0)
    lw $a2, 8($a0)
    li $a0, 1
    li $v0, 15
    syscall
    jr $ra
rt_print_boolean:
    lw $t0, 4($a0)
    la $a0, rt_false_text
    beq $t0, $zero, rt_print_text
    la $a0, rt_true_text
rt_print_text:
    li $v0, 4
    syscall
    jr $ra
rt_print_null:
    la $a0, rt_null_text
    j rt_print_text
rt_print_number:
    l.d $f12, 8($a0)
    li $t0, 2147483647
    mtc1 $t0, $f0
    cvt.d.w $f0, $f0
    c.le.d $f12, $f0
    bc1f rt_print_double
    li $t0, -2147483648
    mtc1 $t0, $f0
    cvt.d.w $f0, $f0
    c.le.d $f0, $f12
    bc1f rt_print_double
    trunc.w.d $f0, $f12
    cvt.d.w $f2, $f0
    c.eq.d $f2, $f12
    bc1f rt_print_double
    mfc1 $a0, $f0
    li $v0, 1
    syscall
    jr $ra
rt_print_double:
    li $v0, 3
    syscall
    jr $ra

# Binary long reduction avoids narrowing the quotient to a 32-bit integer.
rt_fmod:
    abs.d $f4, $f0
    abs.d $f6, $f2
    li $t6, 0x7ff00000
    mfc1 $t7, $f1
    and $t7, $t7, $t6
    beq $t7, $t6, rt_fmod_nan
    mfc1 $t7, $f3
    and $t7, $t7, $t6
    beq $t7, $t6, rt_fmod_original
    mov.d $f8, $f6
    li $t6, 2
    mtc1 $t6, $f10
    cvt.d.w $f10, $f10
rt_fmod_scale:
    c.lt.d $f4, $f8
    bc1t rt_fmod_sign
    mul.d $f12, $f8, $f10
    c.le.d $f12, $f4
    bc1f rt_fmod_reduce
    mov.d $f8, $f12
    j rt_fmod_scale
rt_fmod_reduce:
    c.lt.d $f4, $f8
    bc1t rt_fmod_half
    sub.d $f4, $f4, $f8
rt_fmod_half:
    c.eq.d $f8, $f6
    bc1t rt_fmod_sign
    div.d $f8, $f8, $f10
    j rt_fmod_reduce
rt_fmod_sign:
    mtc1 $zero, $f6
    cvt.d.w $f6, $f6
    c.lt.d $f0, $f6
    bc1f rt_fmod_done
    neg.d $f4, $f4
rt_fmod_done:
    jr $ra
rt_fmod_original:
    mov.d $f4, $f0
    jr $ra
rt_fmod_nan:
    mtc1 $zero, $f4
    cvt.d.w $f4, $f4
    div.d $f4, $f4, $f4
    jr $ra

rt_undefined:
    la $a0, rt_msg_undefined
    j rt_fault
rt_numeric_error:
    la $a0, rt_msg_numeric
    j rt_fault
rt_boolean_error:
    la $a0, rt_msg_boolean
    j rt_fault
rt_string_error:
    la $a0, rt_msg_string
    j rt_fault
rt_division_error:
    la $a0, rt_msg_division
    j rt_fault
rt_remainder_error:
    la $a0, rt_msg_remainder
    j rt_fault
rt_type_error:
    la $a0, rt_msg_type
rt_fault:
    li $v0, 4
    syscall
    li $v0, 4
    la $a0, rt_newline
    syscall
    li $a0, 1
    li $v0, 17
    syscall
.data
rt_msg_undefined: .asciiz "undefined binding"
rt_msg_numeric: .asciiz "invalid numeric operand"
rt_msg_boolean: .asciiz "nonboolean branch"
rt_msg_string: .asciiz "null string operand"
rt_msg_division: .asciiz "division by zero"
rt_msg_remainder: .asciiz "remainder by zero"
rt_msg_type: .asciiz "dynamic value type mismatch"
.text
`

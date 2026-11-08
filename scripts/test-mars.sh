#!/usr/bin/env bash
# Public toolchain smoke test: actual MIPS output and assembler rejection.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "$root/testdata/mips/.smoke.XXXXXX")
trap 'rm -rf "$work"' EXIT
printf 'MARS OK\n' > "$work/expected"
bash "$root/scripts/mars.sh" "$root/testdata/mips/smoke.asm" > "$work/out" 2> "$work/err"
cmp "$work/expected" "$work/out"
# MARS writes one separator newline to its message stream on clean exit.
printf '\n' > "$work/expected-err"
cmp "$work/expected-err" "$work/err"
set +e
bash "$root/scripts/mars.sh" "$root/testdata/mips/invalid.asm" > "$work/out" 2> "$work/err"
status=$?
set -e
test "$status" -eq 1
test ! -s "$work/out"
printf 'Error in %s line 3 column 5: "definitely_not_an_instruction" is not a recognized operator\n\nProcessing terminated due to errors.\n' \
    "$root/testdata/mips/invalid.asm" > "$work/expected-err"
cmp "$work/expected-err" "$work/err"
printf 'MARS smoke: execution and invalid assembly passed\n'

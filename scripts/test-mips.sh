#!/usr/bin/env bash
# End-to-end acceptance through the public CLI and real simulator.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$root"
go build -o "$work/compiscript-mips" ./cmd/compiscript-mips
before=$(shasum -a 256 testdata/mips/full-language.cps)
"$work/compiscript-mips" testdata/mips/full-language.cps > "$work/program.asm" 2> "$work/compiler.err"
test ! -s "$work/compiler.err"
bash scripts/mars.sh 1000000 "$work/program.asm" > "$work/out" 2> "$work/err"
cmp testdata/mips/full-language.stdout "$work/out"
printf '\n' > "$work/expected-err"
cmp "$work/expected-err" "$work/err"
test "$before" = "$(shasum -a 256 testdata/mips/full-language.cps)"
printf 'MIPS acceptance: source, CLI, assembly and simulator passed\n'

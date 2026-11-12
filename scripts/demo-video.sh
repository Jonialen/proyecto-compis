#!/usr/bin/env bash
# Ensayo aislado: conserva los artefactos aun si una comprobación falla.
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
mkdir -p .atl
work=$(mktemp -d "$root/.atl/demo-video.XXXXXX")
printf 'Artefactos: %s\n' "$work"
mkdir -p "$work/db" "$work/gocache" "$work/gotmp"
export GOCACHE="$work/gocache" GOTMPDIR="$work/gotmp" GOPROXY=off GOTOOLCHAIN=local
fixtures="$root/testdata/video/sql"
go build -o "$work/tac" ./cmd/compiscript-tac
go build -o "$work/mips" ./cmd/compiscript-mips
go build -o "$work/sql" ./cmd/sql
"$work/tac" testdata/video/mips.cps >"$work/program.tac" 2>"$work/tac.stderr"
test ! -s "$work/tac.stderr"
"$work/mips" testdata/video/mips.cps >"$work/program.asm" 2>"$work/mips-compile.stderr"
test ! -s "$work/mips-compile.stderr"
if bash scripts/mars.sh "$work/program.asm" >"$work/mars.stdout" 2>"$work/mars.stderr"; then code=0; else code=$?; fi
printf '%s\n' "$code" >"$work/mars.exit"
if [[ $code != 0 ]] || ! cmp -s "$work/mars.stdout" testdata/video/mips.stdout || ! cmp -s "$work/mars.stderr" testdata/video/mips.stderr; then
    printf 'MARS no coincide; revisar mars.stdout, mars.stderr y mars.exit\n' >&2
    exit 1
fi
printf 'MIPS: objeto, campos, override y fib(8) verificados.\n'

step() {
    local name=$1 expected_code=$2 expected_out=$3 expected_err=$4
    if "$work/sql" -db "$work/db" <"$fixtures/$name.sql" >"$work/$name.stdout" 2>"$work/$name.stderr"; then code=0; else code=$?; fi
    printf '%s\n' "$code" >"$work/$name.exit"
    if [[ $code != "$expected_code" ]] || ! cmp -s "$work/$name.stdout" "$fixtures/$expected_out" || ! cmp -s "$work/$name.stderr" "$fixtures/$expected_err"; then
        printf 'SQL %s no coincide; revisar streams y exit en %s\n' "$name" "$work" >&2
        exit 1
    fi
    printf 'SQL %s: salida y exit %s correctos.\n' "$name" "$code"
}
snapshot() {
    local name=$1
    "$work/sql" -db "$work/db" -format json <"$fixtures/select.sql" >"$work/$name.json" 2>"$work/$name.stderr"
    test ! -s "$work/$name.stderr"
}
reject() {
    local name=$1 label=$2
    snapshot "$label-before"
    step "$name" 1 empty "$name.stderr"
    snapshot "$label-after"
    if ! cmp -s "$work/$label-before.json" "$work/$label-after.json"; then
        printf 'SQL %s: estado público cambió\n' "$label" >&2
        exit 1
    fi
    printf 'SQL %s: ambos SELECT y esquema público sin cambios.\n' "$label"
}
step setup 0 setup.stdout empty
reject duplicate duplicate
reject missing missing
reject delete-blocked delete-blocked
reject drop-blocked drop-blocked
step delete-free 0 delete-free.stdout empty
step after-free 0 after-free.stdout empty
step clear-child 0 clear-child.stdout empty
step after-clear 0 after-clear.stdout empty
reject drop-blocked drop-empty-child
step drop-all 0 drop-all.stdout empty
printf 'Ensayo completado. ASM, TAC, base temporal y registros: %s\n' "$work"

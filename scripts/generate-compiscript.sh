#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
grammar="$root/docs/semestre2/entrega1/Compiscript.g4"
output="$root/internal/compiscript/frontend/generated"
jar="$root/tools/antlr/antlr-4.13.2-complete.jar"
checksum="$root/tools/antlr/antlr-4.13.2-complete.jar.sha256"
java_bin=${JAVA_BIN:-java}
gofmt_bin=${GOFMT_BIN:-gofmt}

usage() {
	printf 'usage: %s [--grammar PATH] [--output PATH] [--jar PATH] [--checksum PATH]\n' "$0" >&2
}

while (($#)); do
	case "$1" in
	--grammar|--output|--jar|--checksum)
		(($# >= 2)) || { usage; exit 2; }
		case "$1" in
		--grammar) grammar=$2 ;;
		--output) output=$2 ;;
		--jar) jar=$2 ;;
		--checksum) checksum=$2 ;;
		esac
		shift 2
		;;
	*) usage; exit 2 ;;
	esac
done

for path in "$grammar" "$jar" "$checksum"; do
	[[ -f "$path" ]] || { printf 'missing required file: %s\n' "$path" >&2; exit 1; }
done
command -v "$java_bin" >/dev/null || { printf 'Java executable not found: %s\n' "$java_bin" >&2; exit 1; }
command -v "$gofmt_bin" >/dev/null || { printf 'gofmt executable not found: %s\n' "$gofmt_bin" >&2; exit 1; }

expected=$(awk 'NF { print $1; exit }' "$checksum")
actual=$(sha256sum "$jar" | awk '{ print $1 }')
[[ -n "$expected" && "$actual" == "$expected" ]] || { printf 'ANTLR JAR checksum mismatch\n' >&2; exit 1; }

parent=$(dirname "$output")
base=$(basename "$output")
mkdir -p "$parent"
lock="$output.lock"
mkdir "$lock" 2>/dev/null || { printf 'generation already running for %s\n' "$output" >&2; exit 1; }
stage=$(mktemp -d "$parent/.${base}.tmp.XXXXXX")
backup=

cleanup() {
	status=$?
	if [[ -n "$backup" && -e "$backup" && ! -e "$output" ]]; then
		mv "$backup" "$output" || true
	fi
	rm -rf "$stage"
	rmdir "$lock" 2>/dev/null || true
	exit "$status"
}
trap cleanup EXIT HUP INT TERM

"$java_bin" -jar "$jar" -Dlanguage=Go -package generated -visitor -no-listener -o "$stage" "$grammar"
find "$stage" -type f -name '*.go' -exec "$gofmt_bin" -w {} +
find "$stage" -type f -name '*.go' -print -quit | grep -q . || { printf 'ANTLR generated no Go files\n' >&2; exit 1; }

if [[ -d "$output" ]] && diff -qr "$output" "$stage" >/dev/null; then
	exit 0
fi
if [[ -e "$output" ]]; then
	backup="${output}.backup.$$"
	mv "$output" "$backup"
fi
if ! mv "$stage" "$output"; then
	[[ -z "$backup" || ! -e "$backup" ]] || mv "$backup" "$output"
	exit 1
fi
if [[ -n "$backup" ]]; then
	rm -rf "$backup"
	backup=
fi

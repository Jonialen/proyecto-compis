#!/usr/bin/env bash
# Download the original MARS release into the repository-local ignored cache.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
cache="$root/tools/mars"
mkdir -p "$cache"
if [[ -f "$cache/Mars4_5.jar" ]]; then
    (cd "$cache" && shasum -a 256 -c SHA256SUMS)
    exit 0
fi
partial="$cache/Mars4_5.jar.partial"
trap 'rm -f "$partial"' EXIT
curl --fail --location --connect-timeout 15 --max-time 120 \
    'https://github.com/dpetersanderson/MARS/releases/download/v.4.5.1/Mars4_5.jar' \
    -o "$partial"
expected=$(awk '{print $1}' "$cache/SHA256SUMS")
actual=$(shasum -a 256 "$partial" | awk '{print $1}')
if [[ "$actual" != "$expected" ]]; then
    printf 'MARS: checksum inválido; no se instaló ni ejecutó el archivo.\n' >&2
    exit 1
fi
mv "$partial" "$cache/Mars4_5.jar"
printf 'MARS 4.5 instalado en tools/mars/Mars4_5.jar\n'

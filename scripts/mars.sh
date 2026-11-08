#!/usr/bin/env bash
# Run the pinned local simulator; never install global tools or preferences.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
if ! command -v java >/dev/null 2>&1; then
    printf 'MARS requiere Java disponible en PATH.\n' >&2
    exit 127
fi
if [[ ! -f "$root/tools/mars/Mars4_5.jar" ]]; then
    printf 'MARS no está instalado: ejecutá bash scripts/install-mars.sh\n' >&2
    exit 127
fi
(cd "$root/tools/mars" && shasum -a 256 -c SHA256SUMS >/dev/null)
mkdir -p "$root/tools/mars/.prefs/.java/.userPrefs"
exec java -Djava.awt.headless=true \
    -Djava.util.prefs.userRoot="$root/tools/mars/.prefs" \
    -Djava.util.prefs.systemRoot="$root/tools/mars/.prefs/system" \
    -jar "$root/tools/mars/Mars4_5.jar" nc me ae1 se2 "$@"

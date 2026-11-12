#!/usr/bin/env bash
# Publishes the lead's task snapshot into the Flowra feed.
# The export must finish with success before the import starts: a footer proves the content is whole,
# not that the exporter ended well.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LEAD_TASK="${LEAD_TASK:-$HOME/projects/lead/bin/lead-task}"
LEADFEED="${LEADFEED:-$ROOT_DIR/bin/leadfeed}"

if [[ ! -x "$LEADFEED" ]]; then
	echo "[leadfeed-sync] $LEADFEED not found; run: make build" >&2
	exit 1
fi

snapshot="$(mktemp "${TMPDIR:-/tmp}/leadfeed-snapshot.XXXXXX")"
trap 'rm -f "$snapshot"' EXIT

export_status=0
"$LEAD_TASK" export > "$snapshot" || export_status=$?
if [[ $export_status -ne 0 ]]; then
	echo "[leadfeed-sync] $LEAD_TASK export exited with $export_status; nothing imported" >&2
	"$LEADFEED" fail "lead-task export exited with $export_status" || true
	exit 1
fi

"$LEADFEED" import "$snapshot"

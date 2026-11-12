#!/usr/bin/env bash
# Publishes the lead's task snapshot into the Flowra feed and passes owner answers to the lead's journal.
# The export must finish with success before the import starts: a footer proves the content is whole,
# not that the exporter ended well. Delivery runs whatever happened to the import, and the exit code is
# non-zero if either part failed.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LEAD_TASK="${LEAD_TASK:-$HOME/projects/lead/bin/lead-task}"
LEADFEED="${LEADFEED:-$ROOT_DIR/bin/leadfeed}"
LOCK_FILE="${LEADFEED_SYNC_LOCK:-${XDG_RUNTIME_DIR:-/tmp}/leadfeed-sync.lock}"
LOCK_WAIT="${LEADFEED_SYNC_LOCK_WAIT:-90}"

if [[ ! -x "$LEADFEED" ]]; then
	echo "[leadfeed-sync] $LEADFEED not found; run: make build" >&2
	exit 1
fi

exec 9>"$LOCK_FILE"
if ! flock -w "$LOCK_WAIT" 9; then
	echo "[leadfeed-sync] another run holds $LOCK_FILE for more than ${LOCK_WAIT}s" >&2
	exit 1
fi

snapshot="$(mktemp "${TMPDIR:-/tmp}/leadfeed-snapshot.XXXXXX")"
trap 'rm -f "$snapshot"' EXIT

failed=0

export_status=0
"$LEAD_TASK" export > "$snapshot" || export_status=$?
if [[ $export_status -ne 0 ]]; then
	echo "[leadfeed-sync] $LEAD_TASK export exited with $export_status; nothing imported" >&2
	"$LEADFEED" fail "lead-task export exited with $export_status" || true
	failed=1
elif ! "$LEADFEED" import "$snapshot"; then
	failed=1
fi

"$LEADFEED" deliver --lead-task "$LEAD_TASK" || failed=1

exit "$failed"

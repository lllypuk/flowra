#!/usr/bin/env bash
# Builds the binaries and installs the API and the feed sync as systemd user units; safe to repeat.
# Tokens are printed once, the moment they are generated and before anything is written or can fail;
# the files keep only SHA-256 digests. Settings in flowra.env persist unless set in the environment again.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="$HOME/.config/flowra"
UNIT_DIR="$HOME/.config/systemd/user"
ENV_FILE="$CONFIG_DIR/flowra.env"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

env_value() { [[ -f "$ENV_FILE" ]] && sed -n "s/^$1=//p" "$ENV_FILE" | head -n 1 || true; }

persisted_lead_task="$(env_value LEAD_TASK)"
persisted_lead_session="$(env_value LEAD_SESSION)"
persisted_agterm_socket="$(env_value AGTERM_SOCKET)"
LEAD_TASK="${LEAD_TASK:-${persisted_lead_task:-$HOME/projects/lead/bin/lead-task}}"
LEAD_SESSION="${LEAD_SESSION:-$persisted_lead_session}"
AGTERM_SOCKET="${AGTERM_SOCKET:-${persisted_agterm_socket:-$HOME/.local/share/agterm/agterm.sock}}"

for tool in go docker-compose systemctl; do
	command -v "$tool" >/dev/null || { echo "[install] $tool not found" >&2; exit 1; }
done

new_token() { head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'; }
digest() { printf '%s' "$1" | sha256sum | cut -d' ' -f1; }

umask 077
mkdir -p "$CONFIG_DIR" "$UNIT_DIR"

owner_digest="$(env_value AUTH_OWNER_TOKEN_SHA256)"
phone_digest=""
[[ -f "$CONFIG_FILE" ]] && phone_digest="$(sed -n 's/^ *sha256: "\([0-9a-f]*\)"$/\1/p' "$CONFIG_FILE")"

show_token() {
	[[ -n "${shown:-}" ]] || echo "Save these tokens now; they are not shown again:"
	shown=1
	printf '%s: %s\n' "$1" "$2"
}
if [[ -z "$owner_digest" ]]; then
	token="$(new_token)"
	show_token "owner (browser of this laptop)" "$token"
	owner_digest="$(digest "$token")"
fi
if [[ -z "$phone_digest" ]]; then
	token="$(new_token)"
	show_token "phone" "$token"
	phone_digest="$(digest "$token")"
fi

# The owner token comes from the environment: a second "owner" entry in auth.tokens would be a duplicate.
awk -v digest="$phone_digest" '
	{ print }
	/^auth:$/ { print "  tokens:"; print "    - name: \"phone\""; print "      sha256: \"" digest "\"" }
' "$ROOT_DIR/configs/config.yaml" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

agterm_dir=""
if agtermctl_path="$(command -v agtermctl)"; then
	agterm_dir="$(dirname "$agtermctl_path"):"
fi
lead_session_line=""
[[ -n "$LEAD_SESSION" ]] && lead_session_line="LEAD_SESSION=$LEAD_SESSION"
cat > "$ENV_FILE.tmp" <<ENV
CONFIG_PATH=$CONFIG_FILE
AUTH_OWNER_TOKEN_SHA256=$owner_digest
FLOWRA_WORKER=true
LEAD_TASK=$LEAD_TASK
${lead_session_line}
AGTERM_SOCKET=$AGTERM_SOCKET
PATH=${agterm_dir}/usr/local/bin:/usr/bin:/bin
ENV
mv "$ENV_FILE.tmp" "$ENV_FILE"

for unit in flowra-api.service leadfeed-sync.service leadfeed-sync.timer; do
	sed -e "s|@ROOT@|$ROOT_DIR|g" -e "s|@ENV_FILE@|$ENV_FILE|g" \
		"$ROOT_DIR/scripts/units/$unit.in" > "$UNIT_DIR/$unit"
done

(cd "$ROOT_DIR" && go build -o bin/api ./cmd/api && go build -o bin/leadfeed ./cmd/leadfeed)

docker-compose -f "$ROOT_DIR/docker-compose.yml" up -d mongodb redis mongo-init

systemctl --user daemon-reload
systemctl --user enable --now flowra-api.service leadfeed-sync.timer
systemctl --user try-restart flowra-api.service

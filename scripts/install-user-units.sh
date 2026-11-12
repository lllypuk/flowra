#!/usr/bin/env bash
# Builds the binaries and installs the API and the feed sync as systemd user units; safe to repeat.
# Tokens are generated and printed only on the first run; the files keep their SHA-256 digests.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="$HOME/.config/flowra"
UNIT_DIR="$HOME/.config/systemd/user"
ENV_FILE="$CONFIG_DIR/flowra.env"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

LEAD_TASK="${LEAD_TASK:-$HOME/projects/lead/bin/lead-task}"
AGTERM_SOCKET="${AGTERM_SOCKET:-$HOME/.local/share/agterm/agterm.sock}"

for tool in go docker-compose systemctl; do
	command -v "$tool" >/dev/null || { echo "[install] $tool not found" >&2; exit 1; }
done

new_token() { head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'; }
digest() { printf '%s' "$1" | sha256sum | cut -d' ' -f1; }

umask 077
mkdir -p "$CONFIG_DIR" "$UNIT_DIR"

owner_digest=""
phone_digest=""
[[ -f "$ENV_FILE" ]] && owner_digest="$(sed -n 's/^AUTH_OWNER_TOKEN_SHA256=//p' "$ENV_FILE")"
[[ -f "$CONFIG_FILE" ]] && phone_digest="$(sed -n 's/^ *sha256: "\([0-9a-f]*\)"$/\1/p' "$CONFIG_FILE")"

printed=""
if [[ -z "$owner_digest" ]]; then
	token="$(new_token)"
	owner_digest="$(digest "$token")"
	printed+="owner (browser of this laptop): $token"$'\n'
fi
if [[ -z "$phone_digest" ]]; then
	token="$(new_token)"
	phone_digest="$(digest "$token")"
	printed+="phone: $token"$'\n'
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
[[ -n "${LEAD_SESSION:-}" ]] && lead_session_line="LEAD_SESSION=$LEAD_SESSION"
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

if [[ -n "$printed" ]]; then
	echo "Save these tokens now; they are not shown again:"
	printf '%s' "$printed"
fi

#!/usr/bin/env bash
# sync-env.sh — Merge local .env.example schema with server .env values.
#
# Rules:
#   • Server value wins if the key exists on server (keeps secrets/prod config).
#   • New keys (in example but not on server) are added with example defaults.
#   • Removed keys (on server but not in example) are dropped.
#   • Renamed keys: old server name mapped to new local name, value preserved.
#
# Usage: bash scripts/sync-env.sh [--dry-run]
set -euo pipefail

REMOTE_USER=komkov.vv
REMOTE_HOST=185.104.114.242
REMOTE_ENV=/home/komkov.vv/mii_api/.env
LOCAL_EXAMPLE=.env.example

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

[ -f "$LOCAL_EXAMPLE" ] || { echo "ERROR: $LOCAL_EXAMPLE not found"; exit 1; }

# Download server .env to temp file
TMP_SERVER=$(mktemp)
TMP_MERGED=$(mktemp)
trap 'rm -f $TMP_SERVER $TMP_MERGED' EXIT

echo "Fetching server .env..."
ssh "$REMOTE_USER@$REMOTE_HOST" "cat $REMOTE_ENV" > "$TMP_SERVER"

# ── Python merge ──────────────────────────────────────────────────────────────
python3 <<PYEOF > "$TMP_MERGED"
import re

# Old server key -> new local key (value from server preserved under new name)
RENAMES = {
    "SSH_USER":             "OVPN_SSH_USER",
    "SSH_STATUS_PATH":      "OVPN_STATUS_PATH",
    "SSH_CREATE_USER_FILE": "OVPN_CREATE_USER_FILE",
    "SSH_REVOKE_USER_FILE": "OVPN_REVOKE_USER_FILE",
    "SSH_CONFIGS_DIR":      "OVPN_CONFIGS_DIR",
    "JWT_SECRET":           "JWT_SECRET_AUTH",
}

def parse_env(path):
    vals = {}
    for line in open(path):
        line = line.rstrip('\n')
        if not line.strip() or line.strip().startswith('#'):
            continue
        m = re.match(r'^([A-Za-z_][A-Za-z0-9_]*)=(.*)', line)
        if m:
            vals[m.group(1)] = m.group(2)
    return vals

server_vals = parse_env("$TMP_SERVER")
example_lines = open("$LOCAL_EXAMPLE").read().splitlines()

# Apply renames: map old server key value → new local key
merged_vals = dict(server_vals)
for old, new in RENAMES.items():
    if old in server_vals and new not in server_vals:
        merged_vals[new] = server_vals[old]

# Walk the example template; replace value with server value if available
for line in example_lines:
    m = re.match(r'^([A-Za-z_][A-Za-z0-9_]*)=(.*)', line)
    if m:
        key = m.group(1)
        val = merged_vals.get(key, m.group(2))
        print(f"{key}={val}")
    else:
        print(line)
PYEOF

# ── Diff ──────────────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}── Diff (server .env → merged .env) ───────────────────────────────${NC}"
diff --color=always -u "$TMP_SERVER" "$TMP_MERGED" || true
echo -e "${CYAN}────────────────────────────────────────────────────────────────────${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}--dry-run: no changes uploaded.${NC}"
    exit 0
fi

read -r -p "Upload merged .env to server? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

cat "$TMP_MERGED" | ssh "$REMOTE_USER@$REMOTE_HOST" "sudo tee $REMOTE_ENV > /dev/null"
echo -e "${GREEN}==> .env synced successfully.${NC}"

#!/usr/bin/env bash
# Remote-deploy script for Micocards. Called from .github/workflows/deploy.yml
# after the build steps. Ships frontend/dist, backend/bin/api and the
# per-context migration tree to the prod VDSina box over sshpass, applies
# goose migrations, swaps the binary atomically, restarts the systemd unit
# and probes /api/healthz.
#
# Expected env (provided by the workflow):
#   DEPLOY_SSH_HOST       bare host (e.g. v811467.hosted-by-vdsina.com)
#   DEPLOY_SSH_USER       remote user (e.g. root)
#   DEPLOY_SSH_PASSWORD   password for sshpass
#   DEPLOY_DOMAIN         public domain for the health probe (e.g. v811467...)
#
# Migration layout choice:
#   backend/migrations/ holds four per-context subdirs (shared, iam, decks,
#   practice). Goose takes a single `-dir` per invocation, so we keep the
#   layout intact on the server and apply the dirs in deterministic order
#   from a shell loop (option (a) from the brief). That mirrors the local
#   Makefile target `backend-migrate-up`. Order is `shared → iam → decks →
#   practice` and must NOT change — later contexts may reference shared
#   tables. Goose is assumed to be on the server's PATH per docs/stack.md.

set -euo pipefail

# --- preconditions ---------------------------------------------------------
: "${DEPLOY_SSH_HOST:?DEPLOY_SSH_HOST is required}"
: "${DEPLOY_SSH_USER:?DEPLOY_SSH_USER is required}"
: "${DEPLOY_SSH_PASSWORD:?DEPLOY_SSH_PASSWORD is required}"
: "${DEPLOY_DOMAIN:?DEPLOY_DOMAIN is required}"

# sshpass reads the password from SSHPASS when invoked with -e. Exporting it
# once at the top means the password never appears on a command line or in
# `set -x` output — keep `set -x` OFF for the rest of the script and rely on
# the `step` helper for progress logs.
export SSHPASS="$DEPLOY_SSH_PASSWORD"

REMOTE="${DEPLOY_SSH_USER}@${DEPLOY_SSH_HOST}"
SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o UserKnownHostsFile="$HOME/.ssh/known_hosts")

# Helpers ------------------------------------------------------------------
step() { printf '\n==> %s\n' "$*"; }

ssh_run() {
    sshpass -e ssh "${SSH_OPTS[@]}" "$REMOTE" "$@"
}

scp_to() {
    # $1 local path, $2 remote path
    sshpass -e scp "${SSH_OPTS[@]}" "$1" "$REMOTE:$2"
}

rsync_to() {
    # $1 local dir (with trailing slash), $2 remote dir
    sshpass -e rsync -avz --delete-after \
        -e "ssh ${SSH_OPTS[*]}" \
        "$1" "$REMOTE:$2"
}

# --- artifact sanity checks ------------------------------------------------
step "Checking build artifacts"
test -x backend/bin/api      || { echo "backend/bin/api missing or not executable" >&2; exit 1; }
test -d frontend/dist        || { echo "frontend/dist missing" >&2; exit 1; }
test -d backend/migrations   || { echo "backend/migrations missing" >&2; exit 1; }

# --- 1. frontend ----------------------------------------------------------
step "Sync frontend → /var/www/micocards/"
rsync_to "frontend/dist/" "/var/www/micocards/"

# --- 2. backend binary (staged for atomic swap) ---------------------------
step "Upload backend binary → /opt/micocards/bin/api.new"
ssh_run "mkdir -p /opt/micocards/bin"
scp_to "backend/bin/api" "/opt/micocards/bin/api.new"

# --- 3. migrations (preserve per-context layout) --------------------------
step "Sync migrations → /opt/micocards/migrations/"
ssh_run "mkdir -p /opt/micocards/migrations"
rsync_to "backend/migrations/" "/opt/micocards/migrations/"

# --- 4. ensure goose is on the server -------------------------------------
# docs/stack.md assumes goose is present, but a fresh server may not have it.
# Idempotent fetch of the pinned linux/amd64 release binary into /usr/local/bin.
GOOSE_VERSION="v3.27.1"
step "Ensure goose ${GOOSE_VERSION} installed at /usr/local/bin/goose"
ssh_run "bash -se" <<REMOTE_GOOSE
set -euo pipefail
if command -v goose >/dev/null 2>&1; then
    echo "goose already present: \$(goose -version 2>&1 | head -1)"
    exit 0
fi
echo "goose missing — downloading ${GOOSE_VERSION}"
curl -fsSL -o /usr/local/bin/goose \
    "https://github.com/pressly/goose/releases/download/${GOOSE_VERSION}/goose_linux_x86_64"
chmod +x /usr/local/bin/goose
goose -version
REMOTE_GOOSE

# --- 5. apply migrations (goose, per-context order) -----------------------
# DATABASE_URL is held in /opt/micocards/env/api.env (chmod 600, owned by
# www-data) per docs/stack.md. We source it into the SSH shell only for the
# goose invocation so it never has to be exported by the CI runner.
step "Apply migrations (goose) in order: shared, iam, decks, practice"
ssh_run "bash -se" <<'REMOTE_MIGRATE'
set -euo pipefail
set -a
# shellcheck disable=SC1091
source /opt/micocards/env/api.env
set +a
: "${DATABASE_URL:?DATABASE_URL missing from /opt/micocards/env/api.env}"

for ctx in shared iam decks practice; do
    echo "==> goose $ctx"
    goose -dir "/opt/micocards/migrations/$ctx" postgres "$DATABASE_URL" up
done
REMOTE_MIGRATE

# --- 5. atomic binary swap + restart --------------------------------------
step "Swap binary and restart micocards-api"
ssh_run "bash -se" <<'REMOTE_SWAP'
set -euo pipefail
mv /opt/micocards/bin/api.new /opt/micocards/bin/api
chmod +x /opt/micocards/bin/api
systemctl restart micocards-api
REMOTE_SWAP

# --- 6. health probe -------------------------------------------------------
# Retry briefly so the API has a moment to bind its socket after restart.
step "Health probe https://${DEPLOY_DOMAIN}/api/healthz"
if ! curl -fsS --max-time 10 --retry 5 --retry-delay 3 \
        "https://${DEPLOY_DOMAIN}/api/healthz" > /tmp/healthz.out; then
    echo "Health probe FAILED — dumping recent service logs" >&2
    ssh_run "journalctl -u micocards-api --no-pager -n 30" || true
    exit 1
fi
echo "Health probe OK:"
cat /tmp/healthz.out
echo

# --- 7. log tail (informational, on success) ------------------------------
step "Recent micocards-api journal (last 30 lines)"
ssh_run "journalctl -u micocards-api --no-pager -n 30" || true

step "Deploy complete"

#!/usr/bin/env bash
# Remote-deploy script for Micocards. Called from .github/workflows/deploy.yml
# after the build steps. Ships frontend/dist, backend/bin/api, backend/bin/migrate
# and the per-context migration tree to the prod VDSina box over sshpass,
# applies migrations via the in-process runner, swaps the binary atomically,
# restarts the systemd unit, syncs nginx config and probes /api/healthz.
#
# Expected env (provided by the workflow):
#   DEPLOY_SSH_HOST       bare host (e.g. v811467.hosted-by-vdsina.com)
#   DEPLOY_SSH_USER       remote user (e.g. root)
#   DEPLOY_SSH_PASSWORD   password for sshpass
#   DEPLOY_DOMAIN         public domain for the health probe (e.g. v811467...)
#
# Migration layout:
#   backend/migrations/ holds four per-context subdirs (shared, iam, decks,
#   practice). The in-process runner at backend/cmd/migrate iterates them in
#   the canonical order (shared → iam → decks → practice) internally, so we
#   ship the whole tree and invoke a single binary.

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
test -x backend/bin/api       || { echo "backend/bin/api missing or not executable" >&2; exit 1; }
test -x backend/bin/migrate   || { echo "backend/bin/migrate missing or not executable" >&2; exit 1; }
test -d frontend/dist         || { echo "frontend/dist missing" >&2; exit 1; }
test -d backend/migrations    || { echo "backend/migrations missing" >&2; exit 1; }
test -f infra/nginx/micocards.conf || { echo "infra/nginx/micocards.conf missing" >&2; exit 1; }

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

# --- 4. migrate binary (staged + atomic mv) -------------------------------
step "Upload migrate binary → /opt/micocards/bin/migrate"
scp_to "backend/bin/migrate" "/opt/micocards/bin/migrate.new"
ssh_run "bash -se" <<'REMOTE_MIGRATE_BIN'
set -euo pipefail
mv /opt/micocards/bin/migrate.new /opt/micocards/bin/migrate
chmod +x /opt/micocards/bin/migrate
REMOTE_MIGRATE_BIN

# --- 5. apply migrations (in-process runner) ------------------------------
# The runner iterates shared → iam → decks → practice internally, so no shell
# loop here. DATABASE_URL is sourced from /opt/micocards/env/api.env (chmod
# 600, owned by www-data) per docs/stack.md and never exported by CI.
step "Apply migrations (in-process runner)"
ssh_run "bash -se" <<'REMOTE_MIGRATE'
set -euo pipefail
set -a
# shellcheck disable=SC1091
source /opt/micocards/env/api.env
set +a
: "${DATABASE_URL:?DATABASE_URL missing from /opt/micocards/env/api.env}"

/opt/micocards/bin/migrate -dir /opt/micocards/migrations -db "$DATABASE_URL"
REMOTE_MIGRATE

# --- 5. atomic binary swap + restart --------------------------------------
step "Swap binary and restart micocards-api"
ssh_run "bash -se" <<'REMOTE_SWAP'
set -euo pipefail
mv /opt/micocards/bin/api.new /opt/micocards/bin/api
chmod +x /opt/micocards/bin/api
systemctl restart micocards-api
REMOTE_SWAP

# --- 6. nginx config sync + reload ----------------------------------------
# Upload to a staging path, then validate with `nginx -t` BEFORE swapping the
# live config. If validation fails we abort without touching the running
# config so the site stays up.
step "Sync nginx config and reload"
scp_to "infra/nginx/micocards.conf" "/etc/nginx/sites-available/micocards.new"
ssh_run "bash -se" <<'REMOTE_NGINX'
set -euo pipefail
mv /etc/nginx/sites-available/micocards.new /etc/nginx/sites-available/micocards
ln -sf /etc/nginx/sites-available/micocards /etc/nginx/sites-enabled/micocards
if ! nginx -t 2>/tmp/nginx-t.err; then
    echo "nginx -t FAILED — config NOT reloaded" >&2
    cat /tmp/nginx-t.err >&2
    exit 1
fi
systemctl reload nginx
REMOTE_NGINX

# --- 7. health probe -------------------------------------------------------
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

# --- 8. log tail (informational, on success) ------------------------------
step "Recent micocards-api journal (last 30 lines)"
ssh_run "journalctl -u micocards-api --no-pager -n 30" || true

step "Deploy complete"

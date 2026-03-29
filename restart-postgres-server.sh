#!/usr/bin/env bash
# ============================================================
#  go2postgres – Restart PostgreSQL Server
#  Usage:  sudo bash restart-postgres-server.sh
# ============================================================
set -euo pipefail

# --- Configuration ------------------------------------------------
PG_BIN="/usr/lib/postgresql/16/bin"
PG_DATA="/opt/go2postgres/data/postgres"
PG_LOG="/opt/go2postgres/data/logs/postgresql.log"
PG_USER="postgres"

# --- Colour helpers -----------------------------------------------
green()  { printf '\033[0;32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[0;33m%s\033[0m\n' "$*"; }
red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }

# --- Pre-flight checks --------------------------------------------
if [[ $EUID -ne 0 ]]; then
    red "ERROR: This script must be run as root (or via sudo)."
    exit 1
fi

if [[ ! -d "${PG_DATA}" ]]; then
    red "ERROR: PostgreSQL data directory not found: ${PG_DATA}"
    exit 1
fi

if [[ ! -f "${PG_BIN}/pg_ctl" ]]; then
    red "ERROR: pg_ctl not found at ${PG_BIN}/pg_ctl"
    exit 1
fi

# --- Ensure correct ownership -------------------------------------
green ">>> Ensuring correct ownership and permissions..."
chown -R "${PG_USER}:${PG_USER}" "${PG_DATA}"

# All parent directories need to be traversable by postgres user
chmod 755 /opt/go2postgres
chmod 755 /opt/go2postgres/data

# Ensure log directory exists
mkdir -p "$(dirname "${PG_LOG}")"
chown "${PG_USER}:${PG_USER}" "$(dirname "${PG_LOG}")"

# --- Check current status -----------------------------------------
green ">>> Checking PostgreSQL status..."
if sudo -u "${PG_USER}" "${PG_BIN}/pg_ctl" -D "${PG_DATA}" status &>/dev/null; then
    yellow ">>> PostgreSQL is running. Stopping..."
    sudo -u "${PG_USER}" "${PG_BIN}/pg_ctl" -D "${PG_DATA}" stop -m fast
    sleep 2
else
    yellow ">>> PostgreSQL is not running."
fi

# --- Start PostgreSQL ---------------------------------------------
green ">>> Starting PostgreSQL..."
sudo -u "${PG_USER}" "${PG_BIN}/pg_ctl" -D "${PG_DATA}" -l "${PG_LOG}" start

sleep 2

# --- Verify -------------------------------------------------------
if sudo -u "${PG_USER}" "${PG_BIN}/pg_ctl" -D "${PG_DATA}" status &>/dev/null; then
    green ">>> PostgreSQL started successfully!"
    
    # Show connection info
    PG_PORT=$(grep -E "^port\s*=" "${PG_DATA}/postgresql.conf" 2>/dev/null | sed 's/.*=\s*//' | tr -d ' ' || echo "5438")
    echo ""
    green "Connection info:"
    echo "  Host: localhost"
    echo "  Port: ${PG_PORT}"
    echo "  Log:  ${PG_LOG}"
else
    red ">>> PostgreSQL failed to start!"
    echo ""
    yellow "Last 20 lines of log:"
    tail -20 "${PG_LOG}" 2>/dev/null || echo "No log available"
    exit 1
fi

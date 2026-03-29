#!/bin/bash
#
# fix-postgres-initial-setup-once.sh
# One-time script to create the PostgreSQL admin user from .env
# Run this ONCE after initial PostgreSQL setup if you forgot to specify admin credentials
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

echo ">>> go2postgres PostgreSQL Admin User Setup (One-Time Fix)"
echo ""

# Check if .env exists
if [ ! -f "$ENV_FILE" ]; then
    echo "ERROR: .env file not found at $ENV_FILE"
    exit 1
fi

# Load variables from .env
source "$ENV_FILE"

# Validate required variables
if [ -z "$POSTGRES_SUPERUSER" ]; then
    echo "ERROR: POSTGRES_SUPERUSER not set in .env"
    exit 1
fi

if [ -z "$POSTGRES_SUPERPASS" ]; then
    echo "ERROR: POSTGRES_SUPERPASS not set in .env"
    exit 1
fi

POSTGRES_PORT="${POSTGRES_PORT:-5438}"
POSTGRES_BIN_PATH="${POSTGRES_BIN_PATH:-/usr/lib/postgresql/16/bin}"
PSQL="${POSTGRES_BIN_PATH}/psql"

echo ">>> Configuration:"
echo "    Admin User: $POSTGRES_SUPERUSER"
echo "    Port:       $POSTGRES_PORT"
echo "    Psql:       $PSQL"
echo ""

# Check if psql exists
if [ ! -x "$PSQL" ]; then
    echo "ERROR: psql not found at $PSQL"
    exit 1
fi

# Check if role already exists
echo ">>> Checking if role '$POSTGRES_SUPERUSER' already exists..."
ROLE_EXISTS=$(sudo -u postgres "$PSQL" -p "$POSTGRES_PORT" -tAc "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_SUPERUSER'" 2>/dev/null || echo "")

if [ "$ROLE_EXISTS" = "1" ]; then
    echo ">>> Role '$POSTGRES_SUPERUSER' already exists. Updating password..."
    sudo -u postgres "$PSQL" -p "$POSTGRES_PORT" -c "ALTER ROLE $POSTGRES_SUPERUSER WITH PASSWORD '$POSTGRES_SUPERPASS';"
    echo ">>> Password updated successfully!"
else
    echo ">>> Creating role '$POSTGRES_SUPERUSER'..."
    sudo -u postgres "$PSQL" -p "$POSTGRES_PORT" -c "CREATE ROLE $POSTGRES_SUPERUSER WITH LOGIN SUPERUSER CREATEDB CREATEROLE PASSWORD '$POSTGRES_SUPERPASS';"
    echo ">>> Role created successfully!"
fi

# Test connection
echo ""
echo ">>> Testing connection with new credentials..."
export PGPASSWORD="$POSTGRES_SUPERPASS"
if "$PSQL" -h 127.0.0.1 -p "$POSTGRES_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "SELECT 1;" > /dev/null 2>&1; then
    echo ">>> Connection test PASSED!"
else
    echo ">>> Connection test FAILED!"
    echo "    Check pg_hba.conf allows password auth for 127.0.0.1"
    exit 1
fi
unset PGPASSWORD

echo ""
echo ">>> Setup complete! You can now start go2postgres:"
echo "    sudo systemctl restart go2postgres"
echo ""

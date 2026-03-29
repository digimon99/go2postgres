#!/bin/bash
#===============================================================================
# go2postgres - Initial PostgreSQL and Project Setup Script
#===============================================================================
# This script:
# 1. Creates the required directory structure
# 2. Initializes a dedicated PostgreSQL instance on port 5438
# 3. Generates secure secrets and creates .env file
# 4. Creates the go2postgres admin database user
# 5. Configures PostgreSQL for multi-tenant operation
#
# Prerequisites:
# - PostgreSQL 14+ installed (postgresql-server package)
# - sudo/root access
# - openssl installed
#
# Usage:
#   chmod +x initial-postgres-and-project-setup.sh
#   sudo ./initial-postgres-and-project-setup.sh
#===============================================================================

set -e  # Exit on any error

#-------------------------------------------------------------------------------
# Configuration
#-------------------------------------------------------------------------------
GO2POSTGRES_BASE="/opt/go2postgres"
POSTGRES_DATA_DIR="${GO2POSTGRES_BASE}/data/postgres"
HTTP_PORT=8443
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"  # Override via: ADMIN_EMAIL=you@example.com sudo ./setup.sh

# PostgreSQL admin user for go2postgres operations
DB_ADMIN_USER="go2postgres_admin"

# PostgreSQL port - will be auto-detected (prefer 5432 if free, else 5438)
POSTGRES_PORT=""

# PostgreSQL binary paths - will be detected
INITDB_CMD=""
PG_CTL_CMD=""
PSQL_CMD=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#-------------------------------------------------------------------------------
# Helper Functions
#-------------------------------------------------------------------------------
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

generate_secret() {
    openssl rand -hex 32
}

generate_password() {
    # Generate a 32-char password with alphanumeric + some special chars
    openssl rand -base64 32 | tr -d '/+=' | head -c 32
}

#-------------------------------------------------------------------------------
# Find PostgreSQL Binaries
#-------------------------------------------------------------------------------
find_postgres_binaries() {
    log_info "Searching for PostgreSQL binaries..."

    # Common PostgreSQL binary locations
    POSTGRES_PATHS=(
        "/usr/lib/postgresql/16/bin"
        "/usr/lib/postgresql/15/bin"
        "/usr/lib/postgresql/14/bin"
        "/usr/lib/postgresql/13/bin"
        "/usr/pgsql-16/bin"
        "/usr/pgsql-15/bin"
        "/usr/pgsql-14/bin"
        "/usr/bin"
        "/usr/local/bin"
        "/usr/local/pgsql/bin"
    )

    for path in "${POSTGRES_PATHS[@]}"; do
        if [[ -x "${path}/initdb" ]] && [[ -x "${path}/pg_ctl" ]] && [[ -x "${path}/psql" ]]; then
            INITDB_CMD="${path}/initdb"
            PG_CTL_CMD="${path}/pg_ctl"
            PSQL_CMD="${path}/psql"
            
            local version=""
            if [[ -x "${path}/postgres" ]]; then
                version=$("${path}/postgres" --version 2>/dev/null | grep -oP '\d+\.\d+' | head -1)
            fi
            
            log_success "Found PostgreSQL ${version} at: ${path}"
            return 0
        fi
    done

    # Try using 'which' as fallback
    if command -v initdb &> /dev/null && command -v pg_ctl &> /dev/null && command -v psql &> /dev/null; then
        INITDB_CMD=$(which initdb)
        PG_CTL_CMD=$(which pg_ctl)
        PSQL_CMD=$(which psql)
        log_success "Found PostgreSQL in PATH"
        return 0
    fi

    return 1
}

#-------------------------------------------------------------------------------
# Auto-detect Best Port
#-------------------------------------------------------------------------------
detect_postgres_port() {
    log_info "Auto-detecting best PostgreSQL port..."

    # Prefer 5432 (standard) if available
    if ! ss -tuln 2>/dev/null | grep -q ":5432 "; then
        POSTGRES_PORT=5432
        log_success "Port 5432 is available (standard PostgreSQL port)"
        return 0
    fi

    # Try 5438 as fallback
    if ! ss -tuln 2>/dev/null | grep -q ":5438 "; then
        POSTGRES_PORT=5438
        log_warn "Port 5432 is in use, using port 5438 instead"
        return 0
    fi

    # Try a few more ports
    for port in 5433 5434 5435 5436 5437 5439 5440; do
        if ! ss -tuln 2>/dev/null | grep -q ":${port} "; then
            POSTGRES_PORT=$port
            log_warn "Ports 5432 and 5438 are in use, using port ${port}"
            return 0
        fi
    done

    log_error "No available ports found in range 5432-5440"
    return 1
}

#-------------------------------------------------------------------------------
# Pre-flight Checks
#-------------------------------------------------------------------------------
preflight_checks() {
    log_info "Running pre-flight checks..."

    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (sudo)"
        exit 1
    fi

    # Find PostgreSQL binaries
    if ! find_postgres_binaries; then
        log_error "PostgreSQL is not installed or binaries not found."
        echo ""
        log_info "To install PostgreSQL 16 on Ubuntu/Debian:"
        echo ""
        echo "  # Add PostgreSQL APT repository (for latest version)"
        echo "  sudo sh -c 'echo \"deb http://apt.postgresql.org/pub/repos/apt \$(lsb_release -cs)-pgdg main\" > /etc/apt/sources.list.d/pgdg.list'"
        echo "  wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -"
        echo "  sudo apt update"
        echo ""
        echo "  # Install PostgreSQL 16"
        echo "  sudo apt install -y postgresql-16 postgresql-contrib-16"
        echo ""
        echo "  # Or install from default Ubuntu repos (may be older version)"
        echo "  sudo apt install -y postgresql postgresql-contrib"
        echo ""
        exit 1
    fi

    # Auto-detect port
    if ! detect_postgres_port; then
        exit 1
    fi

    # Check if openssl is installed
    if ! command -v openssl &> /dev/null; then
        log_error "openssl is not installed. Please install openssl first."
        exit 1
    fi

    # Check if port is available (double-check)
    if ss -tuln 2>/dev/null | grep -q ":${POSTGRES_PORT} "; then
        log_error "Port ${POSTGRES_PORT} is already in use."
        exit 1
    fi

    # Check if port 8443 is available
    if ss -tuln | grep -q ":${HTTP_PORT} "; then
        log_warn "Port ${HTTP_PORT} is already in use. go2postgres may fail to start."
    fi

    log_success "Pre-flight checks passed"
}

#-------------------------------------------------------------------------------
# Create Directory Structure
#-------------------------------------------------------------------------------
create_directories() {
    log_info "Creating directory structure..."

    # Create base directories
    mkdir -p "${GO2POSTGRES_BASE}"
    mkdir -p "${GO2POSTGRES_BASE}/data/postgres"
    mkdir -p "${GO2POSTGRES_BASE}/data/backups"
    mkdir -p "${GO2POSTGRES_BASE}/data/logs"
    mkdir -p "${GO2POSTGRES_BASE}/config/ssl"

    # Set permissions
    chmod 750 "${GO2POSTGRES_BASE}"
    chmod 700 "${GO2POSTGRES_BASE}/data"
    chmod 700 "${GO2POSTGRES_BASE}/data/postgres"
    chmod 750 "${GO2POSTGRES_BASE}/data/backups"
    chmod 750 "${GO2POSTGRES_BASE}/data/logs"
    chmod 700 "${GO2POSTGRES_BASE}/config"

    log_success "Directory structure created at ${GO2POSTGRES_BASE}"
    
    echo ""
    echo "Directory structure:"
    echo "  ${GO2POSTGRES_BASE}/"
    echo "  ├── data/"
    echo "  │   ├── postgres/     # PostgreSQL data files"
    echo "  │   ├── backups/      # Database backups"
    echo "  │   ├── logs/         # Application logs"
    echo "  │   └── meta.db       # SQLite metadata (created at runtime)"
    echo "  ├── config/"
    echo "  │   └── ssl/          # TLS certificates"
    echo "  └── .env              # Environment configuration"
    echo ""
}

#-------------------------------------------------------------------------------
# Create System User
#-------------------------------------------------------------------------------
create_system_user() {
    log_info "Creating go2postgres system user..."

    if id "go2postgres" &>/dev/null; then
        log_warn "User 'go2postgres' already exists, skipping creation"
    else
        useradd --system --home-dir "${GO2POSTGRES_BASE}" --shell /bin/false go2postgres
        log_success "System user 'go2postgres' created"
    fi

    # Set ownership
    chown -R go2postgres:go2postgres "${GO2POSTGRES_BASE}"
}

#-------------------------------------------------------------------------------
# Initialize PostgreSQL
#-------------------------------------------------------------------------------
init_postgresql() {
    log_info "Initializing PostgreSQL instance..."

    # Check if already initialized
    if [[ -f "${POSTGRES_DATA_DIR}/PG_VERSION" ]]; then
        log_warn "PostgreSQL data directory already initialized, skipping"
        return
    fi

    # Use pre-detected initdb command
    if [[ -z "$INITDB_CMD" ]]; then
        log_error "initdb command not found. This should not happen."
        exit 1
    fi

    log_info "Using initdb: ${INITDB_CMD}"

    # Initialize the database cluster
    sudo -u go2postgres ${INITDB_CMD} \
        -D "${POSTGRES_DATA_DIR}" \
        --encoding=UTF8 \
        --locale=en_US.UTF-8 \
        --auth-local=peer \
        --auth-host=scram-sha-256

    log_success "PostgreSQL data directory initialized"
}

#-------------------------------------------------------------------------------
# Configure PostgreSQL
#-------------------------------------------------------------------------------
configure_postgresql() {
    log_info "Configuring PostgreSQL..."

    # Backup original configs
    if [[ -f "${POSTGRES_DATA_DIR}/postgresql.conf" ]]; then
        cp "${POSTGRES_DATA_DIR}/postgresql.conf" "${POSTGRES_DATA_DIR}/postgresql.conf.backup"
    fi

    # Configure postgresql.conf
    cat >> "${POSTGRES_DATA_DIR}/postgresql.conf" << EOF

#-------------------------------------------------------------------------------
# go2postgres Configuration
#-------------------------------------------------------------------------------
# Network
port = ${POSTGRES_PORT}
listen_addresses = 'localhost'

# Connections
max_connections = 200
superuser_reserved_connections = 3

# Memory (adjust based on available RAM)
shared_buffers = 256MB
effective_cache_size = 768MB
work_mem = 16MB
maintenance_work_mem = 128MB

# WAL
wal_level = replica
max_wal_size = 1GB
min_wal_size = 80MB

# Logging
logging_collector = on
log_directory = '${GO2POSTGRES_BASE}/data/logs'
log_filename = 'postgresql-%Y-%m-%d.log'
log_rotation_age = 1d
log_rotation_size = 100MB
log_min_duration_statement = 1000
log_line_prefix = '%m [%p] %u@%d '

# Statement timeout (default for all, can be overridden per-user)
statement_timeout = 60000

# Connection timeout
authentication_timeout = 60s
EOF

    # Configure pg_hba.conf for local and TCP connections
    cat > "${POSTGRES_DATA_DIR}/pg_hba.conf" << EOF
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# Local connections
local   all             all                                     peer

# IPv4 local connections (for go2postgres app)
host    all             all             127.0.0.1/32            scram-sha-256

# IPv6 local connections
host    all             all             ::1/128                 scram-sha-256

# Replication connections (disabled for now)
# local   replication     all                                     peer
# host    replication     all             127.0.0.1/32            scram-sha-256
EOF

    # Set ownership
    chown go2postgres:go2postgres "${POSTGRES_DATA_DIR}/postgresql.conf"
    chown go2postgres:go2postgres "${POSTGRES_DATA_DIR}/pg_hba.conf"

    log_success "PostgreSQL configured"
}

#-------------------------------------------------------------------------------
# Start PostgreSQL
#-------------------------------------------------------------------------------
start_postgresql() {
    log_info "Starting PostgreSQL..."

    # Use pre-detected pg_ctl command
    if [[ -z "$PG_CTL_CMD" ]]; then
        log_error "pg_ctl command not found. This should not happen."
        exit 1
    fi

    # Start PostgreSQL
    sudo -u go2postgres ${PG_CTL_CMD} \
        -D "${POSTGRES_DATA_DIR}" \
        -l "${GO2POSTGRES_BASE}/data/logs/postgresql-startup.log" \
        -o "-p ${POSTGRES_PORT}" \
        start

    # Wait for PostgreSQL to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    sleep 3

    # Verify it's running
    if sudo -u go2postgres ${PG_CTL_CMD} -D "${POSTGRES_DATA_DIR}" status > /dev/null 2>&1; then
        log_success "PostgreSQL started successfully on port ${POSTGRES_PORT}"
    else
        log_error "Failed to start PostgreSQL. Check logs at ${GO2POSTGRES_BASE}/data/logs/"
        exit 1
    fi
}

#-------------------------------------------------------------------------------
# Create Database Admin User
#-------------------------------------------------------------------------------
create_db_admin() {
    log_info "Creating database admin user..."

    # Generate password
    DB_ADMIN_PASSWORD=$(generate_password)

    # Use pre-detected psql command
    if [[ -z "$PSQL_CMD" ]]; then
        log_error "psql command not found. This should not happen."
        exit 1
    fi

    # Create the admin user with necessary privileges
    sudo -u go2postgres ${PSQL_CMD} -p ${POSTGRES_PORT} -d postgres << EOF
-- Create the go2postgres admin user
CREATE USER ${DB_ADMIN_USER} WITH 
    SUPERUSER 
    CREATEDB 
    CREATEROLE 
    LOGIN 
    PASSWORD '${DB_ADMIN_PASSWORD}';

-- Create a management database for go2postgres operations
CREATE DATABASE go2postgres_mgmt OWNER ${DB_ADMIN_USER};

-- Grant necessary permissions
ALTER USER ${DB_ADMIN_USER} SET statement_timeout = '0';

-- Create extension in template1 so all new databases have it
\c template1
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\echo 'Database admin user created successfully'
EOF

    log_success "Database admin user '${DB_ADMIN_USER}' created"
}

#-------------------------------------------------------------------------------
# Generate Environment File
#-------------------------------------------------------------------------------
generate_env_file() {
    log_info "Generating environment file..."

    # Generate secrets
    ENCRYPTION_KEY=$(generate_secret)
    JWT_SECRET=$(generate_secret)

    # Create .env file
    cat > "${GO2POSTGRES_BASE}/.env" << EOF
#===============================================================================
# go2postgres Environment Configuration
# Generated: $(date -Iseconds)
#===============================================================================

#-------------------------------------------------------------------------------
# Security Secrets (DO NOT SHARE!)
#-------------------------------------------------------------------------------
# 32-byte hex key for encrypting database credentials (AES-256-GCM)
ENCRYPTION_KEY=${ENCRYPTION_KEY}

# Secret for signing JWT tokens
JWT_SECRET=${JWT_SECRET}

#-------------------------------------------------------------------------------
# PostgreSQL Connection
#-------------------------------------------------------------------------------
POSTGRES_HOST=localhost
POSTGRES_PORT=${POSTGRES_PORT}
POSTGRES_ADMIN_USER=${DB_ADMIN_USER}
POSTGRES_ADMIN_PASSWORD=${DB_ADMIN_PASSWORD}
POSTGRES_SSLMODE=disable

#-------------------------------------------------------------------------------
# HTTP Server
#-------------------------------------------------------------------------------
HTTP_PORT=${HTTP_PORT}
# TLS_CERT_FILE=/opt/go2postgres/config/ssl/cert.pem
# TLS_KEY_FILE=/opt/go2postgres/config/ssl/key.pem

#-------------------------------------------------------------------------------
# Application Settings
#-------------------------------------------------------------------------------
DATA_DIR=/opt/go2postgres/data
LOG_LEVEL=info
LOG_FORMAT=json

#-------------------------------------------------------------------------------
# Feature Flags
#-------------------------------------------------------------------------------
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
SHUTDOWN_TIMEOUT=30s

#-------------------------------------------------------------------------------
# Default Tenant Limits
#-------------------------------------------------------------------------------
DEFAULT_CONN_LIMIT=10
DEFAULT_STMT_TIMEOUT=30000
MAX_DATABASES_PER_USER=10

#-------------------------------------------------------------------------------
# Bootstrap Admin
#-------------------------------------------------------------------------------
BOOTSTRAP_ADMIN_EMAIL=${ADMIN_EMAIL}
EOF

    # Secure the .env file
    chmod 600 "${GO2POSTGRES_BASE}/.env"
    chown go2postgres:go2postgres "${GO2POSTGRES_BASE}/.env"

    log_success "Environment file created at ${GO2POSTGRES_BASE}/.env"
}

#-------------------------------------------------------------------------------
# Create Systemd Service
#-------------------------------------------------------------------------------
create_systemd_services() {
    log_info "Creating systemd service files..."

    # PostgreSQL service for go2postgres
    cat > /etc/systemd/system/go2postgres-pg.service << EOF
[Unit]
Description=PostgreSQL for go2postgres
After=network.target

[Service]
Type=forking
User=go2postgres
Group=go2postgres

# PostgreSQL paths
Environment=PGDATA=${POSTGRES_DATA_DIR}
Environment=PGPORT=${POSTGRES_PORT}

ExecStart=${PG_CTL_CMD} -D ${POSTGRES_DATA_DIR} -l ${GO2POSTGRES_BASE}/data/logs/postgresql.log -o "-p ${POSTGRES_PORT}" start
ExecStop=${PG_CTL_CMD} -D ${POSTGRES_DATA_DIR} stop -m fast
ExecReload=${PG_CTL_CMD} -D ${POSTGRES_DATA_DIR} reload

TimeoutSec=120
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    # go2postgres application service
    cat > /etc/systemd/system/go2postgres.service << EOF
[Unit]
Description=go2postgres PostgreSQL Provision Manager
After=network.target go2postgres-pg.service
Requires=go2postgres-pg.service

[Service]
Type=simple
User=go2postgres
Group=go2postgres
WorkingDirectory=${GO2POSTGRES_BASE}
ExecStart=${GO2POSTGRES_BASE}/go2postgres-linux
EnvironmentFile=${GO2POSTGRES_BASE}/.env
Restart=always
RestartSec=5
TimeoutStopSec=30

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${GO2POSTGRES_BASE}/data

# Resource limits
LimitNOFILE=65535
MemoryMax=512M

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    systemctl daemon-reload

    log_success "Systemd services created"
    log_info "To enable auto-start: sudo systemctl enable go2postgres-pg go2postgres"
}

#-------------------------------------------------------------------------------
# Print Summary
#-------------------------------------------------------------------------------
print_summary() {
    echo ""
    echo "==============================================================================="
    echo -e "${GREEN}go2postgres Setup Complete!${NC}"
    echo "==============================================================================="
    echo ""
    echo "Configuration Summary:"
    echo "----------------------"
    echo "  Base Directory:     ${GO2POSTGRES_BASE}"
    echo "  PostgreSQL Port:    ${POSTGRES_PORT}"
    echo "  PostgreSQL Data:    ${POSTGRES_DATA_DIR}"
    echo "  HTTP API Port:      ${HTTP_PORT}"
    echo "  Admin Email:        ${ADMIN_EMAIL}"
    echo "  DB Admin User:      ${DB_ADMIN_USER}"
    echo ""
    echo "Environment File:     ${GO2POSTGRES_BASE}/.env"
    echo ""
    echo "Next Steps:"
    echo "-----------"
    echo "1. Review and adjust settings in ${GO2POSTGRES_BASE}/.env"
    echo ""
    echo "2. (Optional) Generate TLS certificates:"
    echo "   openssl req -x509 -nodes -days 365 -newkey rsa:2048 \\"
    echo "     -keyout ${GO2POSTGRES_BASE}/config/ssl/key.pem \\"
    echo "     -out ${GO2POSTGRES_BASE}/config/ssl/cert.pem"
    echo ""
    echo "3. Download and install the go2postgres binary:"
    echo "   wget -O ${GO2POSTGRES_BASE}/go2postgres-linux <release-url>"
    echo "   chmod +x ${GO2POSTGRES_BASE}/go2postgres-linux"
    echo "   chown go2postgres:go2postgres ${GO2POSTGRES_BASE}/go2postgres-linux"
    echo ""
    echo "4. Enable and start services:"
    echo "   sudo systemctl enable go2postgres-pg go2postgres"
    echo "   sudo systemctl start go2postgres"
    echo ""
    echo "5. Access the dashboard:"
    echo "   https://localhost:${HTTP_PORT}"
    echo ""
    echo "PostgreSQL Connection (for testing):"
    echo "-------------------------------------"
    echo "  psql -h localhost -p ${POSTGRES_PORT} -U ${DB_ADMIN_USER} -d go2postgres_mgmt"
    echo ""
    echo -e "${YELLOW}IMPORTANT: Save the credentials from ${GO2POSTGRES_BASE}/.env securely!${NC}"
    echo ""
}

#-------------------------------------------------------------------------------
# Main
#-------------------------------------------------------------------------------
main() {
    echo ""
    echo "==============================================================================="
    echo "go2postgres - Initial PostgreSQL and Project Setup"
    echo "==============================================================================="
    echo ""

    preflight_checks
    create_directories
    create_system_user
    init_postgresql
    configure_postgresql
    start_postgresql
    create_db_admin
    generate_env_file
    create_systemd_services
    print_summary
}

# Run main
main "$@"

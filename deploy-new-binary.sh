#!/usr/bin/env bash
# ============================================================
#  go2postgres – Deploy New Binary
#  Usage:  sudo bash deploy-new-binary.sh
# ============================================================
set -euo pipefail

# --- Configuration ------------------------------------------------
SERVICE_NAME="go2postgres"
INSTALL_DIR="/opt/go2postgres"
BINARY_NAME="go2postgres-linux"
NEW_BINARY="${BINARY_NAME}-new"
RUN_USER="go2postgres"
RUN_GROUP="go2postgres"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

# --- Colour helpers -----------------------------------------------
green()  { printf '\033[0;32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[0;33m%s\033[0m\n' "$*"; }
red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }

# --- Pre-flight checks --------------------------------------------
if [[ $EUID -ne 0 ]]; then
    red "ERROR: This script must be run as root (or via sudo)."
    exit 1
fi

if [[ ! -f "${INSTALL_DIR}/${NEW_BINARY}" ]]; then
    red "ERROR: New binary '${NEW_BINARY}' not found in ${INSTALL_DIR}."
    red "       Upload ${NEW_BINARY} to ${INSTALL_DIR} first."
    exit 1
fi

# --- Create service if not exists ---------------------------------
create_service() {
    green "Creating service user '${RUN_USER}'..."
    if ! id "${RUN_USER}" &>/dev/null; then
        useradd --system --no-create-home --shell /usr/sbin/nologin "${RUN_USER}"
    else
        yellow "User '${RUN_USER}' already exists."
    fi

    green "Creating directory structure..."
    mkdir -p "${INSTALL_DIR}/data"
    
    # Fix .env ownership if it exists
    if [[ -f "${INSTALL_DIR}/.env" ]]; then
        chown "${RUN_USER}:${RUN_GROUP}" "${INSTALL_DIR}/.env"
        chmod 600 "${INSTALL_DIR}/.env"
    fi
    
    chown -R "${RUN_USER}:${RUN_GROUP}" "${INSTALL_DIR}"

    green "Writing systemd unit → ${UNIT_FILE}"
    cat > "${UNIT_FILE}" <<EOF
[Unit]
Description=go2postgres PostgreSQL Database Provisioning Service
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
Group=${RUN_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Environment
EnvironmentFile=${INSTALL_DIR}/.env

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}/data
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    green "Reloading systemd and enabling ${SERVICE_NAME}..."
    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
}

# --- Check if service exists, create if not -----------------------
if [[ ! -f "${UNIT_FILE}" ]]; then
    yellow "Service '${SERVICE_NAME}' does not exist. Creating..."
    create_service
    
    # For first deployment, just copy the binary
    green ">>> First deployment - copying binary..."
    cp "${INSTALL_DIR}/${NEW_BINARY}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
    chown "${RUN_USER}:${RUN_GROUP}" "${INSTALL_DIR}/${BINARY_NAME}"
    
    echo ""
    green "======================================================"
    green "  Initial setup complete!"
    green "======================================================"
    echo ""
    yellow "Next steps:"
    yellow "  1. Edit ${INSTALL_DIR}/.env (set JWT_SECRET, ENCRYPTION_KEY, etc.)"
    yellow "  2. sudo systemctl start ${SERVICE_NAME}"
    yellow "  3. sudo systemctl status ${SERVICE_NAME}"
    yellow "  4. journalctl -u ${SERVICE_NAME} -f   (follow logs)"
    exit 0
fi

# --- Deploy new binary --------------------------------------------
green ">>> Stopping ${SERVICE_NAME}..."
systemctl stop "${SERVICE_NAME}" || true

if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
    green ">>> Backing up ${BINARY_NAME} → ${BINARY_NAME}.bak"
    cp "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.bak"
fi

green ">>> Deploying new binary..."
cp "${INSTALL_DIR}/${NEW_BINARY}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
chown "${RUN_USER}:${RUN_GROUP}" "${INSTALL_DIR}/${BINARY_NAME}"

green ">>> Starting ${SERVICE_NAME}..."
systemctl start "${SERVICE_NAME}"

green ">>> Deployment complete!"
echo ""
green ">>> Tailing logs (Ctrl+C to stop)..."
journalctl -u "${SERVICE_NAME}" -f

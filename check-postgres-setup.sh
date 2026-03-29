#!/bin/bash
#===============================================================================
# go2postgres - Pre-Setup Diagnostic Script
#===============================================================================
# This script checks:
# 1. If PostgreSQL is installed and where
# 2. Which ports are available (5432, 5438)
# 3. Current PostgreSQL instances running
#===============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo "==============================================================================="
echo "go2postgres - Pre-Setup Diagnostics"
echo "==============================================================================="
echo ""

#-------------------------------------------------------------------------------
# Check Port Availability
#-------------------------------------------------------------------------------
echo -e "${BLUE}[1/4] Checking port availability...${NC}"
echo ""

check_port() {
    local port=$1
    if ss -tuln 2>/dev/null | grep -q ":${port} "; then
        # Port is in use - find what's using it
        local process=$(ss -tulnp 2>/dev/null | grep ":${port} " | head -1)
        echo -e "  Port ${port}: ${RED}IN USE${NC}"
        echo "    $process"
        return 1
    else
        echo -e "  Port ${port}: ${GREEN}AVAILABLE${NC}"
        return 0
    fi
}

PORT_5432_FREE=false
PORT_5438_FREE=false

if check_port 5432; then
    PORT_5432_FREE=true
fi

if check_port 5438; then
    PORT_5438_FREE=true
fi

if check_port 15432; then
    :
fi

echo ""

#-------------------------------------------------------------------------------
# Check PostgreSQL Installation
#-------------------------------------------------------------------------------
echo -e "${BLUE}[2/4] Checking PostgreSQL installation...${NC}"
echo ""

POSTGRES_FOUND=false
POSTGRES_VERSION=""
POSTGRES_BIN_DIR=""

# Check common locations
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
)

for path in "${POSTGRES_PATHS[@]}"; do
    if [[ -x "${path}/initdb" ]] && [[ -x "${path}/pg_ctl" ]]; then
        POSTGRES_FOUND=true
        POSTGRES_BIN_DIR="$path"
        if [[ -x "${path}/postgres" ]]; then
            POSTGRES_VERSION=$("${path}/postgres" --version 2>/dev/null | head -1)
        fi
        echo -e "  ${GREEN}Found PostgreSQL at: ${path}${NC}"
        echo "  Version: ${POSTGRES_VERSION}"
        break
    fi
done

if [[ "$POSTGRES_FOUND" == "false" ]]; then
    echo -e "  ${RED}PostgreSQL binaries NOT found in standard locations${NC}"
    echo ""
    echo "  Checking if PostgreSQL package is installed..."
    
    if command -v dpkg &> /dev/null; then
        # Debian/Ubuntu
        INSTALLED=$(dpkg -l | grep -E "^ii.*postgresql-[0-9]+" | head -1 || true)
        if [[ -n "$INSTALLED" ]]; then
            echo -e "  ${GREEN}Package found:${NC} $INSTALLED"
        else
            echo -e "  ${YELLOW}No PostgreSQL server package installed${NC}"
        fi
    elif command -v rpm &> /dev/null; then
        # RHEL/CentOS
        INSTALLED=$(rpm -qa | grep -E "postgresql[0-9]+-server" | head -1 || true)
        if [[ -n "$INSTALLED" ]]; then
            echo -e "  ${GREEN}Package found:${NC} $INSTALLED"
        else
            echo -e "  ${YELLOW}No PostgreSQL server package installed${NC}"
        fi
    fi
fi

echo ""

#-------------------------------------------------------------------------------
# Check Running PostgreSQL Processes
#-------------------------------------------------------------------------------
echo -e "${BLUE}[3/4] Checking running PostgreSQL processes...${NC}"
echo ""

PG_PROCESSES=$(ps aux | grep -E "postgres:|postmaster" | grep -v grep || true)
if [[ -n "$PG_PROCESSES" ]]; then
    echo "  Running PostgreSQL processes:"
    echo "$PG_PROCESSES" | while read line; do
        echo "    $line"
    done
else
    echo -e "  ${YELLOW}No PostgreSQL processes found${NC}"
fi

# Check for Docker PostgreSQL
DOCKER_PG=$(docker ps 2>/dev/null | grep -i postgres || true)
if [[ -n "$DOCKER_PG" ]]; then
    echo ""
    echo "  Docker PostgreSQL containers:"
    echo "$DOCKER_PG" | while read line; do
        echo "    $line"
    done
fi

echo ""

#-------------------------------------------------------------------------------
# Recommendations
#-------------------------------------------------------------------------------
echo -e "${BLUE}[4/4] Recommendations...${NC}"
echo ""

if [[ "$POSTGRES_FOUND" == "true" ]]; then
    echo -e "  ${GREEN}✓ PostgreSQL is installed${NC}"
    echo "    Binary location: ${POSTGRES_BIN_DIR}"
    
    if [[ "$PORT_5432_FREE" == "true" ]]; then
        echo ""
        echo -e "  ${GREEN}✓ Recommended: Use port 5432${NC} (standard PostgreSQL port, currently free)"
        RECOMMENDED_PORT=5432
    elif [[ "$PORT_5438_FREE" == "true" ]]; then
        echo ""
        echo -e "  ${YELLOW}→ Recommended: Use port 5438${NC} (5432 is in use)"
        RECOMMENDED_PORT=5438
    else
        echo ""
        echo -e "  ${RED}✗ Both ports 5432 and 5438 are in use${NC}"
        echo "    Please free up a port or choose a different one"
        RECOMMENDED_PORT=""
    fi
else
    echo -e "  ${RED}✗ PostgreSQL server is NOT installed${NC}"
    echo ""
    echo "  To install PostgreSQL 16 on Ubuntu/Debian:"
    echo ""
    echo "    # Add PostgreSQL APT repository"
    echo "    sudo sh -c 'echo \"deb http://apt.postgresql.org/pub/repos/apt \$(lsb_release -cs)-pgdg main\" > /etc/apt/sources.list.d/pgdg.list'"
    echo "    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -"
    echo "    sudo apt update"
    echo ""
    echo "    # Install PostgreSQL 16"
    echo "    sudo apt install -y postgresql-16 postgresql-contrib-16"
    echo ""
    echo "  Note: This will install PostgreSQL but NOT start a default instance."
    echo "        The go2postgres setup script will create a dedicated instance."
fi

echo ""
echo "==============================================================================="
echo "Summary"
echo "==============================================================================="
echo ""
echo "  PostgreSQL installed:  $([ "$POSTGRES_FOUND" == "true" ] && echo -e "${GREEN}Yes${NC}" || echo -e "${RED}No${NC}")"
echo "  Port 5432 available:   $([ "$PORT_5432_FREE" == "true" ] && echo -e "${GREEN}Yes${NC}" || echo -e "${RED}No${NC}")"
echo "  Port 5438 available:   $([ "$PORT_5438_FREE" == "true" ] && echo -e "${GREEN}Yes${NC}" || echo -e "${RED}No${NC}")"

if [[ -n "$RECOMMENDED_PORT" ]]; then
    echo ""
    echo -e "  ${GREEN}Recommended port for go2postgres: ${RECOMMENDED_PORT}${NC}"
fi

echo ""

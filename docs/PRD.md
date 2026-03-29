# Project: go2postgres - PostgreSQL Provision Manager

> Generated: 2026-03-29
> Status: Draft
> Version: 1.0

---

## 1. Executive Summary

**go2postgres** is a lightweight, single-binary Go application that provides multi-tenant PostgreSQL instance provisioning and management on a single server. It offers a REST API, user dashboard, and admin panel for managing isolated PostgreSQL databases with automatic user/credential management. Built for developers who need Supabase-like provisioning without vendor lock-in or Kubernetes complexity.

**Why it matters:** Self-hosted teams, SMBs, and developers need simple PostgreSQL multi-tenancy without the overhead of managed services or complex IaC tools like Crossplane/Pulumi.

---

## 2. Problem Statement

### The Problem
Developers and small teams need to:
- Provision isolated PostgreSQL databases for multiple projects/users
- Manage credentials and access control securely
- Monitor database health and usage
- Avoid vendor lock-in from managed services (Supabase, RDS, Aiven)
- Keep infrastructure simple (single server, no Kubernetes)

### Current Solutions Fall Short
- **Managed services** (Supabase, Aiven): Vendor lock-in, expensive at scale, limited control
- **IaC tools** (Terraform, Pulumi, Crossplane): Over-engineered, require Kubernetes or complex state management
- **Manual provisioning**: Error-prone, no audit trail, credential management nightmares

### Target Audience
- Self-hosted SaaS providers
- Development agencies managing multiple client databases
- SMBs needing multi-tenant database isolation
- DevOps teams wanting simple PostgreSQL orchestration

---

## 3. Competitive Analysis

### Existing Solutions

| Competitor | Strengths | Weaknesses | Our Opportunity |
|------------|-----------|------------|------------------|
| **Supabase** | Managed service, auto-scaling, built-in auth, real-time, free tier | Vendor lock-in, expensive at scale, limited PostgreSQL config control | Self-hosted alternative with full control, no vendor lock-in, lower cost |
| **Pulumi** | Multi-language IaC, Kubernetes-native, state management, cloud-agnostic | Complex setup, requires Kubernetes, heavy resource usage (~500MB+ RAM) | Lightweight single-binary (<50MB), no Kubernetes dependency, <100MB RAM |
| **Crossplane** | Kubernetes CRDs, GitOps workflow, multi-cloud support | Kubernetes required, steep learning curve, over-engineered for single-server | Simple CLI + REST API, single-server optimized, no YAML hell |
| **Terraform** | Mature ecosystem, provider plugins, state management | HCL learning curve, state file management, not real-time, no dashboard | Real-time REST API + dashboard, no state files, Go-native simplicity |
| **pgAdmin** | Free, feature-rich PostgreSQL management | Single-user, no multi-tenant isolation, no API, manual credential management | Multi-tenant isolation, full REST API, automated provisioning |

### Differentiation Strategy

**Match (Table Stakes):**
- Automated PostgreSQL user creation
- Database provisioning with custom credentials
- Connection string management
- Basic monitoring (health checks, disk usage)

**Beat (Competitor Weaknesses):**
- ✅ **Single-binary deployment** (vs. Kubernetes/Pulumi complexity)
- ✅ **No vendor lock-in** (vs. Supabase/Aiven)
- ✅ **Real-time REST API** (vs. Terraform state files)
- ✅ **Built-in dashboard** (vs. CLI-only tools)
- ✅ **Multi-tenant isolation** (vs. pgAdmin single-user)

**Innovate (Unique Features):**
- 🚀 **Built-in connection pooling** (PgBouncer-style pool per database)
- 🚀 **SQLite metadata store** for credentials (no external DB dependency)
- 🚀 **Per-user database isolation** (separate PostgreSQL users with strict permissions)
- 🚀 **Go-native performance** (<100MB RAM, <10ms cold start)
- 🚀 **Prometheus metrics export** for observability
- 🚀 **Automatic maintenance** (VACUUM, ANALYZE scheduling)

---

## 4. Goals

### Primary Goals
1. **Provision PostgreSQL instances in <5 seconds** via REST API or dashboard
2. **Support 100+ isolated databases** on a single server with per-user credentials
3. **Provide full API parity** between dashboard and REST endpoints
4. **Zero external dependencies** beyond PostgreSQL itself

### Secondary Goals
- Built-in backup/restore functionality
- Usage analytics (connections, query counts, disk usage trends)
- Webhook notifications for provisioning events
- CLI tool for automation

### Success Metrics
| Metric | Target | Measurement |
|--------|--------|-------------|
| Provisioning time | <5 seconds | API response time |
| Memory usage | <100MB | `ps aux` RSS |
| Max concurrent databases | 100+ | Stress test |
| API uptime | 99.9% | Health check monitoring |
| Dashboard load time | <500ms | Lighthouse score |
| Database connection latency | <50ms | Connection pool metrics |
| Health check interval | 30 seconds | Background monitor |
| Graceful shutdown time | <30 seconds | All connections drained |

---

## 5. Non-Goals (Explicit Exclusions)

- ❌ **Multi-server clustering** (single-server only for MVP)
- ❌ **Kubernetes integration** (intentionally K8s-free)
- ❌ **Managed backups to cloud storage** (local backups only for MVP)
- ❌ **PostgreSQL extension management** (beyond standard extensions)
- ❌ **Query builder** or visual query interface
- ❌ **Multi-database support** (PostgreSQL only, not MySQL/MariaDB)

---

## 6. Target Users

| Persona | Description | Key Needs |
|---------|-------------|-----------|
| **Self-Hosted SaaS Founder** | Technical founder running multi-tenant SaaS on own infrastructure | Easy database isolation per customer, automated credential rotation, monitoring |
| **Development Agency** | Agency managing databases for multiple clients | Fast provisioning, client-specific credentials, usage tracking, low cost |
| **DevOps Engineer** | Small team DevOps managing infrastructure | Single-binary deployment, no Kubernetes, REST API for automation, simple monitoring |
| **Hobbyist Developer** | Developer running personal projects on VPS | Free/cheap solution, easy setup, dashboard for management, no vendor lock-in |

---

## 7. Functional Requirements

### Core Features

#### 1. **Multi-Tenant PostgreSQL Provisioning**
- **Description:** Create isolated PostgreSQL databases with unique users and passwords on a single PostgreSQL instance
- **Acceptance Criteria:**
  - ✅ Creates database with `{user_id}_{project_id}` naming convention
  - ✅ Generates secure random password (32 chars, alphanumeric + special chars)
  - ✅ Creates dedicated PostgreSQL user with CONNECT privilege only to own database
  - ✅ Stores credentials in SQLite metadata store (encrypted with AES-256-GCM)
  - ✅ Returns connection string in API response
  - ✅ Provisioning completes in <5 seconds
  - ✅ Applies resource limits (connection limit, statement timeout) per database
  - ✅ Revokes PUBLIC schema privileges for tenant isolation

#### 2. **User Management & Authentication**
- **Description:** Passwordless OTP authentication and role-based access control
- **Acceptance Criteria:**
  - ✅ Two roles: `admin` (full access) and `user` (own databases only)
  - ✅ OTP email authentication via Resend (6-digit codes, 10-minute expiry)
  - ✅ JWT-based session management (15-min access, 7-day refresh)
  - ✅ Admin can view/manage all databases and users
  - ✅ Regular users can only access their own databases
  - ✅ Auto-create user on first OTP verification

#### 3. **REST API (Full CRUD)**
- **Description:** Complete RESTful API for all operations
- **Acceptance Criteria:**
  - ✅ OpenAPI 3.0 specification
  - ✅ JWT authentication on all endpoints
  - ✅ Rate limiting (100 req/min per user)
  - ✅ Pagination for list endpoints
  - ✅ Consistent error responses (RFC 7807 problem details)

#### 4. **Dashboard UI (User)**
- **Description:** Modern React-based web interface for users to manage their databases
- **Acceptance Criteria:**
  - ✅ Responsive design with TailwindCSS
  - ✅ OTP sign-in/sign-up flow with auto-submit
  - ✅ List all user databases with status badges
  - ✅ Create/delete databases via modal UI
  - ✅ View connection strings (copy-to-clipboard)
  - ✅ Reveal passwords on demand (rate-limited, auto-hide after 30s)
  - ✅ Embedded in Go binary (single deployment)

#### 5. **Admin Panel**
- **Description:** Administrative interface for system management
- **Acceptance Criteria:**
  - ✅ Overview tab with stats (total users, instances, active counts)
  - ✅ Users tab with search, sort, suspend/activate actions
  - ✅ Instances tab with search, sort, suspend/activate actions
  - ✅ Tab-based navigation with responsive tables
  - ✅ Real-time data refresh

#### 6. **Credential Management**
- **Description:** Secure storage and rotation of PostgreSQL credentials
- **Acceptance Criteria:**
  - ✅ Credentials stored in SQLite with encryption (AES-256-GCM)
  - ✅ Master encryption key from environment variable
  - ✅ Password rotation endpoint (API + UI)
  - ✅ Audit log for credential access

#### 7. **Health Monitoring**
- **Description:** Real-time monitoring of provisioned databases
- **Acceptance Criteria:**
  - ✅ Health check endpoint per database (actual query execution test)
  - ✅ Disk usage monitoring (via PostgreSQL `pg_database_size`)
  - ✅ Connection count tracking (via `pg_stat_activity`)
  - ✅ Alerting via webhooks (configurable thresholds)
  - ✅ Background health monitor (30-second intervals)
  - ✅ Prometheus metrics endpoint (`/metrics`)

#### 8. **Connection Pooling**
- **Description:** Built-in connection pooler for efficient database connections
- **Acceptance Criteria:**
  - ✅ Pool per database with configurable min/max connections
  - ✅ Idle connection timeout (default: 5 minutes)
  - ✅ Connection health validation before use
  - ✅ Pool metrics exposed via Prometheus
  - ✅ Graceful connection draining on shutdown

#### 9. **Graceful Operations**
- **Description:** Reliable startup, shutdown, and error handling
- **Acceptance Criteria:**
  - ✅ Graceful shutdown with connection draining (30-second timeout)
  - ✅ Retry logic with exponential backoff for transient failures
  - ✅ Circuit breaker pattern for PostgreSQL operations
  - ✅ Structured JSON logging with correlation IDs
  - ✅ Startup health validation before accepting requests

### Feature Priority Matrix

| Feature | Priority | Complexity | Phase |
|---------|----------|------------|-------|
| Multi-tenant provisioning | P0 | Medium | MVP |
| REST API (CRUD) | P0 | Medium | MVP |
| User authentication (JWT) | P0 | Low | MVP |
| Dashboard UI | P0 | Medium | MVP |
| Credential encryption | P0 | Low | MVP |
| Connection pooling | P0 | Medium | MVP |
| Health monitoring | P1 | Low | MVP |
| Graceful shutdown | P1 | Low | MVP |
| Prometheus metrics | P1 | Low | MVP |
| Admin panel | P1 | Medium | MVP |
| Structured logging | P1 | Low | MVP |
| Backup/restore | P2 | High | V2 |
| Webhook notifications | P2 | Medium | V2 |
| Usage analytics | P2 | Medium | V2 |
| CLI tool | P3 | Low | V2 |
| Auto-maintenance (VACUUM) | P2 | Medium | V2 |

---

## 8. Technical Architecture

### Tech Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Backend** | Go 1.23+ | Single-binary, high performance, PostgreSQL drivers |
| **Web Framework** | Gin | Fast, minimal, excellent middleware support |
| **Frontend** | React 18 + TypeScript | Modern SPA, type-safe, component-based |
| **Build Tool** | Vite | Fast builds, hot reload, optimized output |
| **CSS Framework** | TailwindCSS | Utility-first styling, small bundle |
| **Routing** | React Router | Client-side SPA routing |
| **Frontend Embedding** | go:embed | Compile frontend into Go binary |
| **Metadata DB** | SQLite 3 | Embedded, no external dependency, single file |
| **User Database** | PostgreSQL 16+ | Single installation, multi-tenant |
| **Authentication** | JWT + OTP | Passwordless email authentication via Resend |
| **Email Service** | Resend | Transactional email for OTP delivery |
| **Encryption** | AEAD (AES-256-GCM) | Authenticated encryption for credentials |
| **Deployment** | Single binary + systemd | Simple, no containers required, ~28MB binary |

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  ┌──────────────┐        ┌──────────────┐                   │
│  │   React UI   │        │   REST API   │                   │
│  │  (Embedded)  │        │   Clients    │                   │
│  └──────┬───────┘        └──────┬───────┘                   │
└─────────┼──────────────────────┼─────────────────────────────┘
          │                      │
          └──────────┬───────────┘
                     │ HTTPS (port 8443)
┌────────────────────▼────────────────────────────────────────┐
│                    go2postgres Binary                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Web Server (Gin)                    │   │
│  │  ┌────────────────┐  ┌─────────────────┐             │   │
│  │  │  Static Files  │  │   REST API      │             │   │
│  │  │  (go:embed)    │  │   (JSON)        │             │   │
│  │  └────────────────┘  └─────────────────┘             │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  Service Layer                        │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │   │
│  │  │   Instance   │  │     User     │  │    OTP     │  │   │
│  │  │   Service    │  │    Service   │  │   Service  │  │   │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                 Data Access Layer                     │   │
│  │  ┌──────────────┐  ┌──────────────┐                  │   │
│  │  │   SQLite     │  │  PostgreSQL  │                  │   │
│  │  │  (Metadata)  │  │   (pgx)      │                  │   │
│  │  └──────────────┘  └──────────────┘                  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
          │                      │
          │                      │
┌─────────▼───────────┐  ┌──────▼────────────────────────────┐
│  SQLite File        │  │   Single PostgreSQL Installation  │
│  ./data/meta.db     │  │   (localhost:5438)                │
│  - users            │  │   - Multiple databases            │
│  - instances        │  │   - Dedicated user per database   │
│  - otp_codes        │  │   - Connection limits per tenant  │
└─────────────────────┘  └───────────────────────────────────┘
```

### Data Flow: Database Provisioning

```
User → REST API → Auth Middleware → DatabaseService → SQLite (check)
                                           ↓
                                    PostgreSQL Manager
                                           ↓
                              CREATE DATABASE {user_id}_{project_id}
                              CREATE USER {username} WITH PASSWORD
                              GRANT ALL PRIVILEGES
                                           ↓
                                    SQLite (store encrypted creds)
                                           ↓
                                    Return connection info (no password)
```

### API Design

#### Authentication Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/v1/auth/otp/send` | Send OTP code to email | No |
| POST | `/api/v1/auth/otp/verify` | Verify OTP and receive JWT | No |
| POST | `/api/v1/auth/refresh` | Refresh JWT token | Yes (refresh token) |
| POST | `/api/v1/auth/logout` | Invalidate token | Yes |

#### User Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/me` | Get current user profile | Yes |

#### Instance Management

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/instances` | List user's databases | Yes |
| POST | `/api/v1/instances` | Provision new database | Yes |
| GET | `/api/v1/instances/:id` | Get database details | Yes (owner/admin) |
| DELETE | `/api/v1/instances/:id` | Delete database + user | Yes (owner/admin) |
| GET | `/api/v1/instances/:id/password` | Reveal password (rate-limited) | Yes (owner/admin) |

#### Admin Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/admin/stats` | System statistics | Yes (admin) |
| GET | `/api/v1/admin/users` | List all users | Yes (admin) |
| GET | `/api/v1/admin/instances` | List all instances | Yes (admin) |
| PATCH | `/api/v1/admin/users/:id` | Update user (status, role) | Yes (admin) |
| PATCH | `/api/v1/admin/instances/:id` | Update instance (status) | Yes (admin) |

### Database Schema

#### SQLite (Metadata Store)

**Location:** `./data/meta.db`

```sql
-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,            -- Optional (OTP-based auth)
    role TEXT NOT NULL DEFAULT 'user',
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- OTP codes table
CREATE TABLE otp_codes (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- PostgreSQL instances table
CREATE TABLE instances (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    database_name TEXT UNIQUE NOT NULL,  -- {user_id}_{project_id}
    postgres_user TEXT UNIQUE NOT NULL,  -- Generated username
    postgres_password_encrypted TEXT NOT NULL,  -- AES-256-GCM encrypted
    postgres_password_nonce TEXT NOT NULL,  -- GCM nonce for decryption
    host TEXT DEFAULT 'localhost',
    port INTEGER DEFAULT 5432,     -- PostgreSQL port (shared by all databases)
    connection_limit INTEGER DEFAULT 10,  -- Max concurrent connections
    statement_timeout_ms INTEGER DEFAULT 30000,  -- Query timeout (30s default)
    extensions TEXT,               -- JSON array: ["pgcrypto", "uuid-ossp"]
    status TEXT DEFAULT 'active',  -- 'active' | 'suspended' | 'deleted'
    disk_usage_bytes INTEGER DEFAULT 0,
    connection_count INTEGER DEFAULT 0,
    last_health_check DATETIME,
    health_status TEXT DEFAULT 'unknown',  -- 'healthy' | 'unhealthy' | 'unknown'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, project_id)
);

-- Audit logs table
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    action TEXT NOT NULL,          -- 'database.created', 'user.deleted', etc.
    resource_type TEXT,            -- 'database', 'user', 'instance'
    resource_id TEXT,
    metadata TEXT,                 -- JSON blob with details
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- System settings table
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_instances_user_id ON instances(user_id);
CREATE INDEX idx_instances_status ON instances(status);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

### Security Model

#### Authentication
- **JWT-based authentication** (RS256 or HS256)
- **Access token:** 15-minute expiry
- **Refresh token:** 7-day expiry, stored in SQLite
- **Password hashing:** bcrypt (cost factor 12)
- **API endpoints:** All require JWT except `/auth/register` and `/auth/login`

#### Authorization
| Role | Permissions |
|------|-------------|
| **admin** | Full access to all resources, user management, system settings |
| **user** | CRUD on own databases only, read own profile |

#### Data Protection
1. **Credential Encryption:** AES-256-GCM with unique nonce per credential, master key from env var
2. **Connection String Security:** Passwords only revealed via explicit `/reveal-password` endpoint (rate-limited, audit-logged)
3. **Database Isolation:** Per-database PostgreSQL roles with CONNECT privilege only to own database
4. **Schema Isolation:** Revoke PUBLIC privileges, dedicated schema per tenant
5. **Resource Limits:** Connection limits and statement timeouts per database
6. **Audit Logging:** All credential access and provisioning logged with correlation IDs

### Reliability & Operations

#### Graceful Shutdown
- Signal handling (SIGTERM, SIGINT)
- Stop accepting new HTTP requests
- Drain active connections (30-second timeout)
- Close database pools cleanly
- Flush pending audit logs

#### Error Handling
- Retry with exponential backoff for transient PostgreSQL errors
- Circuit breaker pattern prevents cascade failures
- Structured error responses (RFC 7807 Problem Details)
- Correlation IDs for distributed tracing

#### Observability
- **Metrics:** Prometheus endpoint (`/metrics`) with:
  - Database count by status
  - Provisioning duration histograms
  - Active connections gauge
  - Health check status per database
- **Logging:** Structured JSON logs with:
  - Correlation IDs
  - User IDs
  - Request duration
  - Error stack traces
- **Health Checks:** 
  - `/health` for load balancer health
  - `/ready` for readiness probes
  - Background health monitor (30-second intervals)

#### Backup Strategy (MVP)
- Manual pg_dump via admin API or CLI
- Backup stored in `./data/backups/`
- Retention policy configurable (default: 7 days)

### Deployment Architecture

#### Single-Server Deployment

```
/opt/go2postgres/
├── go2postgres-linux          # Main binary (chmod +x)
├── data/
│   ├── meta.db                # SQLite metadata store
│   ├── backups/               # PostgreSQL dumps (optional)
│   └── logs/                  # Application logs
├── templates/                 # Templ-generated HTML (embedded)
├── static/                    # Static assets (embedded)
└── config/
    ├── config.yaml            # Optional config file
    └── ssl/                   # TLS certificates
        ├── cert.pem
        └── key.pem
```

#### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `ENCRYPTION_KEY` | Yes | 32-byte hex key for credential encryption | `openssl rand -hex 32` |
| `JWT_SECRET` | Yes | Secret for JWT signing | `openssl rand -hex 32` |
| `POSTGRES_HOST` | No | PostgreSQL server host (default: localhost) | `localhost` |
| `POSTGRES_PORT` | No | PostgreSQL server port (auto-detected: 5432 if free) | `5432` |
| `POSTGRES_ADMIN_USER` | Yes | PostgreSQL superuser | `go2postgres_admin` |
| `POSTGRES_ADMIN_PASSWORD` | Yes | PostgreSQL superuser password | (auto-generated) |
| `POSTGRES_SSLMODE` | No | PostgreSQL SSL mode (default: prefer) | `require` |
| `HTTP_PORT` | No | HTTP server port (default: 8443) | `8443` |
| `DATA_DIR` | No | Data directory (default: ./data) | `/opt/go2postgres/data` |
| `LOG_LEVEL` | No | Log level: debug, info, warn, error (default: info) | `info` |
| `LOG_FORMAT` | No | Log format: json, text (default: json) | `json` |
| `METRICS_ENABLED` | No | Enable Prometheus metrics (default: true) | `true` |
| `HEALTH_CHECK_INTERVAL` | No | Database health check interval (default: 30s) | `30s` |
| `SHUTDOWN_TIMEOUT` | No | Graceful shutdown timeout (default: 30s) | `30s` |
| `DEFAULT_CONN_LIMIT` | No | Default connection limit per database (default: 10) | `10` |
| `DEFAULT_STMT_TIMEOUT` | No | Default statement timeout in ms (default: 30000) | `30000` |

---

## 9. UI/UX Overview

### Key Screens

#### 1. Dashboard (User)
- List all user databases with status
- Quick actions: Create, View Details, Reset Password, Delete
- Usage stats: Disk usage, connection count

#### 2. Create Database Modal
- Project name input
- Extensions multi-select
- Initial disk quota (optional)

#### 3. Database Details Page
- Full connection details (host, port, database name, username)
- Connection string with copy button (password revealed separately)
- "Reveal Password" button (rate-limited, audit-logged)
- Usage stats with progress bars (disk, connections)
- Health status indicator (healthy/unhealthy/unknown)
- Actions: Test Connection, Reset Password, Suspend, Delete

#### 4. Admin Panel
- System health overview
- User management table
- Audit logs with filters
- Settings configuration

---

## 10. Development Phases

### Phase 1: MVP (4-6 weeks)
- User authentication (register, login, JWT)
- Database provisioning (create, delete, list)
- Basic SQLite metadata store with encryption
- PostgreSQL user/database creation with isolation
- REST API (endpoints above)
- Dashboard UI (Alpine.js + Templ)
- Credential encryption (AES-256-GCM)
- Connection limits and statement timeouts
- Graceful shutdown handling
- Prometheus metrics export
- Structured JSON logging
- Health checks (background + on-demand)
- Systemd deployment

### Phase 1.1: Hardening (2 weeks)
- Comprehensive test suite (unit + integration)
- Security audit (SQL injection, auth bypass, credential leaks)
- Load testing (100+ concurrent databases)
- API documentation (OpenAPI 3.0 spec)
- Backup/restore scripts (pg_dump wrapper)
- Runbook documentation

### Phase 2: V1.1 (4 weeks)
- Admin panel
- Backup/restore automation
- Webhook notifications
- Usage analytics
- CLI tool
- Email notifications

### Phase 2.1: Multi-Server Support (6 weeks)
- Multiple PostgreSQL servers
- Load balancing
- Centralized metadata store (PostgreSQL)
- Horizontal scaling

### Phase 3: V2 - Enterprise (8 weeks)
- LDAP/SSO integration
- Advanced RBAC
- Audit log export (SIEM)
- Point-in-time recovery
- Read replicas
- Connection pooling (PgBouncer)

---

## 11. Assumptions

1. Dedicated PostgreSQL instance for go2postgres (port 5438, data in `/opt/go2postgres/data/postgres/`)
2. PostgreSQL 14+ with superuser access required for automation
3. Trusted network environment (VPC, private subnet) for PostgreSQL connections
4. SQLite sufficient for metadata (tested up to 1000+ instances)
5. No Kubernetes dependency (intentional design choice for simplicity)
6. Manual backups for MVP (pg_dump scripts, automated backup in V2)
7. First admin created via CLI flag on first run (`--bootstrap-admin`)
8. Linux deployment target (amd64, arm64)
9. TLS certificates provided by user (Let's Encrypt recommended)
10. Configuration via environment file (`/opt/go2postgres/.env`)

---

## 12. Open Questions (Resolved)

| Question | Resolution |
|----------|------------|
| ~~Port Allocation~~ | **Resolved:** All databases share PostgreSQL port 5438 (dedicated instance). Multi-tenancy achieved via separate database users with CONNECT privileges. |
| Quota Management | Enforce via PostgreSQL: connection limits (default 10/db), statement timeouts (default 30s). Disk quotas P2 feature. |
| Multi-PostgreSQL | Out of scope for MVP. V2 may support multiple PostgreSQL servers. |

### New Open Questions
1. **Password Reveal Policy:** Should passwords be retrievable multiple times or one-time only?
2. **Extension Allow-List:** Should we restrict which extensions users can enable?
3. **Database Suspend/Resume:** Implement as connection blocking or actual database pause?

---

## 13. Quick Start Commands

```bash
# Clone repository and run setup script
git clone https://github.com/yourorg/go2postgres.git
cd go2postgres

# Run the initial setup script (creates PostgreSQL instance + directories + .env)
sudo ./initial-postgres-and-project-setup.sh

# Review generated configuration
cat /opt/go2postgres/.env

# Download go2postgres binary (when available)
sudo wget -O /opt/go2postgres/go2postgres-linux https://github.com/yourorg/go2postgres/releases/latest/download/go2postgres-linux
sudo chmod +x /opt/go2postgres/go2postgres-linux
sudo chown go2postgres:go2postgres /opt/go2postgres/go2postgres-linux

# Enable and start services
sudo systemctl enable go2postgres-pg go2postgres
sudo systemctl start go2postgres

# Open dashboard
xdg-open https://localhost:8443

# Test PostgreSQL connection
psql -h localhost -p 5438 -U go2postgres_admin -d go2postgres_mgmt

# Create first database via API
curl -X POST https://localhost:8443/api/v1/databases \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"project_id": "my-first-db"}'
```

---

**END OF PRD**

Next Steps:
1. Review and approve this PRD
2. Create GitHub repository
3. Set up project board
4. Start development (Phase 1 MVP)

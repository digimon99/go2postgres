# go2postgres Architecture

## Overview

go2postgres is a single-binary Go application that provides multi-tenant PostgreSQL provisioning with a REST API and embedded React dashboard.

## System Architecture

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

## Components

### 1. Web Server (Gin)
- HTTP router and middleware
- JWT authentication
- Rate limiting
- CORS handling
- Static file serving (embedded React app)
- SPA fallback routing

### 2. Frontend (React + TypeScript)
The frontend is built with modern web technologies and embedded into the Go binary:

| Technology | Purpose |
|------------|---------|
| React 18 | Component-based UI framework |
| TypeScript | Type-safe JavaScript |
| Vite | Fast build tool and dev server |
| TailwindCSS | Utility-first CSS framework |
| React Router | Client-side routing |
| Lucide React | Icon library |

**Build Process:**
1. `npm run build` in `web/` folder
2. Vite outputs to `internal/static/dist/`
3. Go's `//go:embed` directive embeds files into binary
4. Gin serves static files and falls back to `index.html` for SPA routes

### 3. Service Layer
- **InstanceService:** Provisioning, deletion, connection management, resource limits
- **UserService:** User registration, authentication, RBAC
- **OTPService:** OTP generation, verification, email delivery via Resend
- **HealthService:** Background health checks, metrics collection

### 4. Data Access Layer
- **SQLite Repository:** Metadata storage (users, instances, OTP codes)
- **PostgreSQL Manager:** Direct PostgreSQL administration with retry logic
- **Connection Pool:** pgx connection pool with health validation

## Data Flow

### Database Provisioning Flow

```
1. User submits POST /api/v1/databases
2. JWT validated, user_id extracted, correlation ID generated
3. Rate limit checked (100 req/min per user)
4. DatabaseService validates project_id format (alphanumeric + hyphens)
5. Check if {user_id}_{project_id} already exists in SQLite
6. Generate secure random password (32 chars, alphanumeric + special)
7. Execute on PostgreSQL (with retry + circuit breaker):
   a. CREATE DATABASE "{user_id}_{project_id}" WITH ENCODING 'UTF8'
   b. CREATE USER "{username}" WITH PASSWORD '{password}'
   c. ALTER DATABASE "{database}" OWNER TO "{username}"
   d. GRANT CONNECT ON DATABASE "{database}" TO "{username}"
   e. REVOKE ALL ON DATABASE "{database}" FROM PUBLIC
   f. ALTER USER "{username}" CONNECTION LIMIT 10
   g. ALTER USER "{username}" SET statement_timeout = '30s'
8. Connect to new database and enable extensions
9. Encrypt password with AES-256-GCM (unique nonce)
10. Store in SQLite instances table (transactional)
11. Audit log the creation
12. Return connection info (password available via /reveal-password)
```

### Database Deletion Flow

```
1. Validate ownership (user_id or admin)
2. Terminate all active connections to the database
3. Execute on PostgreSQL:
   a. DROP DATABASE "{database}" WITH (FORCE)
   b. DROP USER "{username}"
4. Soft-delete in SQLite (set deleted_at, status='deleted')
5. Audit log the deletion
```

## Security Architecture

### Authentication
- JWT tokens (RS256)
- 15-minute access tokens
- 7-day refresh tokens
- bcrypt password hashing (cost 12)

### Authorization
- Role-based access control (admin, user)
- Resource-level permissions
- Admin impersonation capability

### Data Protection
- AES-256-GCM for credential encryption with unique nonce per credential
- Master key from environment variable (ENCRYPTION_KEY)
- Passwords revealed only via rate-limited /reveal-password endpoint (3/hour)
- Per-database PostgreSQL users with CONNECT privilege only
- Connection limits and statement timeouts per tenant
- TLS for all HTTP traffic
- Audit logging for all sensitive operations with correlation IDs

## Reliability

### Graceful Shutdown
- SIGTERM/SIGINT signal handling
- Stop accepting new requests
- Drain existing connections (30-second timeout)
- Close database connections cleanly
- Flush audit logs

### Error Handling
- Retry with exponential backoff for transient PostgreSQL errors
- Circuit breaker pattern (fail fast when PostgreSQL unavailable)
- Structured error responses (RFC 7807 Problem Details)
- Correlation IDs for request tracing

### Health Monitoring
- Background health checker (30-second intervals)
- Per-database health status tracking
- Prometheus metrics export
- `/health` and `/ready` endpoints for load balancers

## Deployment

### File Structure
```
/opt/go2postgres/
├── go2postgres-linux          # Main binary
├── data/
│   ├── meta.db                # SQLite
│   ├── backups/
│   └── logs/
├── templates/                 # Embedded
├── static/                    # Embedded
└── config/
    └── ssl/
```

### Systemd Service
```ini
[Unit]
Description=go2postgres PostgreSQL Provision Manager
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=go2postgres
Group=go2postgres
WorkingDirectory=/opt/go2postgres
ExecStart=/opt/go2postgres/go2postgres-linux
ExecReload=/bin/kill -HUP $MAINPID
EnvironmentFile=/opt/go2postgres/config/env
Restart=always
RestartSec=5
TimeoutStopSec=30

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/go2postgres/data

# Resource limits
LimitNOFILE=65535
MemoryMax=512M

[Install]
WantedBy=multi-user.target
```

### Environment File (/opt/go2postgres/config/env)
```bash
ENCRYPTION_KEY=<32-byte-hex>
JWT_SECRET=<32-byte-hex>
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=<password>
POSTGRES_SSLMODE=prefer
HTTP_PORT=8443
LOG_LEVEL=info
LOG_FORMAT=json
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
SHUTDOWN_TIMEOUT=30s
```

## Tech Stack

- **Language:** Go 1.23+
- **Web Framework:** Gin
- **Templates:** Templ
- **Frontend:** Alpine.js + Tailwind CSS
- **Metadata DB:** SQLite (modernc.org/sqlite - pure Go)
- **PostgreSQL Driver:** pgx/v5 (with connection pooling)
- **Auth:** golang-jwt/jwt/v5
- **Encryption:** crypto/aes (AES-256-GCM)
- **Metrics:** prometheus/client_golang
- **Logging:** slog (structured logging, stdlib)
- **Validation:** go-playground/validator/v10

## Performance Considerations

### Connection Pooling
- Admin connection pool: 5-20 connections for provisioning operations
- Separate pools per tenant database (on-demand, lazy initialization)
- Idle connection timeout: 5 minutes
- Max connection lifetime: 1 hour

### Caching
- JWT validation caching (in-memory, 5-minute TTL)
- Database metadata caching (in-memory, 1-minute TTL)
- Health status caching (30-second background refresh)

### Resource Limits
- Statement timeout per tenant (default 30s, configurable)
- Connection limit per database (default 10, configurable)
- Rate limiting: 100 requests/min per user

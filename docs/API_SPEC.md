# go2postgres REST API Specification

## Base URL
```
https://localhost:8443/api/v1
```

## Authentication

All endpoints (except `/auth/register` and `/auth/login`) require JWT authentication.

**Header:**
```
Authorization: Bearer <access_token>
```

## Error Response Format

All errors follow RFC 7807 Problem Details:

```json
{
  "type": "https://go2postgres.dev/errors/unauthorized",
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or expired JWT token",
  "instance": "/api/v1/databases",
  "timestamp": "2026-03-29T06:45:00Z"
}
```

---

## Authentication Endpoints

### POST /auth/register

Register a new user account.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecureP@ssw0rd123",
  "full_name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "user_id": "user_abc123",
  "email": "user@example.com",
  "status": "pending_approval",
  "message": "Registration successful. Awaiting admin approval."
}
```

**Errors:**
- `400 Bad Request` - Invalid email format, weak password
- `409 Conflict` - Email already registered

---

### POST /auth/login

Authenticate and receive JWT tokens.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecureP@ssw0rd123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "user_id": "user_abc123",
    "email": "user@example.com",
    "role": "user",
    "full_name": "John Doe"
  }
}
```

**Errors:**
- `401 Unauthorized` - Invalid credentials
- `403 Forbidden` - Account not approved or disabled

---

### POST /auth/refresh

Refresh access token using refresh token.

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Errors:**
- `401 Unauthorized` - Invalid or expired refresh token

---

### POST /auth/logout

Invalidate current token.

**Response (200 OK):**
```json
{
  "message": "Successfully logged out"
}
```

---

## Database Management Endpoints

### GET /databases

List all databases for the authenticated user.

**Query Parameters:**
- `page` (integer, default: 1) - Page number
- `per_page` (integer, default: 20, max: 100) - Items per page
- `status` (string, optional) - Filter by status: `active`, `stopped`, `deleted`

**Response (200 OK):**
```json
{
  "data": [
    {
      "instance_id": "inst_xyz789",
      "project_id": "ecommerce-prod",
      "database_name": "user_abc123_ecommerce_prod",
      "username": "u_abc123_ecommerce",
      "host": "localhost",
      "port": 5438,
      "status": "active",
      "disk_usage_bytes": 134217728,
      "connection_count": 3,
      "extensions": ["pgcrypto", "uuid-ossp"],
      "created_at": "2026-03-29T06:45:00Z",
      "updated_at": "2026-03-29T06:45:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total_items": 1,
    "total_pages": 1
  }
}
```

---

### POST /databases

Provision a new PostgreSQL database.

**Request:**
```json
{
  "project_id": "ecommerce-prod",
  "extensions": ["pgcrypto", "uuid-ossp"]
}
```

**Response (201 Created):**
```json
{
  "instance_id": "inst_xyz789",
  "user_id": "user_abc123",
  "project_id": "ecommerce-prod",
  "database_name": "user_abc123_ecommerce_prod",
  "username": "u_abc123_ecommerce",
  "host": "localhost",
  "port": 5438,
  "connection_string": "postgresql://u_abc123_ecommerce@localhost:5438/user_abc123_ecommerce_prod?sslmode=prefer",
  "connection_limit": 10,
  "statement_timeout_ms": 30000,
  "status": "active",
  "created_at": "2026-03-29T06:45:00Z",
  "message": "Database provisioned successfully. Use /reveal-password to get the password once."
}
```

**Errors:**
- `400 Bad Request` - Invalid project_id format
- `409 Conflict` - Database already exists for this project
- `503 Service Unavailable` - PostgreSQL connection failed

---

### GET /databases/:id

Get details for a specific database.

**Response (200 OK):**
```json
{
  "instance_id": "inst_xyz789",
  "user_id": "user_abc123",
  "project_id": "ecommerce-prod",
  "database_name": "user_abc123_ecommerce_prod",
  "username": "u_abc123_ecommerce",
  "host": "localhost",
  "port": 5438,
  "connection_string": "postgresql://u_abc123_ecommerce@localhost:5438/user_abc123_ecommerce_prod?sslmode=prefer",
  "connection_limit": 10,
  "statement_timeout_ms": 30000,
  "status": "active",
  "disk_usage_bytes": 134217728,
  "connection_count": 3,
  "health_status": "healthy",
  "extensions": ["pgcrypto", "uuid-ossp"],
  "created_at": "2026-03-29T06:45:00Z",
  "updated_at": "2026-03-29T06:45:00Z"
}
```

**Errors:**
- `404 Not Found` - Database not found
- `403 Forbidden` - User does not own this database

---

### DELETE /databases/:id

Delete a database and its PostgreSQL user.

**Response (200 OK):**
```json
{
  "message": "Database deleted successfully",
  "instance_id": "inst_xyz789",
  "deleted_at": "2026-03-29T07:00:00Z"
}
```

**Errors:**
- `404 Not Found` - Database not found
- `403 Forbidden` - User does not own this database

---

### POST /databases/:id/reset-password

Rotate the PostgreSQL password for a database.

**Response (200 OK):**
```json
{
  "message": "Password reset successfully. New password stored securely.",
  "instance_id": "inst_xyz789",
  "reset_at": "2026-03-29T07:00:00Z"
}
```

**Note:** The new password is NOT returned. Use `/reveal-password` to retrieve it once.

---

### POST /databases/:id/reveal-password

Reveal the database password. **Rate-limited to 3 requests per hour per database.** Each reveal is audit-logged.

**Response (200 OK):**
```json
{
  "instance_id": "inst_xyz789",
  "username": "u_abc123_ecommerce",
  "password": "Xk9#mP2$nQ7@wL5!",
  "connection_string": "postgresql://u_abc123_ecommerce:Xk9%23mP2%24nQ7%40wL5%21@localhost:5438/user_abc123_ecommerce_prod?sslmode=prefer",
  "warning": "This password will not be shown again. Store it securely.",
  "revealed_at": "2026-03-29T07:00:00Z"
}
```

**Errors:**
- `429 Too Many Requests` - Rate limit exceeded (3/hour)
- `404 Not Found` - Database not found

---

### GET /databases/:id/health

Check health status of a database.

**Response (200 OK):**
```json
{
  "instance_id": "inst_xyz789",
  "status": "healthy",
  "response_time_ms": 12,
  "last_checked": "2026-03-29T07:00:00Z"
}
```

**Response (503 Service Unavailable):**
```json
{
  "instance_id": "inst_xyz789",
  "status": "unhealthy",
  "error": "Connection timeout",
  "last_checked": "2026-03-29T07:00:00Z"
}
```

---

### GET /databases/:id/stats

Get usage statistics for a database.

**Response (200 OK):**
```json
{
  "instance_id": "inst_xyz789",
  "disk_usage_bytes": 134217728,
  "disk_usage_formatted": "128 MB",
  "connection_count": 3,
  "active_queries": 1,
  "database_size_mb": 128,
  "table_count": 15,
  "index_count": 23,
  "last_stats_update": "2026-03-29T07:00:00Z"
}
```

---

## Admin Endpoints

All admin endpoints require `admin` role.

### GET /admin/system

Get system-wide health overview.

**Response (200 OK):**
```json
{
  "postgresql_status": "running",
  "postgresql_version": "16.2",
  "go2postgres_uptime_seconds": 259200,
  "total_databases": 47,
  "total_users": 23,
  "ports_available": 23,
  "ports_total": 68,
  "total_disk_usage_bytes": 13314392064,
  "total_disk_usage_formatted": "12.4 GB",
  "active_connections": 89
}
```

---

### GET /admin/audit-logs

View system audit logs.

**Query Parameters:**
- `page`, `per_page` - Pagination
- `user_id` - Filter by user
- `action` - Filter by action type
- `resource_type` - Filter by resource type
- `start_date`, `end_date` - Date range

**Response (200 OK):**
```json
{
  "data": [
    {
      "log_id": "log_abc123",
      "user_id": "user_abc123",
      "action": "database.created",
      "resource_type": "database",
      "resource_id": "inst_xyz789",
      "metadata": {
        "project_id": "ecommerce-prod",
        "connection_limit": 10
      },
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2026-03-29T06:45:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total_items": 156,
    "total_pages": 8
  }
}
```

---

## Observability Endpoints

### GET /metrics

Prometheus metrics endpoint (no auth required if enabled).

**Response (200 OK):**
```text
# HELP go2postgres_databases_total Total number of databases
# TYPE go2postgres_databases_total gauge
go2postgres_databases_total{status="active"} 47
go2postgres_databases_total{status="suspended"} 3

# HELP go2postgres_connections_active Active database connections
# TYPE go2postgres_connections_active gauge
go2postgres_connections_active 89

# HELP go2postgres_provisioning_duration_seconds Database provisioning duration
# TYPE go2postgres_provisioning_duration_seconds histogram
go2postgres_provisioning_duration_seconds_bucket{le="1"} 45
go2postgres_provisioning_duration_seconds_bucket{le="5"} 98

# HELP go2postgres_health_check_status Database health check status
# TYPE go2postgres_health_check_status gauge
go2postgres_health_check_status{database="user_abc123_ecommerce_prod"} 1
```

---

### GET /health

Application health check (no auth required).

**Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 259200,
  "checks": {
    "sqlite": "ok",
    "postgresql": "ok",
    "disk_space": "ok"
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "unhealthy",
  "version": "1.0.0",
  "uptime_seconds": 259200,
  "checks": {
    "sqlite": "ok",
    "postgresql": "failed: connection refused",
    "disk_space": "ok"
  }
}
```

---

### GET /ready

Kubernetes-style readiness probe (no auth required).

**Response (200 OK):**
```json
{"ready": true}
```

**Response (503 Service Unavailable):**
```json
{"ready": false, "reason": "postgresql not available"}
```

---

### GET /admin/settings

Get system settings.

**Response (200 OK):**
```json
{
  "postgresql_host": "localhost",
  "postgresql_port": 5438,
  "default_connection_limit": 10,
  "default_statement_timeout_ms": 30000,
  "default_extensions": ["pgcrypto", "uuid-ossp"],
  "max_databases_per_user": 10,
  "default_disk_quota_bytes": 1073741824,
  "jwt_expiry_seconds": 900,
  "refresh_token_expiry_seconds": 604800,
  "health_check_interval_seconds": 30,
  "metrics_enabled": true
}
```

---

### PUT /admin/settings

Update system settings.

**Request:**
```json
{
  "default_connection_limit": 15,
  "default_statement_timeout_ms": 60000,
  "default_extensions": ["pgcrypto", "uuid-ossp", "postgis"],
  "max_databases_per_user": 20
}
```

**Response (200 OK):**
```json
{
  "message": "Settings updated successfully",
  "updated_at": "2026-03-29T07:00:00Z"
}
```

---

### POST /admin/impersonate/:user_id

Impersonate a user (for support purposes).

**Response (200 OK):**
```json
{
  "message": "Now impersonating user_abc123",
  "impersonation_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900,
  "original_user_id": "user_admin001"
}
```

**Note:** Use the returned `impersonation_token` to access the user's dashboard. The impersonation session expires after 15 minutes.

---

### POST /admin/databases/:id/suspend

Suspend a database (block all connections).

**Response (200 OK):**
```json
{
  "message": "Database suspended successfully",
  "instance_id": "inst_xyz789",
  "status": "suspended",
  "suspended_at": "2026-03-29T07:00:00Z"
}
```

---

### POST /admin/databases/:id/resume

Resume a suspended database.

**Response (200 OK):**
```json
{
  "message": "Database resumed successfully",
  "instance_id": "inst_xyz789",
  "status": "active",
  "resumed_at": "2026-03-29T07:00:00Z"
}
```

---

### POST /admin/databases/:id/backup

Trigger a manual backup (pg_dump).

**Response (202 Accepted):**
```json
{
  "message": "Backup initiated",
  "instance_id": "inst_xyz789",
  "backup_id": "backup_abc123",
  "estimated_duration_seconds": 30
}
```

---

### GET /admin/databases/:id/backups

List backups for a database.

**Response (200 OK):**
```json
{
  "data": [
    {
      "backup_id": "backup_abc123",
      "instance_id": "inst_xyz789",
      "size_bytes": 134217728,
      "created_at": "2026-03-29T06:00:00Z",
      "status": "completed"
    }
  ]
}
```

---

## Rate Limiting

All endpoints are rate-limited:
- **Standard users:** 100 requests/minute
- **Admin users:** 500 requests/minute
- **Login attempts:** 5 requests/minute per IP

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1679999999
```

**Response (429 Too Many Requests):**
```json
{
  "type": "https://go2postgres.dev/errors/rate-limit-exceeded",
  "title": "Rate Limit Exceeded",
  "status": 429,
  "detail": "Too many requests. Please retry after 60 seconds.",
  "retry_after": 60
}
```

---

## OpenAPI Specification

Full OpenAPI 3.0 specification available at:
```
GET /api/v1/openapi.json
```

Swagger UI available at:
```
GET /api/v1/docs
```

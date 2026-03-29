# go2postgres

**PostgreSQL Provision Manager** - Lightweight, single-binary multi-tenant PostgreSQL provisioning with REST API and modern React dashboard.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8)
![PostgreSQL](https://img.shields.io/badge/postgresql-16+-336791)
![React](https://img.shields.io/badge/react-18+-61DAFB)

---

## 🚀 Quick Start

### Installation

```bash
# Download binary
cd /opt
wget https://github.com/digimon99/go2postgres/releases/latest/go2postgres-linux
chmod +x go2postgres-linux

# Generate encryption keys
ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)

# Create bootstrap admin
./go2postgres-linux --bootstrap-admin-email admin@example.com

# Start service
./go2postgres-linux
```

### Access Dashboard

Open `https://localhost:8443` in your browser. The React UI is embedded in the binary and served automatically.

### Authentication

The system uses **OTP email authentication** via [Resend](https://resend.com):
1. Enter your email on the sign-in page
2. Receive a 6-digit code via email
3. Enter the code to authenticate

No passwords to remember!

### Create First Database

```bash
# Get JWT token via OTP authentication
# (or use the dashboard UI)

# Create database via API
curl -X POST https://localhost:8443/api/v1/instances \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"db_name": "my_first_db"}'
```

---

## ✨ Features

- **Multi-Tenant Provisioning:** Isolated PostgreSQL databases with unique users and passwords
- **REST API:** Full CRUD operations with JWT authentication
- **Modern React Dashboard:** Beautiful UI built with React 18, TypeScript, TailwindCSS
- **OTP Authentication:** Passwordless email authentication via Resend
- **Admin Panel:** User management, system stats, instance monitoring
- **Secure Credentials:** AES-256-GCM encryption, passwords revealed only on demand
- **Single Binary:** Frontend embedded via `go:embed` - no separate deployments
- **Lightweight:** ~28MB binary, <100MB RAM usage, <10ms cold start

---

## 🏗 Architecture

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
│  - instances        │  │   - Multiple users                │
│  - otp_codes        │  │   - Tenant isolation              │
└─────────────────────┘  └───────────────────────────────────┘
```

---

## 📋 Documentation

- **[PRD](docs/PRD.md)** - Product Requirements Document
- **[Architecture](docs/ARCHITECTURE.md)** - System Architecture
- **[API Spec](docs/API_SPEC.md)** - REST API Specification

---

## 🔧 Configuration

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `ENCRYPTION_KEY` | Yes | 32-byte hex key for credential encryption | `openssl rand -hex 32` |
| `JWT_SECRET` | Yes | Secret for JWT signing | `openssl rand -hex 32` |
| `POSTGRES_ADMIN_USER` | Yes | PostgreSQL superuser | `go2postgres_admin` |
| `POSTGRES_ADMIN_PASSWORD` | Yes | PostgreSQL superuser password | (from initial setup) |
| `RESEND_API_KEY` | Yes | Resend API key for OTP emails | `re_xxxxx` |
| `FROM_EMAIL` | Yes | Sender email address | `noreply@yourdomain.com` |
| `PORT` | No | HTTP server port (default: 8443) | `8443` |
| `DATA_DIR` | No | Data directory (default: ./data) | `/opt/go2postgres/data` |
| `FRONTEND_URL` | No | Frontend URL for CORS (default: http://localhost:5173) | `https://yourdomain.com` |

---

## 🛠 Development

### Prerequisites

- Go 1.23+
- PostgreSQL 16+
- Node.js 18+ (for frontend build)

### Project Structure

```
go2postgres/
├── cmd/go2postgres/       # Main entry point
├── internal/
│   ├── api/               # HTTP handlers, middleware, server
│   ├── config/            # Configuration loading
│   ├── crypto/            # AES-256-GCM encryption
│   ├── database/          # SQLite repository
│   ├── logger/            # Structured logging
│   ├── postgres/          # PostgreSQL manager
│   ├── services/          # Business logic
│   └── static/            # Embedded frontend (go:embed)
├── pkg/
│   └── email/             # Resend email client
├── web/                   # React frontend source
│   ├── src/
│   │   ├── components/    # Reusable UI components
│   │   ├── contexts/      # React contexts (Auth)
│   │   ├── lib/           # API client
│   │   └── pages/         # Page components
│   ├── package.json
│   └── vite.config.ts
├── docs/                  # Documentation
├── build-binary.ps1       # Build script (Windows)
└── deploy-new-binary.sh   # Deployment script (Linux)
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/yourorg/go2postgres.git
cd go2postgres

# Install frontend dependencies
cd web && npm install && cd ..

# Build frontend (outputs to internal/static/dist/)
cd web && npm run build && cd ..

# Build Go binary (includes embedded frontend)
go build -o go2postgres ./cmd/go2postgres

# Or use the build script (builds frontend + Go binaries)
# PowerShell:
.\build-binary.ps1
```

### Run Development Server

```bash
# Terminal 1: Run Go backend
export ENCRYPTION_KEY=$(openssl rand -hex 32)
export JWT_SECRET=$(openssl rand -hex 32)
export RESEND_API_KEY=re_xxxxx
export FROM_EMAIL=test@example.com
go run ./cmd/go2postgres

# Terminal 2: Run React dev server (with hot reload)
cd web
npm run dev
# Frontend available at http://localhost:5173 (proxies API to :8443)
```

---

## 📊 API Endpoints

### Authentication (OTP)
- `POST /api/v1/auth/otp/send` - Send OTP to email
- `POST /api/v1/auth/otp/verify` - Verify OTP and get JWT
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/logout` - Logout

### User
- `GET /api/v1/me` - Get current user profile

### Instance Management
- `GET /api/v1/instances` - List user's databases
- `POST /api/v1/instances` - Create database
- `GET /api/v1/instances/:id` - Get database details
- `DELETE /api/v1/instances/:id` - Delete database
- `GET /api/v1/instances/:id/password` - Reveal password (rate limited)

### Admin
- `GET /api/v1/admin/stats` - System statistics
- `GET /api/v1/admin/users` - List all users
- `GET /api/v1/admin/instances` - List all instances
- `PATCH /api/v1/admin/users/:id` - Update user (suspend/activate)
- `PATCH /api/v1/admin/instances/:id` - Update instance (suspend/activate)
- `GET /api/v1/admin/settings` - Get settings
- `PUT /api/v1/admin/settings` - Update settings
- `POST /api/v1/admin/impersonate/:user_id` - Impersonate user

See [API Spec](docs/API_SPEC.md) for full details.

---

## � Frontend UI

The UI is built with:
- **React 18** - Modern component-based UI
- **TypeScript** - Type-safe development
- **Vite** - Fast build tooling
- **TailwindCSS** - Utility-first styling
- **Lucide React** - Beautiful icons

### Pages
| Route | Description |
|-------|-------------|
| `/` | Landing page with features overview |
| `/signin` | OTP email authentication |
| `/signup` | New user registration with OTP |
| `/dashboard` | User's database management panel |
| `/admin` | Admin panel (stats, users, instances) |

### Embedding
The frontend is embedded into the Go binary using `go:embed`. When you run `npm run build` in the `web/` folder, it outputs to `internal/static/dist/`, which gets compiled into the binary.

---

## 🔒 Security

- **OTP Authentication:** Passwordless email-based login via Resend
- **JWT Tokens:** 15-minute access tokens, 7-day refresh tokens  
- **Credential Encryption:** AES-256-GCM for PostgreSQL passwords
- **RBAC:** Admin and user roles with resource-level permissions
- **Rate Limiting:** 100 req/min per user, password reveal limited
- **CORS:** Configurable allowed origins

---

## 📦 Deployment

### Systemd Service

```ini
[Unit]
Description=go2postgres - PostgreSQL Provision Manager
After=network.target postgresql.service

[Service]
Type=simple
User=go2postgres
WorkingDirectory=/opt/go2postgres
ExecStart=/opt/go2postgres/go2postgres
EnvironmentFile=/opt/go2postgres/.env
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Build & Deploy

```powershell
# On Windows (development machine)
.\build-binary.ps1  # Builds frontend + Linux binary

# On Linux server
./deploy-new-binary.sh  # Stops service, replaces binary, starts service
```

---

## 🤝 Contributing

Contributions welcome! Please read our contributing guidelines first.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/)
- [Templ](https://templ.guide/)
- [Alpine.js](https://alpinejs.dev/)
- [Tailwind CSS](https://tailwindcss.com/)
- [pgx](https://github.com/jackc/pgx)

---

**Built with ❤️ using Go**

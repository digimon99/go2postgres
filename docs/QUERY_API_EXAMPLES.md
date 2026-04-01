# Query API Examples

Base URL: `https://db.nxdot.com/api/v1/query`

Replace `YOUR_API_KEY` with your actual API key (e.g., `g2p_...`).

---

## 1. Create Users Table

```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, age INT, created_at TIMESTAMP DEFAULT NOW())"
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 0 }
  ]
}
```

---

## 2. Insert One User

```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "INSERT INTO users (name, email, age) VALUES ('\''John Doe'\'', '\''john@example.com'\'', 30)"
  }'
```

**With parameterized query (recommended):**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "INSERT INTO users (name, email, age) VALUES ($1, $2, $3)",
    "params": ["John Doe", "john@example.com", 30]
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 1 }
  ]
}
```

---

## 3. Insert Multiple Users

**Option A: Multiple statements in transaction mode (atomic)**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "INSERT INTO users (name, email, age) VALUES ('\''Alice Smith'\'', '\''alice@example.com'\'', 25); INSERT INTO users (name, email, age) VALUES ('\''Bob Johnson'\'', '\''bob@example.com'\'', 35); INSERT INTO users (name, email, age) VALUES ('\''Carol White'\'', '\''carol@example.com'\'', 28);",
    "mode": "transaction"
  }'
```

**Option B: Single INSERT with multiple rows**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "INSERT INTO users (name, email, age) VALUES ('\''Alice Smith'\'', '\''alice@example.com'\'', 25), ('\''Bob Johnson'\'', '\''bob@example.com'\'', 35), ('\''Carol White'\'', '\''carol@example.com'\'', 28)"
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 3 }
  ]
}
```

---

## 4. Edit User (Update)

**Update by ID:**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "UPDATE users SET name = '\''John Smith'\'', age = 31 WHERE id = 1"
  }'
```

**Update by email:**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "UPDATE users SET age = 32 WHERE email = '\''john@example.com'\''"
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 1 }
  ]
}
```

---

## 5. Delete User

**Delete by ID:**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "DELETE FROM users WHERE id = 1"
  }'
```

**Delete by email:**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "DELETE FROM users WHERE email = '\''john@example.com'\''"
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 1 }
  ]
}
```

---

## 6. Drop Users Table

```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "DROP TABLE users"
  }'
```

**Response:**
```json
{
  "results": [
    { "rows_affected": 0 }
  ]
}
```

---

## Bonus: Query Users

**Select all users:**
```bash
curl -X POST https://db.nxdot.com/api/v1/query \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT * FROM users ORDER BY id"
  }'
```

**Response:**
```json
{
  "results": [
    {
      "columns": ["id", "name", "email", "age", "created_at"],
      "rows": [
        [1, "John Doe", "john@example.com", 30, "2026-04-01T10:00:00Z"],
        [2, "Alice Smith", "alice@example.com", 25, "2026-04-01T10:05:00Z"]
      ],
      "row_count": 2
    }
  ]
}
```

---

## Windows PowerShell Examples

For PowerShell, use double quotes and escape differently:

```powershell
# Create table
Invoke-RestMethod -Uri "https://db.nxdot.com/api/v1/query" `
  -Method POST `
  -Headers @{ "Authorization" = "Bearer YOUR_API_KEY"; "Content-Type" = "application/json" } `
  -Body '{"sql": "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100), email VARCHAR(255) UNIQUE)"}'

# Insert user
Invoke-RestMethod -Uri "https://db.nxdot.com/api/v1/query" `
  -Method POST `
  -Headers @{ "Authorization" = "Bearer YOUR_API_KEY"; "Content-Type" = "application/json" } `
  -Body '{"sql": "INSERT INTO users (name, email) VALUES (''John Doe'', ''john@example.com'')"}'

# Select users
Invoke-RestMethod -Uri "https://db.nxdot.com/api/v1/query" `
  -Method POST `
  -Headers @{ "Authorization" = "Bearer YOUR_API_KEY"; "Content-Type" = "application/json" } `
  -Body '{"sql": "SELECT * FROM users"}'
```

---

## Notes

- **Mode**: Default is `"transaction"` (all statements succeed or all fail). Use `"mode": "pipeline"` for independent statement execution.
- **Row Limit**: Results are capped at 1000 rows per statement.
- **Read-Only Keys**: API keys with `readonly` type can only execute SELECT, EXPLAIN, SHOW, and WITH queries.
- **Blocked Commands**: COPY, ALTER SYSTEM, DROP DATABASE, and other dangerous commands are blocked.

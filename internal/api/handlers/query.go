package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/digimon99/go2postgres/internal/api/middleware"
	"github.com/digimon99/go2postgres/internal/models"
	"github.com/digimon99/go2postgres/internal/postgres"
	"github.com/digimon99/go2postgres/internal/queryguard"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/pkg/logger"
)

// QueryHandler handles the /query endpoint.
type QueryHandler struct {
	svc     *services.Service
	poolMgr *postgres.PoolManager
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(svc *services.Service, poolMgr *postgres.PoolManager) *QueryHandler {
	return &QueryHandler{svc: svc, poolMgr: poolMgr}
}

// QueryRequest is the JSON body for POST /query.
type QueryRequest struct {
	SQL    string        `json:"sql" binding:"required"`
	Params []interface{} `json:"params"`
	Mode   string        `json:"mode"` // "transaction" (default) or "pipeline"
}

// QueryResponse is the response for POST /query.
type QueryResponse struct {
	Results []StatementResult `json:"results"`
}

// StatementResult holds the result of a single SQL statement.
type StatementResult struct {
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	RowCount     int             `json:"row_count"`
	RowsAffected int64           `json:"rows_affected,omitempty"`
	Error        string          `json:"error,omitempty"`
}

const (
	maxRows         = 1000
	maxResponseSize = 10 * 1024 * 1024 // 10 MB estimate
)

// HandleQuery executes SQL statements against the user's database.
func (h *QueryHandler) HandleQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// Get API key and instance from context (set by middleware)
	keyRec, _ := c.Get(middleware.ContextAPIKey)
	instRec, _ := c.Get(middleware.ContextAPIKeyInst)
	apiKey := keyRec.(*models.APIKey)
	inst := instRec.(*models.Instance)

	// Split SQL into statements
	stmts := queryguard.SplitStatements(req.SQL)
	if len(stmts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no sql statements provided"})
		return
	}

	// Check blocklist
	if idx, err := queryguard.CheckBlocked(stmts); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "statement_index": idx})
		return
	}

	// For readonly keys, ensure only SELECT/EXPLAIN/SHOW
	if apiKey.KeyType == models.APIKeyTypeReadOnly {
		if idx, err := queryguard.CheckReadOnly(stmts); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "statement_index": idx})
			return
		}
	}

	// Build DSN and get pool
	dsn, err := h.svc.BuildInstanceDSN(inst)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to build dsn", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	pool, err := h.poolMgr.GetPool(c.Request.Context(), inst.ID, dsn, int32(inst.ConnectionLimit), inst.StatementTimeoutMs)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to get pool", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database connection unavailable"})
		return
	}

	// Execute
	mode := req.Mode
	if mode == "" {
		mode = "transaction"
	}

	var results []StatementResult
	switch mode {
	case "transaction":
		results, err = h.execTransaction(c.Request.Context(), pool, stmts, req.Params)
	case "pipeline":
		results, err = h.execPipeline(c.Request.Context(), pool, stmts, req.Params)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mode, must be 'transaction' or 'pipeline'"})
		return
	}

	if err != nil {
		logger.ErrorContext(c.Request.Context(), "query execution error", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizeError(err)})
		return
	}

	c.JSON(http.StatusOK, QueryResponse{Results: results})
}

// execTransaction runs all statements in a single transaction; rolls back on any error.
func (h *QueryHandler) execTransaction(ctx context.Context, pool *pgxpool.Pool, stmts []string, params []interface{}) ([]StatementResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	results := make([]StatementResult, 0, len(stmts))
	for _, stmt := range stmts {
		res, err := h.execSingleStatement(ctx, tx, stmt, params)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return results, nil
}

// execPipeline runs each statement independently; collects per-statement errors.
func (h *QueryHandler) execPipeline(ctx context.Context, pool *pgxpool.Pool, stmts []string, params []interface{}) ([]StatementResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	results := make([]StatementResult, 0, len(stmts))
	for _, stmt := range stmts {
		res, err := h.execSingleStatement(ctx, conn, stmt, params)
		if err != nil {
			res = StatementResult{Error: sanitizeError(err)}
		}
		results = append(results, res)
	}
	return results, nil
}

// execSingleStatement executes one SQL statement against a queryable (tx or conn).
func (h *QueryHandler) execSingleStatement(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}, stmt string, params []interface{}) (StatementResult, error) {
	// Determine if this looks like a SELECT/query statement
	trimmed := strings.TrimSpace(strings.ToUpper(stmt))
	isQuery := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "WITH") ||
		strings.HasPrefix(trimmed, "TABLE") ||
		strings.HasPrefix(trimmed, "SHOW") ||
		strings.HasPrefix(trimmed, "EXPLAIN")

	if !isQuery {
		// DDL/DML: use Exec directly
		tag, err := q.Exec(ctx, stmt, params...)
		if err != nil {
			return StatementResult{}, err
		}
		return StatementResult{RowsAffected: tag.RowsAffected()}, nil
	}

	// SELECT-like: use Query
	rows, err := q.Query(ctx, stmt, params...)
	if err != nil {
		return StatementResult{}, err
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = string(f.Name)
	}

	var data [][]interface{}
	rowCount := 0
	for rows.Next() {
		if rowCount >= maxRows {
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return StatementResult{}, err
		}
		// Convert to JSON-safe values
		safeVals := make([]interface{}, len(vals))
		for i, v := range vals {
			safeVals[i] = toJSONSafe(v)
		}
		data = append(data, safeVals)
		rowCount++
	}
	if err := rows.Err(); err != nil {
		return StatementResult{}, err
	}

	return StatementResult{
		Columns:  columns,
		Rows:     data,
		RowCount: rowCount,
	}, nil
}

// toJSONSafe converts pgx types to JSON-friendly values.
func toJSONSafe(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339Nano)
	default:
		return val
	}
}

// sanitizeError removes internal details from error messages.
func sanitizeError(err error) string {
	s := err.Error()
	// Remove connection strings, passwords, hostnames
	// For now just truncate long errors
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// HandleDashboardQuery executes SQL for authenticated dashboard users (JWT auth).
// This allows the web dashboard SQL editor to execute queries without API keys.
func (h *QueryHandler) HandleDashboardQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	// Get user from JWT context
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	// Get instance and verify ownership
	inst, err := h.svc.GetInstance(c.Request.Context(), userID, instanceID)
	if err != nil {
		switch err {
		case services.ErrInstanceNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get instance"})
		}
		return
	}

	// Split SQL into statements
	stmts := queryguard.SplitStatements(req.SQL)
	if len(stmts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no sql statements provided"})
		return
	}

	// Check blocklist (dashboard users get full access but still no dangerous commands)
	if idx, err := queryguard.CheckBlocked(stmts); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error(), "statement_index": idx})
		return
	}

	// Build DSN and get pool
	dsn, err := h.svc.BuildInstanceDSN(inst)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to build dsn", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	pool, err := h.poolMgr.GetPool(c.Request.Context(), inst.ID, dsn, int32(inst.ConnectionLimit), inst.StatementTimeoutMs)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to get pool", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database connection unavailable"})
		return
	}

	// Execute
	mode := req.Mode
	if mode == "" {
		mode = "transaction"
	}

	start := time.Now()
	var results []StatementResult
	switch mode {
	case "transaction":
		results, err = h.execTransaction(c.Request.Context(), pool, stmts, req.Params)
	case "pipeline":
		results, err = h.execPipeline(c.Request.Context(), pool, stmts, req.Params)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mode, must be 'transaction' or 'pipeline'"})
		return
	}
	elapsed := time.Since(start)

	if err != nil {
		logger.ErrorContext(c.Request.Context(), "query execution error", "error", err, "instance_id", inst.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizeError(err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     results,
		"elapsed_ms":  elapsed.Milliseconds(),
	})
}

// Package handlers provides HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/digimon99/go2postgres/internal/api/middleware"
	"github.com/digimon99/go2postgres/internal/postgres"
	"github.com/digimon99/go2postgres/internal/services"
	"github.com/digimon99/go2postgres/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TableHandler handles table editor endpoints.
type TableHandler struct {
	svc     *services.Service
	poolMgr *postgres.PoolManager
}

// NewTableHandler creates a new TableHandler.
func NewTableHandler(svc *services.Service, poolMgr *postgres.PoolManager) *TableHandler {
	return &TableHandler{svc: svc, poolMgr: poolMgr}
}

// TableInfo represents table metadata.
type TableInfo struct {
	TableName string `json:"table_name"`
	TableType string `json:"table_type"`
	RowCount  *int64 `json:"row_count,omitempty"`
}

// ColumnInfo represents column metadata.
type ColumnInfo struct {
	ColumnName    string  `json:"column_name"`
	DataType      string  `json:"data_type"`
	IsNullable    bool    `json:"is_nullable"`
	ColumnDefault *string `json:"column_default"`
	IsPrimary     bool    `json:"is_primary"`
	IsUnique      bool    `json:"is_unique"`
	IsArray       bool    `json:"is_array"`
}

// getPool gets a database pool for the instance.
func (h *TableHandler) getPool(ctx context.Context, userID, instanceID string) (*pgxpool.Pool, error) {
	inst, err := h.svc.GetInstance(ctx, userID, instanceID)
	if err != nil {
		return nil, err
	}

	dsn, err := h.svc.BuildInstanceDSN(inst)
	if err != nil {
		return nil, err
	}

	pool, err := h.poolMgr.GetPool(ctx, inst.ID, dsn, int32(inst.ConnectionLimit), inst.StatementTimeoutMs)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// execQuery executes a query and returns rows as maps.
func (h *TableHandler) execQuery(ctx context.Context, pool *pgxpool.Pool, sql string, args []interface{}) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	cols := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[string(col.Name)] = values[i]
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// execExec executes a non-query SQL statement.
func (h *TableHandler) execExec(ctx context.Context, pool *pgxpool.Pool, sql string, args []interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, sql, args...)
	return err
}

// ListTables returns all tables in the database.
func (h *TableHandler) ListTables(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		logger.ErrorContext(c.Request.Context(), "failed to get pool", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Query information_schema for tables
	sql := `
		SELECT table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`

	rows, err := h.execQuery(c.Request.Context(), pool, sql, nil)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to list tables", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tables"})
		return
	}

	var tables []TableInfo
	for _, row := range rows {
		t := TableInfo{
			TableName: fmt.Sprintf("%v", row["table_name"]),
			TableType: fmt.Sprintf("%v", row["table_type"]),
		}
		tables = append(tables, t)
	}

	c.JSON(http.StatusOK, gin.H{"tables": tables})
}

// GetTableSchema returns the column schema for a table.
func (h *TableHandler) GetTableSchema(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	// Validate table name
	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Query column info
	sql := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			COALESCE(tc.constraint_type = 'PRIMARY KEY', false) as is_primary,
			COALESCE(tc.constraint_type = 'UNIQUE', false) OR EXISTS(
				SELECT 1 FROM pg_indexes 
				WHERE tablename = c.table_name 
				AND indexdef LIKE '%UNIQUE%' 
				AND indexdef LIKE '%' || c.column_name || '%'
			) as is_unique,
			c.data_type LIKE '%[]' OR c.data_type = 'ARRAY' as is_array
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu 
			ON c.table_name = kcu.table_name 
			AND c.column_name = kcu.column_name
			AND c.table_schema = kcu.table_schema
		LEFT JOIN information_schema.table_constraints tc 
			ON kcu.constraint_name = tc.constraint_name
			AND tc.table_schema = c.table_schema
		WHERE c.table_schema = 'public' AND c.table_name = $1
		ORDER BY c.ordinal_position
	`

	rows, err := h.execQuery(c.Request.Context(), pool, sql, []interface{}{tableName})
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to get schema", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get schema"})
		return
	}

	var columns []ColumnInfo
	for _, row := range rows {
		col := ColumnInfo{
			ColumnName: fmt.Sprintf("%v", row["column_name"]),
			DataType:   fmt.Sprintf("%v", row["data_type"]),
			IsNullable: row["is_nullable"] == true,
			IsPrimary:  row["is_primary"] == true,
			IsUnique:   row["is_unique"] == true,
			IsArray:    row["is_array"] == true,
		}
		if def, ok := row["column_default"]; ok && def != nil {
			defStr := fmt.Sprintf("%v", def)
			col.ColumnDefault = &defStr
		}
		columns = append(columns, col)
	}

	c.JSON(http.StatusOK, gin.H{"columns": columns})
}

// GetTableRows returns paginated rows from a table.
func (h *TableHandler) GetTableRows(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	// Parse query params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	sortCol := c.Query("sort")
	sortOrder := c.DefaultQuery("order", "asc")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Get total count
	countSQL := fmt.Sprintf(`SELECT COUNT(*) as count FROM %s`, quoteIdentifier(tableName))
	countRows, err := h.execQuery(c.Request.Context(), pool, countSQL, nil)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to get count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get row count"})
		return
	}
	
	var total int64
	if len(countRows) > 0 {
		if v, ok := countRows[0]["count"].(int64); ok {
			total = v
		} else if v, ok := countRows[0]["count"].(float64); ok {
			total = int64(v)
		}
	}

	// Build select query
	offset := (page - 1) * pageSize
	sql := fmt.Sprintf(`SELECT * FROM %s`, quoteIdentifier(tableName))
	
	if sortCol != "" && isValidIdentifier(sortCol) {
		sql += fmt.Sprintf(` ORDER BY %s %s`, quoteIdentifier(sortCol), strings.ToUpper(sortOrder))
	}
	
	sql += fmt.Sprintf(` LIMIT %d OFFSET %d`, pageSize, offset)

	rows, err := h.execQuery(c.Request.Context(), pool, sql, nil)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to get rows", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rows"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":  rows,
		"total": total,
	})
}

// CreateTableRequest represents a create table request.
type CreateTableRequest struct {
	TableName   string         `json:"table_name" binding:"required"`
	Description string         `json:"description"`
	Columns     []CreateColumn `json:"columns" binding:"required,min=1"`
}

// CreateColumn represents a column definition.
type CreateColumn struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	DefaultValue string `json:"default_value"`
	IsPrimary    bool   `json:"is_primary"`
	IsNullable   bool   `json:"is_nullable"`
	IsUnique     bool   `json:"is_unique"`
	IsArray      bool   `json:"is_array"`
}

// CreateTable creates a new table.
func (h *TableHandler) CreateTable(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")

	var req CreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if !isValidIdentifier(req.TableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Build CREATE TABLE SQL
	var colDefs []string
	var primaryKey string

	for _, col := range req.Columns {
		if !isValidIdentifier(col.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid column name: %s", col.Name)})
			return
		}

		def := fmt.Sprintf("%s %s", quoteIdentifier(col.Name), col.Type)
		
		if !col.IsNullable && !col.IsPrimary {
			def += " NOT NULL"
		}
		
		if col.IsUnique && !col.IsPrimary {
			def += " UNIQUE"
		}
		
		if col.DefaultValue != "" {
			def += fmt.Sprintf(" DEFAULT %s", col.DefaultValue)
		}

		colDefs = append(colDefs, def)

		if col.IsPrimary {
			primaryKey = col.Name
		}
	}

	if primaryKey != "" {
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", quoteIdentifier(primaryKey)))
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", 
		quoteIdentifier(req.TableName),
		strings.Join(colDefs, ",\n  "))

	err = h.execExec(c.Request.Context(), pool, sql, nil)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to create table", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create table: %v", err)})
		return
	}

	// Add comment if description provided
	if req.Description != "" {
		commentSQL := fmt.Sprintf("COMMENT ON TABLE %s IS %s",
			quoteIdentifier(req.TableName),
			quoteLiteral(req.Description))
		h.execExec(c.Request.Context(), pool, commentSQL, nil) // Ignore error
	}

	c.JSON(http.StatusCreated, gin.H{"message": "table created successfully"})
}

// UpdateTableRequest represents an update table request.
type UpdateTableRequest struct {
	NewName     string         `json:"new_name"`
	Description string         `json:"description"`
	Columns     []CreateColumn `json:"columns"`
}

// UpdateTable updates a table's schema.
func (h *TableHandler) UpdateTable(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	var req UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Rename table if needed
	if req.NewName != "" && req.NewName != tableName {
		if !isValidIdentifier(req.NewName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid new table name"})
			return
		}
		sql := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", 
			quoteIdentifier(tableName), quoteIdentifier(req.NewName))
		err = h.execExec(c.Request.Context(), pool, sql, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to rename table: %v", err)})
			return
		}
		tableName = req.NewName
	}

	// Update description
	if req.Description != "" {
		sql := fmt.Sprintf("COMMENT ON TABLE %s IS %s",
			quoteIdentifier(tableName), quoteLiteral(req.Description))
		h.execExec(c.Request.Context(), pool, sql, nil)
	}

	// Note: Full schema alteration (adding/removing columns, changing types) is complex
	// For now, we just support rename and description. Full ALTER TABLE support
	// would require comparing old vs new columns and generating appropriate ALTER statements.

	c.JSON(http.StatusOK, gin.H{"message": "table updated successfully"})
}

// DropTable drops a table.
func (h *TableHandler) DropTable(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	sql := fmt.Sprintf("DROP TABLE %s", quoteIdentifier(tableName))
	err = h.execExec(c.Request.Context(), pool, sql, nil)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to drop table", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to drop table: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "table dropped successfully"})
}

// InsertRow inserts a new row.
func (h *TableHandler) InsertRow(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Build INSERT statement
	var cols, placeholders []string
	var values []interface{}
	i := 1
	for col, val := range data {
		if !isValidIdentifier(col) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid column name: %s", col)})
			return
		}
		cols = append(cols, quoteIdentifier(col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(tableName),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	err = h.execExec(c.Request.Context(), pool, sql, values)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to insert row", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert row: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "row inserted successfully"})
}

// UpdateRow updates a row by primary key.
func (h *TableHandler) UpdateRow(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pkColumn, _ := data["_pk_column"].(string)
	pkValue, _ := data["_pk_value"].(interface{})
	delete(data, "_pk_column")
	delete(data, "_pk_value")

	if pkColumn == "" || !isValidIdentifier(pkColumn) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "primary key column required"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	// Build UPDATE statement
	var setClauses []string
	var values []interface{}
	i := 1
	for col, val := range data {
		if !isValidIdentifier(col) {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), i))
		values = append(values, val)
		i++
	}

	values = append(values, pkValue)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		quoteIdentifier(pkColumn),
		i)

	err = h.execExec(c.Request.Context(), pool, sql, values)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to update row", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update row: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "row updated successfully"})
}

// DeleteRow deletes a row by primary key.
func (h *TableHandler) DeleteRow(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	instanceID := c.Param("id")
	tableName := c.Param("table")

	if !isValidIdentifier(tableName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table name"})
		return
	}

	var req struct {
		PKColumn string      `json:"pk_column"`
		PKValue  interface{} `json:"pk_value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.PKColumn == "" || !isValidIdentifier(req.PKColumn) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "primary key column required"})
		return
	}

	pool, err := h.getPool(c.Request.Context(), userID, instanceID)
	if err != nil {
		if err == services.ErrInstanceNotFound || err == services.ErrInstanceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if err == services.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to database"})
		return
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(tableName),
		quoteIdentifier(req.PKColumn))

	err = h.execExec(c.Request.Context(), pool, sql, []interface{}{req.PKValue})
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "failed to delete row", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete row: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "row deleted successfully"})
}

// Helper functions

func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	// Allow alphanumeric and underscore, must not start with number
	for i, r := range name {
		if i == 0 && (r >= '0' && r <= '9') {
			return false
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteLiteral(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

// Unused but may be needed later for filters
func parseFilters(filtersJSON string) ([]map[string]string, error) {
	if filtersJSON == "" {
		return nil, nil
	}
	var filters []map[string]string
	if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
		return nil, err
	}
	return filters, nil
}

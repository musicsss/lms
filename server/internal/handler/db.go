package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DBHandler struct {
	db *gorm.DB
}

func NewDBHandler(db *gorm.DB) *DBHandler {
	return &DBHandler{db: db}
}

type tableInfo struct {
	Name      string `json:"name"`
	RowCount  int64  `json:"row_count"`
}

type columnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
	IsPK     bool   `json:"is_pk"`
}

type tableSchema struct {
	Name    string       `json:"name"`
	Columns []columnInfo `json:"columns"`
}

type rowData struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// systemSchemas lists PostgreSQL internal schemas to exclude.
var systemSchemas = []string{
	"pg_catalog",
	"information_schema",
}

func isSystemSchema(schema string) bool {
	for _, s := range systemSchemas {
		if s == schema {
			return true
		}
	}
	return false
}

// isInternalTable returns true for PostgreSQL internal/extension tables that should not be shown.
func isInternalTable(name string) bool {
	internal := []string{
		"spatial_ref_sys",
		"geography_columns",
		"geometry_columns",
	}
	for _, t := range internal {
		if t == name {
			return true
		}
	}
	return false
}

// ListTables returns all user tables with row counts (blacklist mode — excludes system tables).
func (h *DBHandler) ListTables(c *gin.Context) {
	rows, err := h.db.Raw(`
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name
	`).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tables []tableInfo
	for rows.Next() {
		var schema, name string
		rows.Scan(&schema, &name)
		if isInternalTable(name) {
			continue
		}
		var count int64
		h.db.Table(name).Count(&count)
		tables = append(tables, tableInfo{Name: schema + "." + name, RowCount: count})
	}

	if tables == nil {
		tables = []tableInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"tables": tables})
}

// resolveTable strips schema prefix and validates the table exists.
func (h *DBHandler) resolveTable(name string) (string, error) {
	// Strip schema prefix if present (e.g. "public.users" -> "users")
	parts := strings.SplitN(name, ".", 2)
	tableName := parts[len(parts)-1]

	var count int64
	h.db.Raw(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
		  AND table_name = ?
	`, tableName).Count(&count)
	if count == 0 {
		return "", fmt.Errorf("table %q not found", tableName)
	}
	return tableName, nil
}

// GetTableSchema returns column definitions for a table.
func (h *DBHandler) GetTableSchema(c *gin.Context) {
	name := c.Param("name")
	tableName, err := h.resolveTable(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schRows, err := h.db.Raw(`
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			COALESCE(c.column_default, '') as column_default,
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END as is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON c.table_schema = kcu.table_schema
			AND c.table_name = kcu.table_name
			AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name
			AND tc.constraint_type = 'PRIMARY KEY'
		WHERE c.table_schema NOT IN ('pg_catalog', 'information_schema') AND c.table_name = ?
		ORDER BY c.ordinal_position
	`, tableName).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer schRows.Close()

	var cols []columnInfo
	for schRows.Next() {
		var col columnInfo
		var nullable string
		schRows.Scan(&col.Name, &col.Type, &nullable, &col.Default, &col.IsPK)
		col.Nullable = nullable == "YES"
		cols = append(cols, col)
	}

	if cols == nil {
		cols = []columnInfo{}
	}
	c.JSON(http.StatusOK, tableSchema{Name: tableName, Columns: cols})
}

// ListRows returns paginated rows for a table.
func (h *DBHandler) ListRows(c *gin.Context) {
	name := c.Param("name")
	tableName, err := h.resolveTable(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page := 1
	pageSize := 20
	if p, ok := c.GetQuery("page"); ok {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps, ok := c.GetQuery("page_size"); ok {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// total count
	var total int64
	h.db.Table(tableName).Count(&total)

	// get column names
	var cols []string
	colRows, err := h.db.Raw(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema') AND table_name = ?
		ORDER BY ordinal_position
	`, tableName).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for colRows.Next() {
		var cn string
		colRows.Scan(&cn)
		cols = append(cols, cn)
	}
	colRows.Close()

	if cols == nil {
		cols = []string{}
	}

	// fetch rows
	offset := (page - 1) * pageSize
	sql := fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC LIMIT %d OFFSET %d",
		quoteIdent(tableName), pageSize, offset)
	dataRows, err := h.db.Raw(sql).Rows()
	if err != nil {
		// Table might not have "id" column — fall back to no ORDER BY
		sql = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
			quoteIdent(tableName), pageSize, offset)
		dataRows, err = h.db.Raw(sql).Rows()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	defer dataRows.Close()

	var result []map[string]interface{}
	for dataRows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		dataRows.Scan(ptrs...)

		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		result = append(result, row)
	}

	if result == nil {
		result = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, rowData{
		Columns:  cols,
		Rows:     result,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreateRow inserts a new row into the table.
func (h *DBHandler) CreateRow(c *gin.Context) {
	name := c.Param("name")
	tableName, err := h.resolveTable(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var values map[string]interface{}
	if err := c.ShouldBindJSON(&values); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if len(values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no columns provided"})
		return
	}

	cols := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))
	for k, v := range values {
		if !isValidColumn(k) {
			continue
		}
		cols = append(cols, quoteIdent(k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	if len(cols) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid columns"})
		return
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdent(tableName),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	if err := h.db.Exec(sql, args...).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "row inserted"})
}

// UpdateRow updates a row by ID.
func (h *DBHandler) UpdateRow(c *gin.Context) {
	name := c.Param("name")
	tableName, err := h.resolveTable(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var values map[string]interface{}
	if err := c.ShouldBindJSON(&values); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	if len(values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no columns provided"})
		return
	}

	sets := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values)+1)
	for k, v := range values {
		if !isValidColumn(k) || k == "id" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", quoteIdent(k)))
		args = append(args, v)
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid columns to update"})
		return
	}

	args = append(args, id)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		quoteIdent(tableName), strings.Join(sets, ", "))

	if err := h.db.Exec(sql, args...).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "row updated"})
}

// DeleteRow deletes a row by ID.
func (h *DBHandler) DeleteRow(c *gin.Context) {
	name := c.Param("name")
	tableName, err := h.resolveTable(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", quoteIdent(tableName))
	if err := h.db.Exec(sql, id).Error; err != nil {
		// May not have "id" column — try full table scan delete by primary key
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "row deleted"})
}

func isValidColumn(col string) bool {
	if col == "" {
		return false
	}
	for _, r := range col {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
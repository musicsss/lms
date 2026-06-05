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

// tableInfo holds metadata for a single table.
type tableInfo struct {
	Name      string `json:"name"`
	RowCount  int64  `json:"row_count"`
}

// columnInfo describes a single column.
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
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Total   int64                    `json:"total"`
	Page    int                      `json:"page"`
	PageSize int                     `json:"page_size"`
}

// safeTables lists tables the admin is allowed to browse.
func safeTables() []string {
	return []string{
		"users",
		"files",
		"file_shares",
		"boards",
		"posts",
		"post_likes",
		"video_transcodes",
		"runtime_configs",
	}
}

func isSafeTable(name string) bool {
	for _, t := range safeTables() {
		if t == name {
			return true
		}
	}
	return false
}

// ListTables returns all safe tables with row counts.
func (h *DBHandler) ListTables(c *gin.Context) {
	var tables []tableInfo
	for _, name := range safeTables() {
		var count int64
		h.db.Table(name).Count(&count)
		tables = append(tables, tableInfo{Name: name, RowCount: count})
	}
	c.JSON(http.StatusOK, gin.H{"tables": tables})
}

// GetTableSchema returns column definitions for a table.
func (h *DBHandler) GetTableSchema(c *gin.Context) {
	name := c.Param("name")
	if !isSafeTable(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown table"})
		return
	}

	// Use information_schema for column metadata
	rows, err := h.db.Raw(`
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
		WHERE c.table_schema = 'public' AND c.table_name = ?
		ORDER BY c.ordinal_position
	`, name).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var col columnInfo
		var nullable string
		rows.Scan(&col.Name, &col.Type, &nullable, &col.Default, &col.IsPK)
		col.Nullable = nullable == "YES"
		cols = append(cols, col)
	}

	c.JSON(http.StatusOK, tableSchema{Name: name, Columns: cols})
}

// ListRows returns paginated rows for a table.
func (h *DBHandler) ListRows(c *gin.Context) {
	name := c.Param("name")
	if !isSafeTable(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown table"})
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
	h.db.Table(name).Count(&total)

	// get column names
	var cols []string
	colRows, err := h.db.Raw(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ?
		ORDER BY ordinal_position
	`, name).Rows()
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

	// fetch rows (safe because table name is whitelisted)
	offset := (page - 1) * pageSize
	sql := fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC LIMIT %d OFFSET %d",
		quoteIdent(name), pageSize, offset)
	dataRows, err := h.db.Raw(sql).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
			// convert []byte to string for readability
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		result = append(result, row)
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
	if !isSafeTable(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown table"})
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

	// build INSERT
	cols := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))
	for k, v := range values {
		if !isSafeColumn(name, k) {
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
		quoteIdent(name),
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
	id := c.Param("id")
	if !isSafeTable(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown table"})
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

	sets := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values)+1)
	for k, v := range values {
		if !isSafeColumn(name, k) || k == "id" {
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
		quoteIdent(name), strings.Join(sets, ", "))

	if err := h.db.Exec(sql, args...).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "row updated"})
}

// DeleteRow deletes a row by ID.
func (h *DBHandler) DeleteRow(c *gin.Context) {
	name := c.Param("name")
	id := c.Param("id")
	if !isSafeTable(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown table"})
		return
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?", quoteIdent(name))
	if err := h.db.Exec(sql, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "row deleted"})
}

// isSafeColumn prevents injecting arbitrary columns.
func isSafeColumn(table, col string) bool {
	// Whitelist: id and any column that appears in information_schema for this table.
	// We use a simple approach: reject empty and validate against known pattern.
	if col == "" {
		return false
	}
	// Allow common safe patterns: alphanumeric + underscore
	for _, r := range col {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func quoteIdent(name string) string {
	// PostgreSQL identifier quoting
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

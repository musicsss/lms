package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/service/admin"
)

type AdminHandler struct {
	svc *admin.Service
}

func NewAdminHandler(svc *admin.Service) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// GET /api/v1/admin/stats
func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: stats failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	users, total, err := h.svc.ListUsers(page, pageSize, search)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: list users failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.UpdateUserRole(uint(id), input.Role); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: update user failed", "user_id", id, "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: user role updated", "user_id", id, "role", input.Role)
	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteUser(uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete user failed", "user_id", id, "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: user deleted", "user_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// GET /api/v1/admin/files
func (h *AdminHandler) ListFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	files, total, err := h.svc.ListFiles(page, pageSize)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: list files failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":     files,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// DELETE /api/v1/admin/files/:id
func (h *AdminHandler) DeleteFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteFile(uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete file failed", "file_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: file deleted", "file_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GET /api/v1/admin/boards
func (h *AdminHandler) ListBoards(c *gin.Context) {
	boards, err := h.svc.ListBoards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

// POST /api/v1/admin/boards
func (h *AdminHandler) CreateBoard(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	board, err := h.svc.CreateBoard(input.Name, input.Slug, input.Description, input.SortOrder)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: create board failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: board created", "board_id", board.ID, "name", board.Name)
	c.JSON(http.StatusCreated, board)
}

// PUT /api/v1/admin/boards/:id
func (h *AdminHandler) UpdateBoard(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	c.ShouldBindJSON(&input)

	board, err := h.svc.UpdateBoard(uint(id), input.Name, input.Slug, input.Description, input.SortOrder)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: update board failed", "board_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

// DELETE /api/v1/admin/boards/:id
func (h *AdminHandler) DeleteBoard(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteBoard(uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete board failed", "board_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "board deleted"})
}

// DELETE /api/v1/admin/posts/:id
func (h *AdminHandler) DeletePost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeletePostAdmin(uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete post failed", "post_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: post deleted", "post_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "post deleted"})
}

// GET /api/v1/admin/boards/:id/posts
func (h *AdminHandler) ListPosts(c *gin.Context) {
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	posts, total, err := h.svc.ListPosts(uint(boardID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":     posts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

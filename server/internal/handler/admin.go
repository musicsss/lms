package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminctx "github.com/lms/server/internal/dci/context/admin"
	auditctx "github.com/lms/server/internal/dci/context/audit"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db        *gorm.DB
	userRepo  data.UserRepo
	fileRepo  data.FileRepo
	forumRepo data.ForumRepo
	store     storage.Driver
	auditRepo data.AuditLogRepo
}

func NewAdminHandler(db *gorm.DB, userRepo data.UserRepo, fileRepo data.FileRepo, forumRepo data.ForumRepo, store storage.Driver, auditRepo data.AuditLogRepo) *AdminHandler {
	return &AdminHandler{db: db, userRepo: userRepo, fileRepo: fileRepo, forumRepo: forumRepo, store: store, auditRepo: auditRepo}
}

func (h *AdminHandler) Stats(c *gin.Context) {
	ctx := adminctx.NewGetStatsContext(h.db, h.userRepo, h.fileRepo, h.forumRepo)
	stats, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: stats failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	search := c.Query("search")

	ctx := adminctx.NewListUsersContext(h.db, h.userRepo, page, pageSize, search)
	users, total, err := ctx.Execute()
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

	ctx := adminctx.NewUpdateUserRoleContext(h.db, h.userRepo, uint(id), input.Role)
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: update user role failed", "user_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: user role updated", "user_id", id, "role", input.Role)
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := adminctx.NewDeleteUserContext(h.db, h.userRepo, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete user failed", "user_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	adminUserID := c.GetUint(middleware.CtxKeyUserID)
	auditctx.NewRecordContext(h.db, h.auditRepo, adminUserID, model.ActionAdminDeleteUser, "user", uint(id), "", c.ClientIP(), true).Execute()
	slog.InfoContext(c.Request.Context(), "admin: user deleted", "user_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AdminHandler) ListFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	ctx := adminctx.NewListFilesContext(h.db, h.fileRepo, page, pageSize)
	files, total, err := ctx.Execute()
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

func (h *AdminHandler) DeleteFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := adminctx.NewDeleteFileContext(h.db, h.fileRepo, h.store, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete file failed", "file_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	adminUserID := c.GetUint(middleware.CtxKeyUserID)
	auditctx.NewRecordContext(h.db, h.auditRepo, adminUserID, model.ActionAdminDeleteFile, "file", uint(id), "", c.ClientIP(), true).Execute()
	slog.InfoContext(c.Request.Context(), "admin: file deleted", "file_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AdminHandler) ListBoards(c *gin.Context) {
	ctx := adminctx.NewListBoardsContext(h.db, h.forumRepo)
	boards, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: list boards failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

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

	ctx := adminctx.NewCreateBoardContext(h.db, h.forumRepo, input.Name, input.Slug, input.Description, input.SortOrder)
	board, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: create board failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "admin: board created", "board_id", board.ID, "name", board.Name)
	c.JSON(http.StatusCreated, board)
}

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
		SortOrder   *int   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	ctx := adminctx.NewUpdateBoardContext(h.db, h.forumRepo, uint(id), input.Name, input.Slug, input.Description, sortOrder)
	board, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: update board failed", "board_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

func (h *AdminHandler) DeleteBoard(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := adminctx.NewDeleteBoardContext(h.db, h.forumRepo, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete board failed", "board_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *AdminHandler) ListPosts(c *gin.Context) {
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	ctx := adminctx.NewListPostsContext(h.db, h.forumRepo, uint(boardID), page, pageSize)
	posts, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: list posts failed", "err", err)
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

func (h *AdminHandler) DeletePost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := adminctx.NewDeletePostContext(h.db, h.forumRepo, uint(id))
	if err := ctx.Execute(); err != nil {
		slog.ErrorContext(c.Request.Context(), "admin: delete post failed", "post_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

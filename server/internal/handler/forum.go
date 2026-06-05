package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	forumctx "github.com/lms/server/internal/dci/context/forum"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/middleware"
	"gorm.io/gorm"
)

type ForumHandler struct {
	db        *gorm.DB
	forumRepo data.ForumRepo
}

func NewForumHandler(db *gorm.DB, forumRepo data.ForumRepo) *ForumHandler {
	return &ForumHandler{db: db, forumRepo: forumRepo}
}

func (h *ForumHandler) ListBoards(c *gin.Context) {
	ctx := forumctx.NewListBoardsContext(h.db, h.forumRepo)
	boards, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: list boards failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

func (h *ForumHandler) CreatePost(c *gin.Context) {
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	userID := c.GetUint(middleware.CtxKeyUserID)
	var input struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := forumctx.NewCreatePostContext(h.db, h.forumRepo, uint(boardID), userID, input.Title, input.Content)
	post, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: create post failed", "board_id", boardID, "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "forum: post created", "post_id", post.ID, "title", input.Title, "user_id", userID)
	c.JSON(http.StatusCreated, post)
}

func (h *ForumHandler) ListPosts(c *gin.Context) {
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	ctx := forumctx.NewListPostsContext(h.db, h.forumRepo, uint(boardID), page, pageSize)
	posts, total, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: list posts failed", "board_id", boardID, "err", err)
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

func (h *ForumHandler) GetPost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	ctx := forumctx.NewGetPostContext(h.db, h.forumRepo, uint(id))
	post, err := ctx.Execute()
	if err != nil {
		slog.WarnContext(c.Request.Context(), "forum: post not found", "post_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

func (h *ForumHandler) Reply(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID := c.GetUint(middleware.CtxKeyUserID)
	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := forumctx.NewReplyContext(h.db, h.forumRepo, uint(postID), 0, userID, input.Content)
	reply, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: reply failed", "post_id", postID, "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "forum: reply created", "reply_id", reply.ID, "post_id", postID, "user_id", userID)
	c.JSON(http.StatusCreated, reply)
}

func (h *ForumHandler) Like(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID := c.GetUint(middleware.CtxKeyUserID)
	ctx := forumctx.NewToggleLikeContext(h.db, h.forumRepo, uint(postID), userID)
	liked, err := ctx.Execute()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: toggle like failed", "post_id", postID, "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/service/forum"
)

type ForumHandler struct {
	svc *forum.Service
}

func NewForumHandler(svc *forum.Service) *ForumHandler {
	return &ForumHandler{svc: svc}
}

func (h *ForumHandler) ListBoards(c *gin.Context) {
	boards, err := h.svc.ListBoards()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: list boards failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

type createPostInput struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

func (h *ForumHandler) CreatePost(c *gin.Context) {
	boardID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board id"})
		return
	}

	userID := c.GetUint("userID")
	var input createPostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.svc.CreatePost(uint(boardID), userID, input.Title, input.Content)
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

	posts, total, err := h.svc.ListPosts(uint(boardID), page, pageSize)
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

	post, err := h.svc.GetPost(uint(id))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "forum: post not found", "post_id", id, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

type replyInput struct {
	Content string `json:"content" binding:"required"`
}

func (h *ForumHandler) Reply(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID := c.GetUint("userID")
	var input replyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reply, err := h.svc.Reply(uint(postID), 0, userID, input.Content)
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

	userID := c.GetUint("userID")
	liked, err := h.svc.ToggleLike(uint(postID), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "forum: toggle like failed", "post_id", postID, "user_id", userID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

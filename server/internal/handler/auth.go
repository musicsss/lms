package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	authctx "github.com/lms/server/internal/dci/context/auth"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/loginprotect"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/config"
	"gorm.io/gorm"
)

// AuthHandler 处理认证相关的 HTTP 请求（注册、登录、验证码）。
type AuthHandler struct {
	db       *gorm.DB
	userRepo data.UserRepo
	cfg      *config.Config
	rtEngine *runtimecfg.Engine
	guard    *loginprotect.Guard
}

func NewAuthHandler(db *gorm.DB, userRepo data.UserRepo, cfg *config.Config, rtEngine *runtimecfg.Engine, guard *loginprotect.Guard) *AuthHandler {
	return &AuthHandler{db: db, userRepo: userRepo, cfg: cfg, rtEngine: rtEngine, guard: guard}
}

// Register 处理 POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required,min=2,max=64"`
		Password string `json:"password" binding:"required,min=6,max=128"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.WarnContext(c.Request.Context(), "auth: register bad input", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := authctx.NewRegisterContext(h.db, h.userRepo, h.cfg, h.rtEngine, input.Username, input.Password, input.Email)
	user, token, err := ctx.Execute()
	if err != nil {
		slog.WarnContext(c.Request.Context(), "auth: register failed", "username", input.Username, "err", err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "auth: user registered", "user_id", user.ID, "username", user.Username)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

// Login 处理 POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Username      string `json:"username" binding:"required"`
		Password      string `json:"password" binding:"required"`
		CaptchaID     string `json:"captcha_id"`
		CaptchaAnswer string `json:"captcha_answer"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.WarnContext(c.Request.Context(), "auth: login bad input", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()

	action, blockUntil := h.guard.Check(ip, input.Username)
	switch action {
	case loginprotect.ActionBlocked:
		slog.WarnContext(c.Request.Context(), "auth: login blocked", "ip", ip, "until", blockUntil)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":         authctx.ErrBlocked.Error(),
			"blocked_until": blockUntil.Unix(),
		})
		return

	case loginprotect.ActionCaptcha:
		if input.CaptchaID == "" || input.CaptchaAnswer == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        authctx.ErrCaptchaRequired.Error(),
				"need_captcha": true,
			})
			return
		}
		if !h.guard.VerifyCaptcha(input.CaptchaID, input.CaptchaAnswer) {
			h.guard.RecordFailure(ip, input.Username)
			slog.WarnContext(c.Request.Context(), "auth: invalid captcha", "ip", ip)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        authctx.ErrCaptchaInvalid.Error(),
				"need_captcha": true,
			})
			return
		}
	}

	ctx := authctx.NewLoginContext(h.db, h.userRepo, h.cfg, h.rtEngine, input.Username, input.Password)
	user, token, err := ctx.Execute()
	if err != nil {
		h.guard.RecordFailure(ip, input.Username)

		if errors.Is(err, authctx.ErrInvalidCredentials) {
			action2, _ := h.guard.Check(ip, input.Username)
			slog.WarnContext(c.Request.Context(), "auth: login failed", "username", input.Username, "ip", ip)
			if action2 == loginprotect.ActionCaptcha {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":        authctx.ErrInvalidCredentials.Error(),
					"need_captcha": true,
				})
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		slog.ErrorContext(c.Request.Context(), "auth: login error", "username", input.Username, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.guard.RecordSuccess(ip, input.Username)
	slog.InfoContext(c.Request.Context(), "auth: user logged in", "user_id", user.ID, "username", user.Username)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) Captcha(c *gin.Context) {
	id, question, _ := h.guard.GenerateCaptcha()
	c.JSON(http.StatusOK, gin.H{
		"captcha_id": id,
		"question":   question,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint(middleware.CtxKeyUserID)
	role := c.GetString(middleware.CtxKeyRole)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
}

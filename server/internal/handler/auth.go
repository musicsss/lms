package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/loginprotect"
	"github.com/lms/server/internal/service/auth"
)

type AuthHandler struct {
	svc   *auth.Service
	guard *loginprotect.Guard
}

func NewAuthHandler(svc *auth.Service, guard *loginprotect.Guard) *AuthHandler {
	return &AuthHandler{svc: svc, guard: guard}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input auth.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.WarnContext(c.Request.Context(), "auth: register bad input", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Register(input)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "auth: register failed", "username", input.Username, "err", err)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	slog.InfoContext(c.Request.Context(), "auth: user registered", "user_id", resp.User.ID, "username", resp.User.Username)
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input auth.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		slog.WarnContext(c.Request.Context(), "auth: login bad input", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()

	action, blockUntil := h.guard.Check(ip, input.Username)
	switch action {
	case "blocked":
		slog.WarnContext(c.Request.Context(), "auth: login blocked", "ip", ip, "until", blockUntil)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":         auth.ErrBlocked.Error(),
			"blocked_until": blockUntil.Unix(),
		})
		return
	case "captcha":
		if input.CaptchaID == "" || input.CaptchaAnswer == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        auth.ErrCaptchaRequired.Error(),
				"need_captcha": true,
			})
			return
		}
		if !h.guard.VerifyCaptcha(input.CaptchaID, input.CaptchaAnswer) {
			h.guard.RecordFailure(ip, input.Username)
			slog.WarnContext(c.Request.Context(), "auth: invalid captcha", "ip", ip)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        auth.ErrCaptchaInvalid.Error(),
				"need_captcha": true,
			})
			return
		}
	}

	resp, err := h.svc.Login(input)
	if err != nil {
		h.guard.RecordFailure(ip, input.Username)

		if errors.Is(err, auth.ErrInvalidCredentials) {
			action2, _ := h.guard.Check(ip, input.Username)
			slog.WarnContext(c.Request.Context(), "auth: login failed", "username", input.Username, "ip", ip)
			if action2 == "captcha" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":        auth.ErrInvalidCredentials.Error(),
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
	slog.InfoContext(c.Request.Context(), "auth: user logged in", "user_id", resp.User.ID, "username", resp.User.Username)
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Captcha(c *gin.Context) {
	id, question, _ := h.guard.GenerateCaptcha()
	c.JSON(http.StatusOK, gin.H{
		"captcha_id": id,
		"question":   question,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint("userID")
	role := c.GetString("role")
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
}

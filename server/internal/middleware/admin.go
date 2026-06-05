package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/model"
)

// AdminMiddleware 检查当前请求是否具有管理员角色，
// 非管理员返回 403 Forbidden。
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

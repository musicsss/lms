package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Keys stored in gin.Context for downstream handlers.
const (
	CtxKeyUserID = "userID" // current login user ID
	CtxKeyRole   = "role"   // current user role (admin / user)
)

// JWT standard claim keys.
const (
	JWTClaimSub  = "sub"  // user ID
	JWTClaimRole = "role" // role
)

// AuthMiddleware parses Authorization: Bearer <token> header,
// validates JWT signature and expiry, and injects userID and role into gin.Context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.WarnContext(c.Request.Context(), "auth: missing authorization header", "ip", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			slog.WarnContext(c.Request.Context(), "auth: invalid authorization format", "ip", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			slog.WarnContext(c.Request.Context(), "auth: invalid token", "ip", c.ClientIP(), "err", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.WarnContext(c.Request.Context(), "auth: invalid token claims", "ip", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		userID, ok := claims[JWTClaimSub].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in token"})
			return
		}

		role, _ := claims[JWTClaimRole].(string)

		c.Set(CtxKeyUserID, uint(userID))
		c.Set(CtxKeyRole, role)
		c.Next()
	}
}

package middleware

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/runtimecfg"
)

var (
	staticOrigins  = []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8081"}
	dynamicOrigins []string
	originsMu      sync.RWMutex
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		originsMu.RLock()
		all := append([]string{}, staticOrigins...)
		all = append(all, dynamicOrigins...)
		originsMu.RUnlock()

		cors.New(cors.Config{
			AllowOrigins:     all,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		})(c)
	}
}

// UpdateCORSOrigins updates the dynamic CORS allowlist from runtime config.
func UpdateCORSOrigins(rows []runtimecfg.RuntimeConfig) {
	originsMu.Lock()
	defer originsMu.Unlock()

	dynamicOrigins = make([]string, 0, len(rows))
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		if origin := attrs["ORIGIN"]; origin != "" {
			dynamicOrigins = append(dynamicOrigins, origin)
		}
	}
}

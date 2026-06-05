package middleware

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/runtimecfg"
)

// CORS 预检缓存时长
const corsMaxAge = 12 * time.Hour

// 开发环境允许的来源 (无需运行时配置)
var staticOrigins = []string{
	"http://localhost:5173",
	"http://localhost:3000",
	"http://localhost:8081",
}

var (
	dynamicOrigins []string
	originsMu      sync.RWMutex
)

// CORS 返回一个支持动态白名单的 CORS 中间件。
// 静态来源 (localhost 开发端口) 始终放行；动态来源由运行时配置 ADD CORS 注入。
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
			MaxAge:           corsMaxAge,
		})(c)
	}
}

// UpdateCORSOrigins 从运行时配置中提取 CORS 白名单并更新动态来源列表。
func UpdateCORSOrigins(rows []runtimecfg.RuntimeConfig) {
	originsMu.Lock()
	defer originsMu.Unlock()

	dynamicOrigins = make([]string, 0, len(rows))
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		if origin := attrs[runtimecfg.FieldOrigin]; origin != "" {
			dynamicOrigins = append(dynamicOrigins, origin)
		}
	}
}

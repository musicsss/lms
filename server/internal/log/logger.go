package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/middleware"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志等级字符串常量
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

const loggerKey = "logger"

var (
	globalLogger  *slog.Logger
	globalLevel   = new(slog.LevelVar)
	globalLevelMu sync.RWMutex
)

func init() {
	globalLevel.Set(slog.LevelInfo)
}

// Config 日志配置
type Config struct {
	Mode       string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
}

// New 创建 slog.Logger，debug=Text release=JSON，双写 stdout + 日志文件。
func New(cfg Config) *slog.Logger {
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	if cfg.LogDir != "" {
		lj := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.LogDir, "server.log"),
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     0,
			Compress:   true,
			LocalTime:  true,
		}
		writers = append(writers, lj)
	}

	writer := io.MultiWriter(writers...)
	opts := &slog.HandlerOptions{Level: globalLevel}

	var handler slog.Handler
	if cfg.Mode == "debug" {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	globalLogger = slog.New(handler)
	return globalLogger
}

// SetLevel 运行时切换日志等级。
func SetLevel(level string) {
	globalLevelMu.Lock()
	defer globalLevelMu.Unlock()
	switch level {
	case LevelDebug:
		globalLevel.Set(slog.LevelDebug)
	case LevelInfo:
		globalLevel.Set(slog.LevelInfo)
	case LevelWarn:
		globalLevel.Set(slog.LevelWarn)
	case LevelError:
		globalLevel.Set(slog.LevelError)
	}
}

// Middleware 注入 logger 并记录每个请求。
func Middleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(loggerKey, logger)
		c.Next()

		status := c.Writer.Status()
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.String("ip", c.ClientIP()),
		}
		if q := c.Request.URL.RawQuery; q != "" {
			attrs = append(attrs, slog.String("query", q))
		}
		if userID, ok := c.Get(middleware.CtxKeyUserID); ok {
			attrs = append(attrs, slog.Any("user_id", userID))
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		logger.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// FromContext 从 gin.Context 提取 logger。
func FromContext(c *gin.Context) *slog.Logger {
	if logger, ok := c.Get(loggerKey); ok {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}

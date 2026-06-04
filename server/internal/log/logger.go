package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

const loggerKey = "logger"

var (
	globalLogger   *slog.Logger
	globalLevel    = new(slog.LevelVar)
	globalLevelMu  sync.RWMutex
)

func init() {
	globalLevel.Set(slog.LevelInfo)
}

// Config controls log output.
type Config struct {
	Mode       string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
}

// New creates a slog.Logger.
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

	opts := &slog.HandlerOptions{
		Level: globalLevel,
	}

	var handler slog.Handler
	if cfg.Mode == "debug" {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	globalLogger = slog.New(handler)
	return globalLogger
}

// SetLevel changes the global log level at runtime.
func SetLevel(level string) {
	globalLevelMu.Lock()
	defer globalLevelMu.Unlock()
	switch level {
	case "DEBUG":
		globalLevel.Set(slog.LevelDebug)
	case "INFO":
		globalLevel.Set(slog.LevelInfo)
	case "WARN":
		globalLevel.Set(slog.LevelWarn)
	case "ERROR":
		globalLevel.Set(slog.LevelError)
	}
}

// Middleware injects logger into gin.Context and logs every request.
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
		if userID, ok := c.Get("userID"); ok {
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

// FromContext extracts the logger from gin context.
func FromContext(c *gin.Context) *slog.Logger {
	if logger, ok := c.Get(loggerKey); ok {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lms/server/internal/config"
	lmslog "github.com/lms/server/internal/log"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/router"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/storage"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "setup-admin" {
		runSetupAdmin()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}

	logger := lmslog.New(lmslog.Config{
		Mode:       cfg.Server.Mode,
		LogDir:     cfg.Log.Dir,
		MaxSizeMB:  cfg.Log.MaxSizeMB,
		MaxBackups: cfg.Log.MaxBackups,
	})
	logger.Info("starting LMS server", "mode", cfg.Server.Mode)

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		logger.Error("connect database", "err", err)
		panic(fmt.Sprintf("connect database: %v", err))
	}
	logger.Info("database connected")

	if err := db.AutoMigrate(
		&model.User{},
		&model.File{},
		&model.FileShare{},
		&model.Board{},
		&model.Post{},
		&model.PostLike{},
		&model.VideoComment{},
		&model.VideoLike{},
		&model.Danmaku{},
		&model.VideoTranscode{},
		&runtimecfg.RuntimeConfig{},
	); err != nil {
		logger.Error("auto migrate", "err", err)
		panic(fmt.Sprintf("auto migrate: %v", err))
	}

	if v := os.Getenv("LMS_SETUP_ADMIN"); v != "" {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) == 2 {
			ensureAdmin(db, parts[0], parts[1])
		}
	}

	store, err := storage.NewLocalDriver(cfg.Storage.Root)
	if err != nil {
		logger.Error("init storage", "err", err)
		panic(fmt.Sprintf("init storage: %v", err))
	}

	r := router.Setup(cfg, db, store, logger)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		logger.Error("server failed", "err", err)
		panic(fmt.Sprintf("server: %v", err))
	}
}

func runSetupAdmin() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(1)
	}

	db.AutoMigrate(&model.User{})

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Admin username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Admin password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "username and password are required")
		os.Exit(1)
	}

	ensureAdmin(db, username, password)
	fmt.Println("Admin user ready.")
}

func ensureAdmin(db *gorm.DB, username, password string) {
	var existing model.User
	if err := db.Where("username = ?", username).First(&existing).Error; err == nil {
		if existing.Role == model.RoleAdmin {
			return
		}
		db.Model(&existing).Update("role", model.RoleAdmin)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
	}
	if err := db.Create(user).Error; err != nil {
		fmt.Fprintf(os.Stderr, "create admin: %v\n", err)
		os.Exit(1)
	}
}


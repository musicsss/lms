package main

import (
	"fmt"
	"log"

	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/router"
	"github.com/lms/server/internal/storage"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.File{},
		&model.FileShare{},
		&model.Board{},
		&model.Post{},
		&model.PostLike{},
		&model.VideoTranscode{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// storage
	store, err := storage.NewLocalDriver(cfg.Storage.Root)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	// router
	r := router.Setup(cfg, db, store)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("LMS server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

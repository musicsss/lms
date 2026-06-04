package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/handler"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/repository"
	"github.com/lms/server/internal/service/auth"
	"github.com/lms/server/internal/service/file"
	"github.com/lms/server/internal/service/forum"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, store storage.Driver) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS())

	// repos
	userRepo := repository.NewUserRepo(db)
	fileRepo := repository.NewFileRepo(db)
	shareRepo := repository.NewShareRepo(db)
	forumRepo := repository.NewForumRepo(db)

	// services
	authSvc := auth.NewService(userRepo, cfg)
	fileSvc := file.NewService(fileRepo, shareRepo, store)
	forumSvc := forum.NewService(forumRepo)

	// handlers
	authH := handler.NewAuthHandler(authSvc)
	fileH := handler.NewFileHandler(fileSvc)
	forumH := handler.NewForumHandler(forumSvc)

	// public routes
	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
		}

		api.GET("/share/:token", fileH.GetShare)
	}

	// protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		protected.GET("/auth/me", authH.Me)

		protected.GET("/files", fileH.List)
		protected.POST("/files/upload", fileH.Upload)
		protected.POST("/files/mkdir", fileH.Mkdir)
		protected.GET("/files/:id/download", fileH.Download)
		protected.DELETE("/files/:id", fileH.Delete)
		protected.POST("/files/:id/share", fileH.Share)

		protected.GET("/boards", forumH.ListBoards)
		protected.GET("/boards/:id/posts", forumH.ListPosts)
		protected.POST("/boards/:id/posts", forumH.CreatePost)
		protected.GET("/posts/:id", forumH.GetPost)
		protected.POST("/posts/:id/reply", forumH.Reply)
		protected.POST("/posts/:id/like", forumH.Like)
	}

	return r
}

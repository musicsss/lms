package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/handler"
	lmslog "github.com/lms/server/internal/log"
	"github.com/lms/server/internal/loginprotect"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/repository"
	"github.com/lms/server/internal/runtimecfg"
	"github.com/lms/server/internal/service/admin"
	"github.com/lms/server/internal/service/auth"
	"github.com/lms/server/internal/service/file"
	"github.com/lms/server/internal/service/forum"
	"github.com/lms/server/internal/storage"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, store storage.Driver, logger *slog.Logger) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(lmslog.Middleware(logger))

	guard := loginprotect.NewGuard()

	// runtime config engine
	rtStore := runtimecfg.NewStore(db)
	rtEngine := runtimecfg.NewEngine(rtStore)
	if err := rtEngine.Start(); err != nil {
		logger.Error("start runtime config engine", "err", err)
	}

	// wire runtime config into modules
	rtEngine.OnChange(func(target string) {
		switch target {
		case "SYSLOG":
			if v := rtEngine.GetSet("SYSLOG"); v != nil {
				lmslog.SetLevel(v["LEVEL"])
			}
		case "LGFAILFIBPLCY":
			guard.ApplyPolicies(rtEngine.GetAdds("LGFAILFIBPLCY"))
		case "CORS":
			middleware.UpdateCORSOrigins(rtEngine.GetAdds("CORS"))
		case "CLRLIMIT":
			guard.ClearAll()
		case "RELOAD":
			// handled by engine.Reload
		}
	})

	// apply initial config
	if v := rtEngine.GetSet("SYSLOG"); v != nil {
		lmslog.SetLevel(v["LEVEL"])
	}
	guard.ApplyPolicies(rtEngine.GetAdds("LGFAILFIBPLCY"))
	middleware.UpdateCORSOrigins(rtEngine.GetAdds("CORS"))

	// repos
	userRepo := repository.NewUserRepo(db)
	fileRepo := repository.NewFileRepo(db)
	shareRepo := repository.NewShareRepo(db)
	forumRepo := repository.NewForumRepo(db)

	// services
	authSvc := auth.NewService(userRepo, cfg, rtEngine)
	fileSvc := file.NewService(fileRepo, shareRepo, store, rtEngine)
	forumSvc := forum.NewService(forumRepo)
	adminSvc := admin.NewService(userRepo, fileRepo, forumRepo, store)

	// handlers
	authH := handler.NewAuthHandler(authSvc, guard)
	fileH := handler.NewFileHandler(fileSvc)
	forumH := handler.NewForumHandler(forumSvc)
	adminH := handler.NewAdminHandler(adminSvc)
	configH := handler.NewConfigHandler(rtEngine)

	// public routes
	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.GET("/captcha", authH.Captcha)
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

	// admin routes
	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	adminGroup.Use(middleware.AdminMiddleware())
	{
		adminGroup.GET("/stats", adminH.Stats)

		adminGroup.GET("/users", adminH.ListUsers)
		adminGroup.PUT("/users/:id", adminH.UpdateUser)
		adminGroup.DELETE("/users/:id", adminH.DeleteUser)

		adminGroup.GET("/files", adminH.ListFiles)
		adminGroup.DELETE("/files/:id", adminH.DeleteFile)

		adminGroup.GET("/boards", adminH.ListBoards)
		adminGroup.POST("/boards", adminH.CreateBoard)
		adminGroup.PUT("/boards/:id", adminH.UpdateBoard)
		adminGroup.DELETE("/boards/:id", adminH.DeleteBoard)

		adminGroup.GET("/boards/:id/posts", adminH.ListPosts)
		adminGroup.DELETE("/posts/:id", adminH.DeletePost)

		adminGroup.POST("/config/exec", configH.Exec)
		adminGroup.GET("/config/targets", configH.Targets)
	}

	return r
}

package router

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/handler"
	"github.com/lms/server/internal/presence"
	lmslog "github.com/lms/server/internal/log"
	"github.com/lms/server/internal/loginprotect"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/runtimecfg"
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
		case runtimecfg.TargetSyslog:
			if v := rtEngine.GetSet(runtimecfg.TargetSyslog); v != nil {
				lmslog.SetLevel(v["LEVEL"])
			}
		case runtimecfg.TargetLoginFail:
			guard.ApplyPolicies(rtEngine.GetAdds(runtimecfg.TargetLoginFail))
		case runtimecfg.TargetCORS:
			middleware.UpdateCORSOrigins(rtEngine.GetAdds(runtimecfg.TargetCORS))
		case "CLRLIMIT":
			guard.ClearAll()
		}
	})

	// apply initial config
	if v := rtEngine.GetSet(runtimecfg.TargetSyslog); v != nil {
		lmslog.SetLevel(v["LEVEL"])
	}
	guard.ApplyPolicies(rtEngine.GetAdds(runtimecfg.TargetLoginFail))
	middleware.UpdateCORSOrigins(rtEngine.GetAdds(runtimecfg.TargetCORS))

	// DCI data layer: repositories
	userRepo := data.NewUserRepo(db)
	fileRepo := data.NewFileRepo(db)
	shareRepo := data.NewShareRepo(db)
	forumRepo := data.NewForumRepo(db)
	videoSocialRepo := data.NewVideoSocialRepo(db)
	danmakuRepo := data.NewDanmakuRepo(db)

	// presence hub for online watcher tracking
	presenceHub := presence.NewHub()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			presenceHub.Cleanup(15 * time.Second)
		}
	}()

	// handlers (domain wiring via DCI contexts)
	authH := handler.NewAuthHandler(db, userRepo, cfg, rtEngine, guard)
	fileH := handler.NewFileHandler(db, fileRepo, shareRepo, videoSocialRepo, store, rtEngine, presenceHub)
	forumH := handler.NewForumHandler(db, forumRepo)
	adminH := handler.NewAdminHandler(db, userRepo, fileRepo, forumRepo, store)
	configH := handler.NewConfigHandler(rtEngine)
	dbH := handler.NewDBHandler(db)
	videoSocialH := handler.NewVideoSocialHandler(db, videoSocialRepo, fileRepo)
	danmakuH := handler.NewDanmakuHandler(db, danmakuRepo, fileRepo)
	userProfileH := handler.NewUserProfileHandler(db, userRepo, fileRepo, forumRepo, videoSocialRepo, store)

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
		api.GET("/video-play/:id", fileH.Play)
		api.GET("/videos/:id/thumbnail", fileH.Thumbnail)
		api.GET("/videos/:id/info", fileH.VideoInfo)
		api.GET("/videos/:id/comments", videoSocialH.GetComments)
		api.GET("/videos/:id/danmaku", danmakuH.List)
		api.GET("/files/download-by-key/*key", userProfileH.DownloadByKey)
		api.GET("/users/:id/profile", userProfileH.GetUserProfile)
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

		protected.GET("/videos/random", fileH.RandomVideos)
		protected.POST("/videos/:id/heartbeat", fileH.Heartbeat)

		// public watchers count
		api.GET("/videos/:id/watchers", fileH.Watchers)
		protected.POST("/videos/:id/comments", videoSocialH.CreateComment)
		protected.POST("/videos/:id/danmaku", danmakuH.Send)
		protected.POST("/videos/:id/like-toggle", videoSocialH.ToggleLike)
		protected.GET("/videos/:id/like-status", videoSocialH.GetLikeStatus)

		protected.GET("/boards", forumH.ListBoards)
		protected.GET("/boards/:id/posts", forumH.ListPosts)
		protected.POST("/boards/:id/posts", forumH.CreatePost)
		protected.GET("/posts/:id", forumH.GetPost)
		protected.POST("/posts/:id/reply", forumH.Reply)
		protected.POST("/posts/:id/like", forumH.Like)

		protected.GET("/user/profile", userProfileH.GetProfile)
		protected.PUT("/user/profile", userProfileH.UpdateProfile)
		protected.PUT("/user/password", userProfileH.UpdatePassword)
		protected.GET("/user/files", userProfileH.GetUserFiles)
		protected.GET("/user/posts", userProfileH.GetUserPosts)
		protected.GET("/user/liked-videos", userProfileH.GetUserLikedVideos)
		protected.POST("/user/avatar", userProfileH.UploadAvatar)
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

		adminGroup.GET("/danmaku", danmakuH.AdminList)
		adminGroup.POST("/danmaku/:id/approve", danmakuH.AdminApprove)
		adminGroup.POST("/danmaku/:id/reject", danmakuH.AdminReject)
		adminGroup.DELETE("/danmaku/:id", danmakuH.AdminDelete)

		adminGroup.POST("/config/exec", configH.Exec)
		adminGroup.GET("/config/targets", configH.Targets)

		adminGroup.GET("/db/tables", dbH.ListTables)
		adminGroup.GET("/db/tables/:name", dbH.GetTableSchema)
		adminGroup.GET("/db/tables/:name/rows", dbH.ListRows)
		adminGroup.POST("/db/tables/:name", dbH.CreateRow)
		adminGroup.PUT("/db/tables/:name/:id", dbH.UpdateRow)
		adminGroup.DELETE("/db/tables/:name/:id", dbH.DeleteRow)
	}

	return r
}
package routers

import (
	"net/http"
	"path/filepath"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/middleware"
	"favor-dao-backend/internal/routers/api"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewRouter() *gin.Engine {
	e := gin.New()
	e.HandleMethodNotAllowed = true
	e.Use(gin.Logger())
	e.Use(gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders("Authorization")
	e.Use(cors.New(corsConfig))

	// On-demand registration of docs, static resources, LocalOSS routing
	{
		registerDocs(e)
		registerStatick(e)
		routeLocalOSS(e)
	}

	// v1 group api
	r := e.Group("/v1")

	r.GET("/", api.Version)

	r.POST("/auth/login", api.Login)

	r.POST("/auth/register", api.Register)

	noAuthApi := r.Group("/")
	{
		noAuthApi.GET("/post", api.GetPost)

		noAuthApi.GET("/tags", api.GetPostTags)

		noAuthApi.GET("/user/profile", api.GetUserProfile)
	}

	looseApi := r.Group("/").Use(middleware.JwtLoose())
	{
		looseApi.GET("/posts", api.GetPostList)

		looseApi.GET("/user/posts", api.GetUserPosts)
	}

	authApi := r.Group("/").Use(middleware.JWT())
	privApi := r.Group("/").Use(middleware.JWT())
	adminApi := r.Group("/").Use(middleware.JWT()).Use(middleware.Admin())
	{
		authApi.GET("/user/info", api.GetUserInfo)

		authApi.GET("/user/collections", api.GetUserCollections)

		authApi.GET("/user/stars", api.GetUserStars)

		authApi.POST("/user/password", api.ChangeUserPassword)

		authApi.POST("/user/nickname", api.ChangeNickname)

		authApi.POST("/user/avatar", api.ChangeAvatar)

		authApi.GET("/suggest/users", api.GetSuggestUsers)

		authApi.GET("/suggest/tags", api.GetSuggestTags)

		privApi.POST("/attachment", api.UploadAttachment)

		privApi.POST("/post", api.CreatePost)

		privApi.DELETE("/post", api.DeletePost)

		authApi.GET("/post/star", api.GetPostStar)

		privApi.POST("/post/star", api.PostStar)

		authApi.GET("/post/collection", api.GetPostCollection)

		privApi.POST("/post/collection", api.PostCollection)

		privApi.POST("/post/lock", api.LockPost)

		privApi.POST("/post/stick", api.StickPost)

		privApi.POST("/post/visibility", api.VisiblePost)

		// Management - Banned/Unblocked Users
		adminApi.POST("/admin/user/status", api.ChangeUserStatus)
	}

	// default 404
	e.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "Not Found",
		})
	})

	// default 405
	e.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"code": 405,
			"msg":  "Method Not Allowed",
		})
	})

	return e
}

// routeLocalOSS register LocalOSS route if neeed
func routeLocalOSS(e *gin.Engine) {
	if !conf.CfgIf("LocalOSS") {
		return
	}

	savePath, err := filepath.Abs(conf.LocalOSSSetting.SavePath)
	if err != nil {
		logrus.Fatalf("get localOSS save path err: %v", err)
	}
	e.Static("/oss", savePath)

	logrus.Infof("register LocalOSS route in /oss on save path: %s", savePath)
}

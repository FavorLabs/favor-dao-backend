package routers

import (
	"net/http"

	"favor-dao-backend/internal/middleware"
	"favor-dao-backend/internal/routers/api"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	e := gin.New()
	e.HandleMethodNotAllowed = true
	e.Use(gin.Logger())
	e.Use(gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders("X-Session-Token")
	e.Use(cors.New(corsConfig))

	// v1 group api
	r := e.Group("/v1")

	r.Use(Aggregate())

	r.GET("/", api.Version)

	//r.POST("/auth/login", api.Login)
	r.Use(cacheResponse).POST("/auth/login_hello", api.Login)

	noAuthApi := r.Group("/")
	{
		noAuthApi.GET("/post", api.GetPost)

		noAuthApi.GET("/tags", api.GetPostTags)

		noAuthApi.GET("/user/profile", api.GetUserProfile)

		noAuthApi.GET("/posts", api.GetPostList)

		noAuthApi.GET("/user/posts", api.GetUserPosts)

		noAuthApi.GET("/dao/posts", api.GetDaoPosts)
	}

	authApi := r.Group("/").Use(middleware.Session())
	privApi := r.Group("/").Use(middleware.Session())
	{
		authApi.GET("/user/info", api.GetUserInfo)

		authApi.GET("/user/collections", api.GetUserCollections)

		authApi.GET("/user/stars", api.GetUserStars)

		authApi.POST("/user/nickname", api.ChangeNickname)

		authApi.POST("/user/avatar", api.ChangeAvatar)

		authApi.GET("/suggest/users", api.GetSuggestUsers)

		authApi.GET("/suggest/tags", api.GetSuggestTags)

		authApi.GET("/posts/focus", api.GetFocusPostList)

		privApi.POST("/post", api.CreatePost)

		privApi.DELETE("/post", api.DeletePost)

		authApi.GET("/post/star", api.GetPostStar)

		privApi.POST("/post/star", api.PostStar)

		authApi.GET("/post/view", api.GetPostView)

		privApi.POST("/post/view", api.PostView)

		authApi.GET("/post/collection", api.GetPostCollection)

		privApi.POST("/post/collection", api.PostCollection)

		// privApi.POST("/post/lock", api.LockPost)

		privApi.POST("/post/stick", api.StickPost)

		privApi.POST("/post/visibility", api.VisiblePost)

		// dao
		authApi.GET("/daos", api.GetDaos)
		authApi.GET("/dao", api.GetDao)
		authApi.GET("/dao/my", api.GetMyDaoList)
		//authApi.POST("/dao", api.CreateDao)
		authApi.Use(cacheResponse).POST("/dao_server", api.CreateDao)
		authApi.PUT("/dao", api.UpdateDao)
		authApi.GET("/dao/bookmark", api.GetDaoBookmark)
		//authApi.POST("/dao/bookmark", api.ActionDaoBookmark)
		authApi.Use(cacheResponse).POST("/dao/bookmark_server", api.ActionDaoBookmark)
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

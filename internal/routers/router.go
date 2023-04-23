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

	r.GET("/", api.Version)

	r.POST("/auth/login", api.Login)

	r.GET("/pay/notify", api.PayNotify).Use(middleware.AllowHost())

	noAuthApi := r.Group("/").Use(middleware.Session())
	{
		noAuthApi.GET("/post", api.GetPost)

		noAuthApi.GET("/tags", api.GetPostTags)

		noAuthApi.GET("/user/profile", api.GetUserProfile)

		noAuthApi.GET("/posts", api.GetPostList)

		noAuthApi.GET("/post/comments", api.GetPostComments)

		noAuthApi.GET("/user/posts", api.GetUserPosts)

		noAuthApi.GET("/dao/posts", api.GetDaoPosts)

	}

	authApi := r.Group("/").Use(middleware.Login())
	privApi := r.Group("/").Use(middleware.Login())
	{
		authApi.DELETE("/account", api.DeleteAccount)

		authApi.GET("/user/info", api.GetUserInfo)

		authApi.GET("/user/statistic", api.GetUserStatistic)

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

		privApi.POST("/post/stick", api.StickPost)

		privApi.POST("/post/visibility", api.VisiblePost)

		privApi.POST("/post/comment", api.CreatePostComment)

		privApi.DELETE("/post/comment", api.DeletePostComment)

		privApi.POST("/post/comment/reply", api.CreatePostCommentReply)

		privApi.DELETE("/post/comment/reply", api.DeletePostCommentReply)

		// dao
		authApi.GET("/daos", api.GetDaos)
		authApi.GET("/dao", api.GetDao)
		authApi.GET("/dao/my", api.GetMyDaoList)
		authApi.POST("/dao", api.CreateDao)
		authApi.PUT("/dao", api.UpdateDao)
		authApi.GET("/dao/bookmark", api.GetDaoBookmark)
		authApi.POST("/dao/bookmark", api.ActionDaoBookmark)
		authApi.POST("/dao/sub/:daoId", api.SubDao)

		// chat
		authApi.GET("/chat/groups", api.GetChatGroups)
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

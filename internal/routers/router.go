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

	r.Group("/pay").Use(middleware.AllowHost()).GET("/notify", api.PayNotify)

	noAuthApi := r.Group("/").Use(middleware.Session())
	{
		noAuthApi.GET("/tags", api.GetPostTags)

		noAuthApi.GET("/user/profile", api.GetUserProfile)

		// post
		noAuthApi.GET("/user/posts", api.GetUserPosts)
		noAuthApi.GET("/dao/posts", api.GetDaoPosts)
		noAuthApi.GET("/posts", api.GetPostList)

		noAuthApi.GET("/post", api.GetPost)
		noAuthApi.GET("/post/comments", api.GetPostComments)

		noAuthApi.POST("/post/view", api.PostView)
		noAuthApi.GET("/post/view", api.GetPostView)

		// DAO
		noAuthApi.GET("/dao", api.GetDao)
		noAuthApi.GET("/dao/recommend", api.GetDaoList)
	}

	authApi := r.Group("/").Use(middleware.Login())
	{
		// user
		authApi.DELETE("/account", api.DeleteAccount)
		authApi.GET("/user/info", api.GetUserInfo)
		authApi.POST("/user/nickname", api.ChangeNickname)
		authApi.POST("/user/avatar", api.ChangeAvatar)
		authApi.GET("/user/accounts", api.GetAccounts)
		authApi.GET("/user/statistic", api.GetUserStatistic)
		authApi.GET("/user/collections", api.GetUserCollections)

		// suggest for search
		authApi.GET("/suggest/users", api.GetSuggestUsers)
		authApi.GET("/suggest/tags", api.GetSuggestTags)

		// post
		authApi.GET("/user/stars", api.GetUserStars)
		authApi.GET("/posts/focus", api.GetFocusPostList)

		authApi.POST("/post", api.CreatePost)
		authApi.DELETE("/post", api.DeletePost)

		authApi.GET("/post/star", api.GetPostStar)
		authApi.POST("/post/star", api.PostStar)
		authApi.GET("/post/collection", api.GetPostCollection)
		authApi.POST("/post/collection", api.PostCollection)

		authApi.POST("/post/stick", api.StickPost)
		authApi.POST("/post/visibility", api.VisiblePost)
		authApi.POST("/post/block/:post_id", api.BlockPost)
		authApi.POST("/post/report/:post_id", api.ReportPost)

		authApi.POST("/post/comment", api.CreatePostComment)
		authApi.DELETE("/post/comment", api.DeletePostComment)
		authApi.POST("/post/comment/reply", api.CreatePostCommentReply)
		authApi.DELETE("/post/comment/reply", api.DeletePostCommentReply)

		// red packet
		authApi.POST("/redpacket", api.CreateRedpacket)
		authApi.POST("/redpacket/:redpacket_id", api.ClaimRedpacket)
		authApi.GET("/redpacket/:redpacket_id", api.RedpacketInfo)
		authApi.GET("/redpacket/stats/claims", api.RedpacketStatsClaims)
		authApi.GET("/redpacket/stats/sends", api.RedpacketStatsSends)
		authApi.GET("/redpacket/claims", api.RedpacketClaimList)
		authApi.GET("/redpacket/sends", api.RedpacketSendList)

		// dao
		authApi.GET("/daos", api.GetDaos)
		authApi.GET("/dao/my", api.GetMyDaoList)
		authApi.POST("/dao", api.CreateDao)
		authApi.PUT("/dao", api.UpdateDao)
		authApi.GET("/dao/bookmark", api.GetDaoBookmark)
		authApi.POST("/dao/bookmark", api.ActionDaoBookmark)
		authApi.POST("/dao/sub/:dao_id", api.SubDao)
		authApi.POST("/dao/block/:dao_id", api.BlockDAO)

		// chat
		authApi.GET("/chat/groups", api.GetChatGroups)
	}

	// test := r.Group("/test")
	// {
	// 	test.POST("/redpacket", api.CreateRedpacketTest)
	// 	test.POST("/redpacket/:redpacket_id", api.ClaimRedpacketTest)
	// }

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

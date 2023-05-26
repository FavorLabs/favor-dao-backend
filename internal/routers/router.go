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
		noAuthApi.GET("/post", api.GetPost)
		noAuthApi.GET("/dao", api.GetDao)
		noAuthApi.GET("/dao/recommend", api.GetDaoList)

		noAuthApi.GET("/tags", api.GetPostTags)

		noAuthApi.GET("/user/profile", api.GetUserProfile)

		noAuthApi.GET("/posts", api.GetPostList)

		noAuthApi.POST("/post/view", api.PostView)
		noAuthApi.GET("/post/view", api.GetPostView)

		noAuthApi.GET("/post/comments", api.GetPostComments)

		noAuthApi.GET("/user/posts", api.GetUserPosts)

		noAuthApi.GET("/dao/posts", api.GetDaoPosts)

	}

	authApi := r.Group("/").Use(middleware.Login())
	privApi := r.Group("/").Use(middleware.Login())
	{
		authApi.DELETE("/account", api.DeleteAccount)

		authApi.GET("/user/info", api.GetUserInfo)
		authApi.GET("/user/accounts", api.GetAccounts)

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

		authApi.GET("/post/collection", api.GetPostCollection)

		privApi.POST("/post/collection", api.PostCollection)

		privApi.POST("/post/stick", api.StickPost)

		privApi.POST("/post/visibility", api.VisiblePost)

		privApi.POST("/post/comment", api.CreatePostComment)

		privApi.DELETE("/post/comment", api.DeletePostComment)

		privApi.POST("/post/comment/reply", api.CreatePostCommentReply)

		privApi.DELETE("/post/comment/reply", api.DeletePostCommentReply)

		// notify
		privApi.GET("/notify/group", api.NotifyGroupList)
		privApi.GET("/notify/:fromId", api.NotifyByFrom)
		privApi.GET("notify/sys/:organId", api.NotifySys)
		privApi.GET("notify/unread/:fromId", api.NotifyUnread)
		privApi.DELETE("/notify/:id", api.DeletedNotifyById)
		privApi.DELETE("/notify/group/:fromId", api.DeletedNotifyByFromId)
		privApi.PUT("/notify/group/:fromId", api.PutNotifyRead)

		// red packet
		privApi.POST("/redpacket", api.CreateRedpacket)
		privApi.POST("/redpacket/:redpacket_id", api.ClaimRedpacket)
		privApi.GET("/redpacket/:redpacket_id", api.RedpacketInfo)
		privApi.GET("/redpacket/stats/claims", api.RedpacketStatsClaims)
		privApi.GET("/redpacket/stats/sends", api.RedpacketStatsSends)
		privApi.GET("/redpacket/claims", api.RedpacketClaimList)
		privApi.GET("/redpacket/sends", api.RedpacketSendList)

		// dao
		authApi.GET("/daos", api.GetDaos)
		authApi.GET("/dao/my", api.GetMyDaoList)
		authApi.POST("/dao", api.CreateDao)
		authApi.PUT("/dao", api.UpdateDao)
		authApi.GET("/dao/bookmark", api.GetDaoBookmark)
		authApi.POST("/dao/bookmark", api.ActionDaoBookmark)
		authApi.POST("/dao/sub/:dao_id", api.SubDao)

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

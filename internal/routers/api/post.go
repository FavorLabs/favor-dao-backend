package api

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetPostList(c *gin.Context) {
	response := app.NewResponse(c)

	q := &core.QueryReq{
		Query: c.Query("query"),
		Type:  "search",
	}
	if c.Query("type") == "tag" {
		q.Type = "tag"
	}

	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)
	if q.Query == "" && q.Type == "search" {
		resp, err := service.GetIndexPosts(user, offset, limit)
		if err != nil {
			logrus.Errorf("service.GetPostList err: %v\n", err)
			response.ToErrorResponse(errcode.GetPostsFailed)
			return
		}

		response.ToResponseList(resp.Tweets, resp.Total)
	} else {
		posts, totalRows, err := service.GetPostListFromSearch(user, q, offset, limit)

		if err != nil {
			logrus.Errorf("service.GetPostListFromSearch err: %v\n", err)
			response.ToErrorResponse(errcode.GetPostsFailed)
			return
		}
		response.ToResponseList(posts, totalRows)
	}
}

func GetPost(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).MustInt64()
	response := app.NewResponse(c)

	postFormated, err := service.GetPost(postID)

	if err != nil {
		logrus.Errorf("service.GetPost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}

	response.ToResponse(postFormated)
}

func CreatePost(c *gin.Context) {
	param := service.PostCreationReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	// todo userid? address
	address, _ := c.Get("Address")
	post, err := service.CreatePost(c, address.(string), param)

	if err != nil {
		logrus.Errorf("service.CreatePost err: %v\n", err)
		response.ToErrorResponse(errcode.CreatePostFailed)
		return
	}

	response.ToResponse(post)
}

func DeletePost(c *gin.Context) {
	param := service.PostDelReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, exist := userFrom(c)
	if !exist {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	err := service.DeletePost(user, param.ID)
	if err != nil {
		logrus.Errorf("service.DeletePost err: %v\n", err)
		response.ToErrorResponse(err)
		return
	}

	response.ToResponse(nil)
}

func GetPostStar(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).MustInt64()
	response := app.NewResponse(c)

	userID, _ := c.Get("UID")

	_, err := service.GetPostStar(postID, userID.(int64))
	if err != nil {
		response.ToResponse(gin.H{
			"status": false,
		})

		return
	}

	response.ToResponse(gin.H{
		"status": true,
	})
}

func PostStar(c *gin.Context) {
	param := service.PostStarReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	userID, _ := c.Get("UID")

	status := false
	star, err := service.GetPostStar(param.ID, userID.(int64))
	if err != nil {
		// 创建Star
		_, err = service.CreatePostStar(param.ID, userID.(int64))
		status = true
	} else {
		// 取消Star
		err = service.DeletePostStar(star)
	}

	if err != nil {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	response.ToResponse(gin.H{
		"status": status,
	})
}

func GetPostCollection(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).MustInt64()
	response := app.NewResponse(c)

	userID, _ := c.Get("UID")

	_, err := service.GetPostCollection(postID, userID.(int64))
	if err != nil {
		response.ToResponse(gin.H{
			"status": false,
		})

		return
	}

	response.ToResponse(gin.H{
		"status": true,
	})
}

func PostCollection(c *gin.Context) {
	param := service.PostCollectionReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	userID, _ := c.Get("UID")

	status := false
	collection, err := service.GetPostCollection(param.ID, userID.(int64))
	if err != nil {
		// 创建collection
		_, err = service.CreatePostCollection(param.ID, userID.(int64))
		status = true
	} else {
		// 取消Star
		err = service.DeletePostCollection(collection)
	}

	if err != nil {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	response.ToResponse(gin.H{
		"status": status,
	})
}

func LockPost(c *gin.Context) {
	param := service.PostLockReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, _ := c.Get("USER")

	// 获取Post
	postFormated, err := service.GetPost(param.ID)
	if err != nil {
		logrus.Errorf("service.GetPost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}

	if postFormated.UserID != user.(*model.User).ID && !user.(*model.User).IsAdmin {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}
	err = service.LockPost(param.ID)
	if err != nil {
		logrus.Errorf("service.LockPost err: %v\n", err)
		response.ToErrorResponse(errcode.LockPostFailed)
		return
	}

	response.ToResponse(gin.H{
		"lock_status": 1 - postFormated.IsLock,
	})
}

func StickPost(c *gin.Context) {
	param := service.PostStickReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, _ := c.Get("USER")

	// 获取Post
	postFormated, err := service.GetPost(param.ID)
	if err != nil {
		logrus.Errorf("service.GetPost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}

	if !user.(*model.User).IsAdmin {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}
	err = service.StickPost(param.ID)
	if err != nil {
		logrus.Errorf("service.StickPost err: %v\n", err)
		response.ToErrorResponse(errcode.LockPostFailed)
		return
	}

	response.ToResponse(gin.H{
		"top_status": 1 - postFormated.IsTop,
	})
}

func VisiblePost(c *gin.Context) {
	param := service.PostVisibilityReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, _ := userFrom(c)
	if err := service.VisiblePost(user, param.ID, param.Visibility); err != nil {
		logrus.Errorf("service.VisiblePost err: %v\n", err)
		response.ToErrorResponse(err)
		return
	}

	response.ToResponse(gin.H{
		"visibility": param.Visibility,
	})
}

func GetPostTags(c *gin.Context) {
	param := service.PostTagsReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	tags, err := service.GetPostTags(&param)
	if err != nil {
		logrus.Errorf("service.GetPostTags err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostTagsFailed)
		return

	}

	response.ToResponse(tags)
}

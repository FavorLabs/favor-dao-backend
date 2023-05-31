package api

import (
	"errors"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/util/gconv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func parseQueryReq(c *gin.Context) *core.QueryReq {
	q := &core.QueryReq{
		Query: c.Query("query"),
		Tag:   c.Query("tag"),
	}
	types := c.Query("type")
	if types != "" {
		if types == "post" {
			q.Type = core.AllQueryPostType
		} else {
			for _, v := range strings.Split(types, ",") {
				if v != "" {
					q.Type = append(q.Type, core.PostType(gconv.Int(v)))
				}
			}
		}
	}
	addresses := c.Query("address")
	if addresses != "" {
		for _, v := range strings.Split(addresses, ",") {
			if v != "" {
				q.Addresses = append(q.Addresses, v)
			}
		}
	}
	daoIDs := c.Query("daoId")
	if daoIDs != "" {
		for _, v := range strings.Split(daoIDs, ",") {
			if v != "" {
				q.DaoIDs = append(q.DaoIDs, v)
			}
		}
	}
	if strings.HasPrefix(q.Query, "0x") {
		q.Addresses = append(q.Addresses, q.Query)
	}
	sort := c.Query("sort")
	if sort != "" {
		for _, v := range strings.Split(sort, ",") {
			if v != "" {
				tmp := strings.Split(v, ":")
				if len(tmp) == 2 {
					mp := make(map[string]interface{})
					mp[tmp[0]] = tmp[1]
					q.Sort = append(q.Sort, mp)
				}
			}
		}
	}
	return q
}

func GetPostList(c *gin.Context) {
	response := app.NewResponse(c)
	q := parseQueryReq(c)
	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)

	if user != nil {
		q.BlockDaoIDs = service.GetBlockDaoIDs(user)
		if !(len(q.Type) == 1 && q.Type[0] == model.DAO) {
			q.BlockPostIDs = service.GetBlockPostIDs(user)
		}
	}

	// only public
	posts, totalRows, err := service.GetPostListFromSearch(user, q, offset, limit)
	if err != nil {
		logrus.Errorf("service.GetPostListFromSearch err: %v\n", err)
		response.ToResponseList([]*model.PostFormatted{}, 0)
		return
	}
	response.ToResponseList(posts, totalRows)
}

func GetFocusPostList(c *gin.Context) {
	response := app.NewResponse(c)
	q := parseQueryReq(c)
	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)
	if len(q.Type) == 0 {
		q.Type = core.AllQueryPostType
	}
	daoIds := service.GetDaoBookmarkIDsByAddress(user.Address)
	if len(daoIds) == 0 {
		response.ToResponseList([]*model.PostFormatted{}, 0)
		return
	}
	q.DaoIDs = daoIds

	q.BlockPostIDs = service.GetBlockPostIDs(user)

	// only public
	posts, totalRows, err := service.GetPostListFromSearch(user, q, offset, limit)
	if err != nil {
		logrus.Errorf("service.GetPostListFromSearch err: %v\n", err)
		response.ToResponseList([]*model.PostFormatted{}, 0)
		return
	}
	response.ToResponseList(posts, totalRows)
}

func GetPost(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).String()
	response := app.NewResponse(c)
	postId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		logrus.Errorf("primitive.ObjectIDFromHex err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	var userAddress string
	user, _ := userFrom(c)
	if user != nil {
		userAddress = user.Address
	}
	postFormatted, err := service.GetPost(userAddress, postId)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.ToErrorResponse(errcode.NotFound)
			return
		}
		logrus.Errorf("service.GetPost err: %v\n", err)
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	postFormatted = service.FilterMemberContent(user, postFormatted)

	response.ToResponse(postFormatted)
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

	user, _ := userFrom(c)
	e := service.CheckIsMyDAO(user.Address, param.DaoId)
	if e != nil {
		response.ToErrorResponse(e)
		return
	}
	post, err := service.CreatePost(user, param)

	if err != nil {
		logrus.Errorf("service.CreatePost err: %v\n", err)
		response.ToErrorResponse(errcode.CreatePostFailed)
		return
	}

	response.ToResponse(post)
}

func DeletePost(c *gin.Context) {
	response := app.NewResponse(c)
	id := c.Query("id")

	user, exist := userFrom(c)
	if !exist {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}
	postId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("service.DeletePost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	err1 := service.DeletePost(user, postId)
	if err1 != nil {
		logrus.Errorf("service.DeletePost err: %v\n", err1)
		response.ToErrorResponse(err1)
		return
	}

	response.ToResponse(nil)
}

func GetPostStar(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).String()
	response := app.NewResponse(c)

	userID, _ := c.Get("address")

	postId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		logrus.Errorf("service.GetPostStar err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}

	_, err = service.GetPostStar(postId, userID.(string))
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

	userID, _ := c.Get("address")
	postId, err := primitive.ObjectIDFromHex(param.ID)
	if err != nil {
		logrus.Errorf("service.PostStar err: %v\n", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	status := false
	star, err := service.GetPostStar(postId, userID.(string))
	if err != nil {
		// 创建Star
		_, err = service.CreatePostStar(postId, userID.(string))
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

func GetPostView(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).String()
	response := app.NewResponse(c)
	postId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		logrus.Errorf("service.GetPostView err: %v\n", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(err.Error()))
		return
	}
	view, _ := service.GetPostView(postId)
	response.ToResponse(gin.H{
		"count": view,
	})

}

func PostView(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).String()
	response := app.NewResponse(c)
	postId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		logrus.Errorf("service.PostView err: %v\n", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(err.Error()))
		return
	}
	err = service.CreatePostView(postId)
	if err != nil {
		logrus.Errorf("service.PostView err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	response.ToResponse(gin.H{
		"status": true,
	})
}

func GetPostCollection(c *gin.Context) {
	postID := convert.StrTo(c.Query("id")).String()
	response := app.NewResponse(c)

	userID, _ := c.Get("address")
	postId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		logrus.Errorf("service.GetPostCollection err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	_, err = service.GetPostCollection(postId, userID.(string))
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

	userID, _ := c.Get("address")
	postId, err := primitive.ObjectIDFromHex(param.ID)
	if err != nil {
		logrus.Errorf("service.PostCollection err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	status := false
	collection, err := service.GetPostCollection(postId, userID.(string))
	if err != nil {
		// 创建collection
		_, err = service.CreatePostCollection(postId, userID.(string))
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

func StickPost(c *gin.Context) {
	param := service.PostStickReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	postId, err := primitive.ObjectIDFromHex(param.ID)
	if err != nil {
		logrus.Errorf("service.StickPost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	postFormatted, err := service.GetPost("", postId)
	if err != nil {
		logrus.Errorf("service.GetPost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	user, _ := c.Get("address")
	e := service.CheckIsMyDAO(user.(string), postFormatted.DaoId)
	if e != nil {
		response.ToErrorResponse(e)
		return
	}
	err = service.StickPost(postId)
	if err != nil {
		logrus.Errorf("service.StickPost err: %v\n", err)
		response.ToErrorResponse(errcode.LockPostFailed)
		return
	}

	response.ToResponse(gin.H{
		"top_status": 1 - postFormatted.IsTop,
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
	postId, err := primitive.ObjectIDFromHex(param.ID)
	if err != nil {
		logrus.Errorf("service.VisiblePost err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	user, _ := userFrom(c)
	if err := service.VisiblePost(user, postId, param.Visibility); err != nil {
		logrus.Errorf("service.VisiblePost err: %v\n", err)
		response.ToErrorResponse(err)
		return
	}

	response.ToResponse(gin.H{
		"visibility": param.Visibility,
	})
}

func BlockPost(c *gin.Context) {
	response := app.NewResponse(c)
	postId := c.Param("post_id")
	postID, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		logrus.Errorf("post_id parase err: %v\n", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(err.Error()))
		return
	}
	user, _ := userFrom(c)
	err = service.BlockPost(user, postID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.ToErrorResponse(errcode.NotFound)
			return
		}
		if mongo.IsDuplicateKeyError(err) {
			response.ToResponse(nil)
			return
		}
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(nil)
}

func ComplaintPost(c *gin.Context) {
	param := service.PostComplaintReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, _ := userFrom(c)
	err := service.ComplaintPost(user, param)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.ToErrorResponse(errcode.NotFound)
			return
		}
		if mongo.IsDuplicateKeyError(err) {
			response.ToResponse(nil)
			return
		}
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(nil)
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

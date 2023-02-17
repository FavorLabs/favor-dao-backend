package api

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func getDaos(c *gin.Context) {
	response := app.NewResponse(c)

	q := &core.QueryReq{
		Query: c.Query("query"),
		Type:  "search",
	}
	if c.Query("type") == "address" {
		q.Type = "address"
	}

	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)

	if q.Query == "" && q.Type == "search" {
		resp, err := service.GetIndexPosts(user, offset, limit) // todo where dao
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

func CreateDao(c *gin.Context) {
	param := service.DaoCreationReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	userAddress, _ := c.Get("address")
	dao, err := service.CreateDao(c, userAddress.(string), param)

	if err != nil {
		logrus.Errorf("service.CreateDao err: %v\n", err)
		response.ToErrorResponse(errcode.CreateDaoFailed)
		return
	}

	response.ToResponse(dao)
}

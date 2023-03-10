package api

import (
	"errors"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetDaos(c *gin.Context) {
	response := app.NewResponse(c)

	q := &core.QueryReq{
		Query: c.Query("query"),
		Type:  "search",
	}
	if strings.HasPrefix(q.Query, "0x") {
		q.Type = "address"
	}

	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)

	resp, total := service.GetDaoBookmarkList(user.Address, q, offset, limit)
	response.ToResponseList(resp, total)
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

	_, err := service.GetDaoByName(param.Name)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		response.ToErrorResponse(errcode.CreateDaoNameDuplication)
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

func UpdateDao(c *gin.Context) {
	param := service.DaoUpdateReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	userAddress, _ := c.Get("address")
	err := service.UpdateDao(userAddress.(string), param)

	if err != nil {
		logrus.Errorf("service.UpdateDao err: %v\n", err)
		response.ToErrorResponse(errcode.UpdateDaoFailed)
		return
	}

	response.ToResponse(nil)
}

func GetDao(c *gin.Context) {
	daoId := convert.StrTo(c.Query("dao_id")).String()
	response := app.NewResponse(c)

	dao, err := service.GetDao(daoId)
	if err != nil {
		logrus.Errorf("service.GetDao err: %v\n", err)
		response.ToErrorResponse(errcode.GetDaoFailed)
		return
	}
	response.ToResponse(dao)
}

func GetMyDaoList(c *gin.Context) {
	response := app.NewResponse(c)

	address, _ := c.Get("address")

	dao, _ := service.GetMyDaoList(address.(string))
	response.ToResponseList(dao, int64(len(dao)))
}

func GetDaoBookmark(c *gin.Context) {
	daoId := convert.StrTo(c.Query("dao_id")).String()
	response := app.NewResponse(c)

	address, _ := c.Get("address")

	_, err := service.GetDaoBookmark(address.(string), daoId)
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

func ActionDaoBookmark(c *gin.Context) {
	param := service.DaoFollowReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	address, _ := c.Get("address")

	status := false
	book, err := service.GetDaoBookmark(address.(string), param.DaoID)
	if err != nil {
		// create follow
		_, err = service.CreateDaoBookmark(address.(string), param.DaoID)
		status = true
	} else {
		// cancel follow
		err = service.DeleteDaoBookmark(book)
	}

	if err != nil {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	response.ToResponse(gin.H{
		"status": status,
	})
}

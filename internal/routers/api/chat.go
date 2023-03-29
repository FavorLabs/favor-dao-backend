package api

import (
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetChatGroups(c *gin.Context) {
	response := app.NewResponse(c)

	daoName := c.Query("dao_name")
	if daoName == "" {
		response.ToErrorResponse(errcode.GetDaoFailed)
		return
	}

	address, _ := c.Get("address")
	page := app.GetPage(c)
	perPage := app.GetPageSize(c)

	resp, err := service.ListChatGroups(address.(string), daoName, page, perPage)
	if err != nil {
		logrus.Errorf("service.ListChatGroups err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponseList(resp, int64(len(resp)))
}

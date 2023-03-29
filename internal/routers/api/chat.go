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

	daoId := c.Query("dao_id")
	if daoId == "" {
		response.ToErrorResponse(errcode.GetDaoFailed)
		return
	}

	page := app.GetPage(c)
	perPage := app.GetPageSize(c)

	resp, err := service.ListChatGroups(daoId, page, perPage)
	if err != nil {
		logrus.Errorf("service.ListChatGroups err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponseList(resp, int64(len(resp)))
}

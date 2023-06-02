package api

import (
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NotifyGroupList(c *gin.Context) {
	response := app.NewResponse(c)
	user, _ := userFrom(c)
	offset, limit := app.GetPageOffset(c)

	list, count, err := service.NotifyGroupList(user.ID, limit, offset)
	if err != nil {
		logrus.Errorf("service.NotifyGroupList err: %v\n", err)
		response.ToResponseList([]service.NotifyGroup{}, 0)
		return
	}
	response.ToResponseList(list, count)
}

func NotifyOrgan(c *gin.Context) {
	response := app.NewResponse(c)
	user, _ := userFrom(c)
	to := primitive.NilObjectID
	if user != nil {
		to = user.ID
	}
	list, err1 := service.NotifyOrganList(to)
	if err1 != nil {
		logrus.Errorf("service.NotifyOrganList err: %v\n", err1)
		response.ToResponseList([]model.Msg{}, 0)
		return
	}
	response.ToResponseList(list, int64(len(*list)))
}

func NotifyByFrom(c *gin.Context) {
	response := app.NewResponse(c)
	from := c.Param("fromId")
	fromId, err := primitive.ObjectIDFromHex(from)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	offset, limit := app.GetPageOffset(c)
	user, _ := userFrom(c)

	list, count, err1 := service.NotifyByFrom(fromId, user.ID, limit, offset)
	if err1 != nil {
		logrus.Errorf("service.NotifyByFrom err: %v\n", err1)
		response.ToResponseList([]model.Msg{}, 0)
		return
	}
	response.ToResponseList(list, count)
}

func NotifyUnread(c *gin.Context) {
	response := app.NewResponse(c)

	from := c.Param("fromId")
	fromId, err := primitive.ObjectIDFromHex(from)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	user, _ := userFrom(c)
	count := service.GetNotifyUnread(fromId, user.ID)
	response.ToResponse(count)
}

func NotifySys(c *gin.Context) {
	response := app.NewResponse(c)
	organ := c.Param("organId")
	organId, err := primitive.ObjectIDFromHex(organ)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	offset, limit := app.GetPageOffset(c)
	list, count, err1 := service.NotifySys(organId, limit, offset)
	if err1 != nil {
		logrus.Errorf("service.NotifySys err: %v\n", err)
		response.ToResponseList([]service.NotifyGroup{}, 0)
		return
	}
	response.ToResponseList(list, count)
}

func DeletedNotifyById(c *gin.Context) {
	response := app.NewResponse(c)
	id := c.Param("id")
	Id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	b, err1 := service.DeleteNotifyById(Id)
	if err1 != nil {
		response.ToErrorResponse(err1)
		return
	}
	response.ToResponse(b)
}

func DeletedNotifyByFromId(c *gin.Context) {
	response := app.NewResponse(c)
	from := c.Param("fromId")
	fromId, err := primitive.ObjectIDFromHex(from)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	user, _ := userFrom(c)

	b, err1 := service.DeleteNotifyByFrom(fromId, user.ID)
	if err1 != nil {
		response.ToErrorResponse(err1)
		return
	}
	response.ToResponse(b)
}

func PutNotifyRead(c *gin.Context) {
	response := app.NewResponse(c)
	from := c.Param("fromId")
	fromId, err := primitive.ObjectIDFromHex(from)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	user, _ := userFrom(c)

	b, err1 := service.PutNotifyRead(fromId, user.ID)
	if err1 != nil {
		response.ToErrorResponse(err1)
		return
	}
	response.ToResponse(b)
}

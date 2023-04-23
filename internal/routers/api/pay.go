package api

import (
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func PayNotify(c *gin.Context) {
	response := app.NewResponse(c)
	param := service.PayCallbackParam{}
	err := c.ShouldBindQuery(&param)
	if err != nil {
		logrus.Errorf("app.BindAndValid err: %s", err)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(err.Error()))
		return
	}
	err = service.PayNotify(param)
	if err != nil {
		logrus.Errorf("service.PayNotify err: %s", err)
		response.ToErrorResponse(errcode.PayNotifyError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(nil)
}

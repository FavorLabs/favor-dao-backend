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
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	err := service.PayNotify(param)
	if err != nil {
		logrus.Errorf("service.PayNotify err: %s", err)
		response.ToErrorResponse(errcode.PayNotifyError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(nil)
}

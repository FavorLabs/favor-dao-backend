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
	param := service.Notify{}
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	service.PayNotify(param)
	response.ToResponse(gin.H{"status": true})
}

package api

import (
	"fmt"

	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateRedpacket(c *gin.Context) {
	param := service.RedpacketRequestAuth{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	guessMessage := fmt.Sprintf("%s Create Redpacket at %d", param.Auth.WalletAddr, param.Auth.Timestamp)
	ok, err := service.VerifySignMessage(c.Request.Context(), &param.Auth, guessMessage)
	if err != nil || !ok {
		response.ToErrorResponse(errcode.InvalidWalletSignature)
		return
	}

	user, _ := userFrom(c)

	RedpacketID, err := service.CreateRedpacket(user.Address, param.RedpacketRequest)
	if err != nil {
		if e, ok := err.(*errcode.Error); ok {
			response.ToErrorResponse(e)
			return
		}
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(gin.H{
		"redpacket_id": RedpacketID,
	})
}

func CreateRedpacketTest(c *gin.Context) {
	param := service.RedpacketRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	RedpacketID, err := service.CreateRedpacket(c.Query("user"), param)
	if err != nil {
		if e, ok := err.(*errcode.Error); ok {
			response.ToErrorResponse(e)
			return
		}
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(gin.H{
		"redpacket_id": RedpacketID,
	})
}

func ClaimRedpacket(c *gin.Context) {
	response := app.NewResponse(c)
	rpd := c.Param("redpacket_id")
	rpID, err := primitive.ObjectIDFromHex(rpd)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	user, _ := userFrom(c)
	info, e := service.ClaimRedpacket(c, user.Address, rpID)
	if e != nil {
		response.ToErrorResponse(e)
		return
	}
	response.ToResponse(info)
}

func ClaimRedpacketTest(c *gin.Context) {
	response := app.NewResponse(c)
	rpd := c.Param("redpacket_id")
	rpID, err := primitive.ObjectIDFromHex(rpd)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	info, e := service.ClaimRedpacket(c, c.Query("user"), rpID)
	if e != nil {
		response.ToErrorResponse(e)
		return
	}
	response.ToResponse(info)
}

func RedpacketInfo(c *gin.Context) {
	response := app.NewResponse(c)
	rpd := c.Param("redpacket_id")
	rpID, err := primitive.ObjectIDFromHex(rpd)
	if err != nil {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}
	user, _ := userFrom(c)
	info, e := service.RedpacketInfo(c, rpID, user.Address)
	if e != nil {
		response.ToErrorResponse(errcode.ServerError.WithDetails(e.Error()))
		return
	}
	response.ToResponse(info)
}

func RedpacketStatsClaims(c *gin.Context) {
	response := app.NewResponse(c)
	user, _ := userFrom(c)
	start, end := app.GetYear(c)
	out := service.RedpacketClaimStats(c, service.RedpacketQueryParam{
		StartTime: start,
		EndTime:   end,
		Address:   user.Address,
	})
	response.ToResponse(out)
}

func RedpacketClaimList(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)
	rpd := c.Param("redpacket_id")
	rpID, err := primitive.ObjectIDFromHex(rpd)
	if err != nil {
		response.ToResponseList(make([]interface{}, 0), 0)
		return
	}
	total, list := service.RedpacketClaimList(c, rpID, limit, offset)

	response.ToResponseList(list, total)
}

func RedpacketClaimListForMy(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)
	start, end := app.GetYear(c)
	user, _ := userFrom(c)
	total, list := service.RedpacketClaimListForMy(c, service.RedpacketQueryParam{
		StartTime: start,
		EndTime:   end,
		Address:   user.Address,
	}, limit, offset)

	response.ToResponseList(list, total)
}

func RedpacketStatsSends(c *gin.Context) {
	response := app.NewResponse(c)
	start, end := app.GetYear(c)
	user, _ := userFrom(c)
	out := service.RedpacketSendStats(c, service.RedpacketQueryParam{
		StartTime: start,
		EndTime:   end,
		Address:   user.Address,
	})
	response.ToResponse(out)
}

func RedpacketSendList(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)
	start, end := app.GetYear(c)
	user, _ := userFrom(c)
	total, list := service.RedpacketSendList(c, service.RedpacketQueryParam{
		StartTime: start,
		EndTime:   end,
		Address:   user.Address,
	}, limit, offset)

	response.ToResponseList(list, total)
}

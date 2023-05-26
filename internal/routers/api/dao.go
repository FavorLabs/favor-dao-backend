package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
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
	}
	if strings.HasPrefix(q.Query, "0x") {
		q.Addresses = []string{q.Query}
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

	var outErr *errcode.Error

	dao, err := service.CreateDao(c, userAddress.(string), param, func(ctx context.Context, dao *model.Dao) (string, error) {
		gid, err := service.CreateChatGroup(ctx, dao.Address, dao.ID.Hex(), dao.Name, dao.Avatar, dao.Introduction)
		if err != nil {
			outErr = errcode.CreateChatGroupFailed
			return "", err
		}
		return gid, nil
	})
	if outErr != nil {
		logrus.Errorf("service.CreateDao err: %v\n", err)
		response.ToErrorResponse(outErr)
		return
	}
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
		response.ToErrorResponse(errcode.UpdateDaoFailed.WithDetails(err.Error()))
		return
	}

	response.ToResponse(nil)
}

func GetDao(c *gin.Context) {
	daoId := c.Query("dao_id")
	response := app.NewResponse(c)

	address, _ := c.Get("address")
	if address == nil {
		address = ""
	}
	dao, err := service.GetDaoFormatted(address.(string), daoId)
	if err != nil {
		logrus.Errorf("service.GetDao err: %v\n", err)
		response.ToErrorResponse(errcode.GetDaoFailed)
		return
	}
	response.ToResponse(dao)
}

func GetDaoList(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)

	list, err := service.GetDaoList(&service.DaoListReq{
		Conditions: model.ConditionsT{
			"query": bson.M{
				"type": model.DaoWithURL,
			},
			"ORDER": bson.M{"_id": -1},
		},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		logrus.Errorf("service.GetDaoList err: %v\n", err)
		response.ToErrorResponse(errcode.GetDaoFailed)
		return
	}

	if len(list) == 0 {
		list = make([]*model.Dao, 0)
	}

	response.ToResponseList(list, int64(len(list)))
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
	start := time.Now()
	param := service.DaoFollowReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	logrus.Debugf("ActionDaoBookmark BindAndValid: %s", time.Since(start))
	address, _ := c.Get("address")
	token := c.GetHeader("X-Session-Token")

	status := false
	book, err := service.GetDaoBookmark(address.(string), param.DaoID)
	logrus.Debugf("ActionDaoBookmark service.GetDaoBookmark: %s", time.Since(start))
	if err != nil {
		// create follow
		err = joinDao(address.(string), param.DaoID, token)
		logrus.Debugf("ActionDaoBookmark service.CreateDaoBookmark: %s", time.Since(start))
		status = true
	} else {
		// cancel follow
		err = service.DeleteDaoBookmark(book, func(ctx context.Context, dao *model.Dao) (string, error) {
			logrus.Debugf("ActionDaoBookmark service.JoinOrLeaveGroup leave: %s", time.Since(start))
			resp, err := service.JoinOrLeaveGroup(ctx, dao.ID.Hex(), false, token)
			if err != nil {
				return "", err
			}
			_, err = service.PushDaoToSearch(dao)
			if err != nil {
				return "", err
			}
			return resp, err
		})
		logrus.Debugf("ActionDaoBookmark service.DeleteDaoBookmark: %s", time.Since(start))
	}

	if err != nil {
		logrus.Errorf("api.ActionDaoBookmark err: %s", err)
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	response.ToResponse(gin.H{
		"status": status,
	})
}

func joinDao(address string, daoID string, token string) error {
	_, err := service.CreateDaoBookmark(address, daoID, func(ctx context.Context, dao *model.Dao) (string, error) {
		gid, err := service.JoinOrLeaveGroup(ctx, dao.ID.Hex(), true, token)
		if err != nil {
			return "", err
		}
		_, err = service.PushDaoToSearch(dao)
		if err != nil {
			return "", err
		}
		return gid, err
	})
	return err
}

func SubDao(c *gin.Context) {
	response := app.NewResponse(c)
	daoId := c.Param("dao_id")
	daoID, err := primitive.ObjectIDFromHex(daoId)
	if err != nil {
		logrus.Errorf("service.GetPostCollection err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostFailed)
		return
	}
	param := service.AuthByWalletRequest{}
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	guessMessage := fmt.Sprintf("%s subscribe DAO at %d", param.WalletAddr, param.Timestamp)
	ok, err := service.VerifySignMessage(c.Request.Context(), &param, guessMessage)
	if err != nil || !ok {
		response.ToErrorResponse(errcode.InvalidWalletSignature)
		return
	}
	if service.CheckSubscribeDAO(param.WalletAddr, daoID) {
		response.ToErrorResponse(errcode.AlreadySubscribedDAO)
		return
	}
	if e := service.CheckDAOUser(daoID); e != nil {
		response.ToErrorResponse(e)
		return
	}
	_, status, err := service.SubDao(c.Request.Context(), daoID, param.WalletAddr)
	if err != nil {
		logrus.Errorf("service.SubDao err: %v\n", err)
		response.ToErrorResponse(errcode.SubscribeDAO.WithDetails(err.Error()))
		return
	}
	// join DAO
	_, err = service.GetDaoBookmark(param.WalletAddr, daoId)
	if err != nil {
		token := c.GetHeader("X-Session-Token")
		err = joinDao(param.WalletAddr, daoId, token)
		if err != nil {
			logrus.Errorf("after subsribe DAO, joinDao err: %s", err)
		}
	}
	response.ToResponse(gin.H{
		"status": status,
	})
}

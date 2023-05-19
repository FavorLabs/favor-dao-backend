package api

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"github.com/sirupsen/logrus"
)

func Login(c *gin.Context) {
	param := service.AuthByWalletRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, err := service.DoLoginWallet(c, &param)
	if err != nil {
		logrus.Errorf("service.DoLogin err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	// Create user and auth token by chat
	token, err := service.GetAuthToken(c, user.Address)
	if err != nil {
		logrus.Errorf("service.GenerateToken err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenGenerate)
		return
	}

	session, _ := json.Marshal(service.Session{
		ID:           ulid.Make().String(),
		FriendlyName: c.DefaultQuery("name", "UnknownDevice"),
		WalletAddr:   param.WalletAddr,
	})
	if err := conf.Redis.Set(c, fmt.Sprintf("token_%s", token), session, core.TokenExpiration).Err(); err != nil {
		logrus.Errorf("conf.Redis.Set err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenError)
		return
	}

	response.ToResponse(gin.H{
		"token": token,
	})
}

func DeleteAccount(c *gin.Context) {
	param := service.AuthByWalletRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	err := service.DeleteUser(c, &param)
	if err != nil {
		logrus.Errorf("service.DeleteUser err: %v", err)
		response.ToErrorResponse(errcode.ServerError)
		return
	}

	response.ToResponse(nil)
}

func GetUserInfo(c *gin.Context) {
	param := service.AuthRequest{}
	response := app.NewResponse(c)

	if username, exists := c.Get("address"); exists {
		param.UserAddress = username.(string)
	}

	user, err := service.GetUserInfo(&param)

	if err != nil {
		response.ToErrorResponse(errcode.UnauthorizedAuthNotExist)
		return
	}

	response.ToResponse(gin.H{
		"id":       user.ID,
		"nickname": user.Nickname,
		"address":  user.Address,
		"avatar":   user.Avatar,
	})
}

func GetAccounts(c *gin.Context) {
	response := app.NewResponse(c)

	user, _ := userFrom(c)

	ac, err := service.FindAccounts(c, user.Address)
	if err != nil {
		response.ToErrorResponse(errcode.ServerError.WithDetails(err.Error()))
		return
	}
	response.ToResponse(ac)
}

func GetUserStatistic(c *gin.Context) {
	param := service.AuthRequest{}
	response := app.NewResponse(c)

	if username, exists := c.Get("address"); exists {
		param.UserAddress = username.(string)
	}

	user, err := service.GetUserInfo(&param)

	if err != nil {
		response.ToErrorResponse(errcode.UnauthorizedAuthNotExist)
		return
	}

	response.ToResponse(gin.H{
		"comment_count": service.GetMyCommentCount(user.Address),
		"upvote_count":  service.GetMyPostStartCount(user.Address),
		"dao_count":     service.GetMyDaoMarkCount(user.Address),
	})
}

func ChangeNickname(c *gin.Context) {
	param := service.ChangeNicknameReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user := &model.User{}
	if u, exists := c.Get("USER"); exists {
		user = u.(*model.User)
	}

	if utf8.RuneCountInString(param.Nickname) < 2 || utf8.RuneCountInString(param.Nickname) > 12 {
		response.ToErrorResponse(errcode.NicknameLengthLimit)
		return
	}

	if err := service.ChangeUserName(user, param.Nickname); err != nil {
		response.ToErrorResponse(err)
		return
	}

	response.ToResponse(nil)
}

func ChangeAvatar(c *gin.Context) {
	param := service.ChangeAvatarReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, exist := userFrom(c)
	if !exist {
		response.ToErrorResponse(errcode.UnauthorizedTokenError)
		return
	}

	if err := service.ChangeUserAvatar(user, param.Avatar); err != nil {
		response.ToErrorResponse(err)
		return
	}

	response.ToResponse(nil)
}

func GetUserProfile(c *gin.Context) {
	response := app.NewResponse(c)
	address := c.Query("address")

	user, err := service.GetUserByAddress(address)
	if err != nil {
		logrus.Errorf("service.GetUserByAddress err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistUserAddress)
		return
	}

	response.ToResponse(gin.H{
		"id":       user.ID,
		"nickname": user.Nickname,
		"address":  user.Address,
		"avatar":   user.Avatar,
	})
}

func GetUserPosts(c *gin.Context) {
	response := app.NewResponse(c)
	address := c.Query("address")
	q := parseQueryReq(c)

	if len(q.Type) == 0 {
		q.Type = core.AllQueryPostType
	}
	visibilities := []model.PostVisibleT{model.PostVisitPublic}
	my, ok := userFrom(c)
	if ok && my.Address == address {
		q.Visibility = append(visibilities, model.PostVisitPrivate)
	}
	offset, limit := app.GetPageOffset(c)

	// Contains my private when query address it's me
	posts, totalRows, err := service.GetPostListFromSearch(my, q, offset, limit)
	if err != nil {
		logrus.Errorf("service.GetPostListFromSearch err: %v\n", err)
		response.ToResponseList([]*model.PostFormatted{}, 0)
		return
	}
	response.ToResponseList(posts, totalRows)
}

func GetDaoPosts(c *gin.Context) {
	response := app.NewResponse(c)
	daoId := c.Query("daoId")
	daoInfo, err := service.GetDao(daoId)
	if err != nil {
		logrus.Errorf("service.GetDaoPosts err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistDao)
		return
	}

	q := parseQueryReq(c)

	if len(q.Type) == 0 {
		q.Type = core.AllQueryPostType
	}
	visibilities := []model.PostVisibleT{model.PostVisitPublic}
	my, ok := userFrom(c)
	if ok && my.Address == daoInfo.Address {
		q.Visibility = append(visibilities, model.PostVisitPrivate)
	}
	offset, limit := app.GetPageOffset(c)

	// Contains dao private when query dao it's me
	posts, totalRows, err := service.GetPostListFromSearch(my, q, offset, limit)
	if err != nil {
		logrus.Errorf("service.GetPostListFromSearch err: %v\n", err)
		response.ToResponseList([]*model.PostFormatted{}, 0)
		return
	}
	response.ToResponseList(posts, totalRows)
}

func GetUserCollections(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)
	address, _ := c.Get("address")
	posts, totalRows, err := service.GetUserCollections(address.(string), offset, limit)

	if err != nil {
		logrus.Errorf("service.GetUserCollections err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponseList(posts, totalRows)
}

func GetUserStars(c *gin.Context) {
	response := app.NewResponse(c)
	offset, limit := app.GetPageOffset(c)
	address, _ := c.Get("address")
	posts, totalRows, err := service.GetUserStars(address.(string), offset, limit)
	if err != nil {
		logrus.Errorf("service.GetUserStars err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponseList(posts, totalRows)
}

func GetSuggestUsers(c *gin.Context) {
	keyword := c.Query("k")
	response := app.NewResponse(c)

	usernames, err := service.GetSuggestUsers(keyword)
	if err != nil {
		logrus.Errorf("service.GetSuggestUsers err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponse(usernames)
}

func GetSuggestTags(c *gin.Context) {
	keyword := c.Query("k")
	response := app.NewResponse(c)

	tags, err := service.GetSuggestTags(keyword)
	if err != nil {
		logrus.Errorf("service.GetSuggestTags err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponse(tags)
}

func userFrom(c *gin.Context) (*model.User, bool) {
	if u, exists := c.Get("USER"); exists {
		user, ok := u.(*model.User)
		return user, ok
	}
	return nil, false
}

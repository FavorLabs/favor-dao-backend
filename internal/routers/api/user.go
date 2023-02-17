package api

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"favor-dao-backend/internal/conf"
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

	token, err := app.GenerateToken()
	if err != nil {
		logrus.Errorf("app.GenerateToken err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenGenerate)
		return
	}

	userInfo, _ := json.Marshal(app.UserInfo{
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	})

	if err := conf.Redis.Set(c, fmt.Sprintf("user_%s", param.WalletAddr), userInfo, 0).Err(); err != nil {
		logrus.Errorf("conf.Redis.Set err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenError)
		return
	}

	session, _ := json.Marshal(app.Session{
		ID:           ulid.Make().String(),
		FriendlyName: c.DefaultQuery("name", "UnknownDevice"),
		WalletAddr:   param.WalletAddr,
	})
	if err := conf.Redis.Set(c, fmt.Sprintf("token_%s", token), session, 0).Err(); err != nil {
		logrus.Errorf("conf.Redis.Set err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenError)
		return
	}

	response.ToResponse(gin.H{
		"token": token,
	})
}

func GetUserInfo(c *gin.Context) {
	param := service.AuthRequest{}
	response := app.NewResponse(c)

	if username, exists := c.Get("USERNAME"); exists {
		param.Username = username.(string)
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

	user.Nickname = param.Nickname
	service.UpdateUserInfo(user)

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
	username := c.Query("username")

	user, err := service.GetUserByUsername(username)
	if err != nil {
		logrus.Errorf("service.GetUserByUsername err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistUsername)
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
	username := c.Query("username")

	user, err := service.GetUserByUsername(username)
	if err != nil {
		logrus.Errorf("service.GetUserByUsername err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistUsername)
		return
	}

	visibilities := []model.PostVisibleT{model.PostVisitPublic}
	if u, exists := c.Get("USER"); exists {
		self := u.(*model.User)
		if self.ID == user.ID || self.IsAdmin {
			visibilities = append(visibilities, model.PostVisitPrivate, model.PostVisitFriend)
		} else if service.IsFriend(user.ID, self.ID) {
			visibilities = append(visibilities, model.PostVisitFriend)
		}
	}
	conditions := model.ConditionsT{
		"user_id":         user.ID,
		"visibility IN ?": visibilities,
		"ORDER":           "latest_replied_on DESC",
	}

	posts, err := service.GetPostList(&service.PostListReq{
		Conditions: &conditions,
		Offset:     (app.GetPage(c) - 1) * app.GetPageSize(c),
		Limit:      app.GetPageSize(c),
	})
	if err != nil {
		logrus.Errorf("service.GetPostList err: %v\n", err)
		response.ToErrorResponse(errcode.GetPostsFailed)
		return
	}
	totalRows, _ := service.GetPostCount(&conditions)

	response.ToResponseList(posts, totalRows)
}

func GetUserCollections(c *gin.Context) {
	response := app.NewResponse(c)

	userID, _ := c.Get("UID")
	posts, totalRows, err := service.GetUserCollections(userID.(int64), (app.GetPage(c)-1)*app.GetPageSize(c), app.GetPageSize(c))

	if err != nil {
		logrus.Errorf("service.GetUserCollections err: %v\n", err)
		response.ToErrorResponse(errcode.GetCollectionsFailed)
		return
	}

	response.ToResponseList(posts, totalRows)
}

func GetUserStars(c *gin.Context) {
	response := app.NewResponse(c)

	userID, _ := c.Get("UID")
	posts, totalRows, err := service.GetUserStars(userID.(int64), (app.GetPage(c)-1)*app.GetPageSize(c), app.GetPageSize(c))
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

func userIdFrom(c *gin.Context) (int64, bool) {
	if u, exists := c.Get("UID"); exists {
		uid, ok := u.(int64)
		return uid, ok
	}
	return -1, false
}

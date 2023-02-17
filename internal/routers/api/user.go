package api

import (
	"go.mongodb.org/mongo-driver/bson"
	"unicode/utf8"

	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Login(c *gin.Context) {
	param := service.AuthRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, err := service.DoLogin(c, &param)
	if err != nil {
		logrus.Errorf("service.DoLogin err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	token, err := app.GenerateToken(user)
	if err != nil {
		logrus.Errorf("app.GenerateToken err: %v", err)
		response.ToErrorResponse(errcode.UnauthorizedTokenGenerate)
		return
	}

	response.ToResponse(gin.H{
		"token": token,
	})
}

func Register(c *gin.Context) {

	param := service.RegisterRequest{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	// check user
	err := service.ValidUsername(param.Username)
	if err != nil {
		logrus.Errorf("service.Register err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	// check password
	err = service.CheckPassword(param.Password)
	if err != nil {
		logrus.Errorf("service.Register err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	user, err := service.Register(
		param.Username,
		param.Password,
	)

	if err != nil {
		logrus.Errorf("service.Register err: %v", err)
		response.ToErrorResponse(errcode.UserRegisterFailed)
		return
	}

	response.ToResponse(gin.H{
		"id":       user.ID,
		"username": user.Username,
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
	phone := ""
	if user.Phone != "" && len(user.Phone) == 11 {
		phone = user.Phone[0:3] + "****" + user.Phone[7:]
	}

	response.ToResponse(gin.H{
		"id":       user.ID,
		"nickname": user.Nickname,
		"username": user.Username,
		"status":   user.Status,
		"avatar":   user.Avatar,
		"balance":  user.Balance,
		"phone":    phone,
		"is_admin": user.IsAdmin,
	})
}

func ChangeUserPassword(c *gin.Context) {
	param := service.ChangePasswordReq{}
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

	err := service.CheckPassword(param.Password)
	if err != nil {
		logrus.Errorf("service.Register err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	if !service.ValidPassword(user.Password, param.OldPassword, user.Salt) {
		response.ToErrorResponse(errcode.ErrorOldPassword)
		return
	}

	user.Password, user.Salt = service.EncryptPasswordAndSalt(param.Password)
	service.UpdateUserInfo(user)

	response.ToResponse(nil)
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

func BindUserPhone(c *gin.Context) {
	param := service.UserPhoneBindReq{}
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

	if service.CheckPhoneExist(user.ID, param.Phone) {
		response.ToErrorResponse(errcode.ExistedUserPhone)
		return
	}

	if err := service.CheckPhoneCaptcha(param.Phone, param.Captcha); err != nil {
		logrus.Errorf("service.CheckPhoneCaptcha err: %v\n", err)
		response.ToErrorResponse(err)
		return
	}

	user.Phone = param.Phone
	service.UpdateUserInfo(user)

	response.ToResponse(nil)
}

func ChangeUserStatus(c *gin.Context) {
	param := service.ChangeUserStatusReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	if param.Status != model.UserStatusNormal && param.Status != model.UserStatusClosed {
		response.ToErrorResponse(errcode.InvalidParams)
		return
	}

	user, err := service.GetUserByID(param.ID)
	if err != nil {
		logrus.Errorf("service.GetUserByID err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistUsername)
		return
	}

	user.Status = param.Status
	service.UpdateUserInfo(user)

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
		"username": user.Username,
		"status":   user.Status,
		"avatar":   user.Avatar,
		"is_admin": user.IsAdmin,
	})
}

func GetUserPosts(c *gin.Context) {
	response := app.NewResponse(c)
	address := c.Query("address")

	// todo address !
	user, err := service.GetUserByUsername(address)
	if err != nil {
		logrus.Errorf("service.GetUserByAddress err: %v\n", err)
		response.ToErrorResponse(errcode.NoExistUsername)
		return
	}

	visibilities := []model.PostVisibleT{model.PostVisitPublic}
	if u, exists := c.Get("USER"); exists {
		self := u.(*model.User)
		if self.Address == user.Address {
			visibilities = append(visibilities, model.PostVisitPrivate)
		}
	}
	conditions := model.ConditionsT{
		"query": bson.M{"address": user.Address, "visibility": bson.M{"$in": visibilities}},
		"ORDER": bson.M{"latest_replied_on": -1},
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

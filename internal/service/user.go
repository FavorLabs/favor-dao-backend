package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type PhoneCaptchaReq struct {
	ImgCaptcha   string `json:"img_captcha" form:"img_captcha" binding:"required"`
	ImgCaptchaID string `json:"img_captcha_id" form:"img_captcha_id" binding:"required"`
}

type AuthRequest struct {
	UserAddress string `json:"user_address" form:"user_address" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type ChangePasswordReq struct {
	Password    string `json:"password" form:"password" binding:"required"`
	OldPassword string `json:"old_password" form:"old_password" binding:"required"`
}

type ChangeNicknameReq struct {
	Nickname string `json:"nickname" form:"nickname" binding:"required"`
}

type ChangeAvatarReq struct {
	Avatar string `json:"avatar" form:"avatar" binding:"required"`
}

type ChangeUserStatusReq struct {
	ID     int64 `json:"id" form:"id" binding:"required"`
	Status int   `json:"status" form:"status" binding:"required"`
}

const LOGIN_ERR_KEY = "DaoUserLoginErr"
const MAX_LOGIN_ERR_TIMES = 10

// DoLoginWallet Wallet login, register if user does not exist
func DoLoginWallet(ctx *gin.Context, param *AuthByWalletRequest) (*model.User, error) {
	created := false

	user, err := ds.GetUserByAddress(param.WalletAddr)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errcode.ServerError
		}
		// if not exists
		created = true
	}

	if !created && user.DeletedOn > 0 {
		return nil, errcode.WaitForDelete
	}

	if created || !user.ID.IsZero() {
		if errTimes, err := conf.Redis.Get(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr)).Result(); err == nil {
			if convert.StrTo(errTimes).MustInt() >= MAX_LOGIN_ERR_TIMES {
				return nil, errcode.TooManyLoginError
			}
		}

		guessMessage := fmt.Sprintf("%s login FavorDAO at %d", param.WalletAddr, param.Timestamp)
		ok, err := verifySignMessage(ctx, param, guessMessage)
		if err != nil {
			return nil, err
		}
		if ok {
			if created {
				user, err = ds.CreateUser(&model.User{
					Nickname: param.WalletAddr[:10],
					Address:  param.WalletAddr,
					Avatar:   GetRandomAvatar(),
					LoginAt:  time.Now().Unix(),
				}, func(ctx context.Context, user *model.User) error {
					return CreateChatUser(ctx, user.Address, user.Nickname, user.Avatar)
				})
				if err != nil {
					return nil, err
				}
			} else {
				err = UpdateLoginAt(user)
				if err != nil {
					return nil, err
				}
			}

			// Clear Login Count
			conf.Redis.Del(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr))
			return user, nil
		}
		// Clear Login Err Count
		_, err = conf.Redis.Incr(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr)).Result()
		if err == nil {
			_ = conf.Redis.Expire(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr), time.Hour).Err()
		}

		return nil, errcode.UnauthorizedAuthFailed
	}

	return nil, errcode.UnauthorizedAuthNotExist
}

func DeleteUser(ctx context.Context, param *AuthByWalletRequest) error {
	user, err := ds.GetUserByAddress(param.WalletAddr)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}

		return err
	}

	uid := userId(user.Address)

	guessMessage := fmt.Sprintf("delete %s account at %d", param.WalletAddr, param.Timestamp)
	ok, err := verifySignMessage(ctx, param, guessMessage)
	if err != nil {
		return err
	}

	if !ok {
		return errcode.NoPermission
	}

	// delete auth token
	tokens, err := chat.Scoped().Context(ctx).Users().AuthToken(uid).List()
	if err != nil {
		return err
	}

	if len(tokens) > 0 {
		for _, token := range tokens {
			_, err = chat.Scoped().Context(ctx).Users().AuthToken(uid).Delete(token.AuthToken)
			if err != nil {
				return err
			}

			err = conf.Redis.Del(ctx, fmt.Sprintf("token_%s", token.AuthToken)).Err()
			if err != nil {
				return err
			}
		}
	}

	return ds.DeleteUser(user)
}

func GetUserInfo(param *AuthRequest) (*model.User, error) {
	return GetUserByAddress(param.UserAddress)
}

func GetUserByAddress(address string) (*model.User, error) {
	user, err := ds.GetUserByAddress(address)

	if err != nil {
		return nil, err
	}

	if !user.ID.IsZero() {
		return user, nil
	}

	return nil, errcode.NoExistUserAddress
}

func UpdateUserInfo(user *model.User) *errcode.Error {
	if err := ds.UpdateUser(user, func(ctx context.Context, user *model.User) error {
		return UpdateChatUser(ctx, user.Address, user.Nickname, user.Avatar)
	}); err != nil {
		return errcode.ServerError
	}
	return nil
}

func UpdateLoginAt(user *model.User) error {
	user.LoginAt = time.Now().Unix()
	if err := ds.UpdateUser(user, func(ctx context.Context, user *model.User) error {
		return nil
	}); err != nil {
		return errcode.ServerError
	}
	return nil
}

func ChangeUserName(user *model.User, nickname string) *errcode.Error {
	user.Nickname = nickname
	return UpdateUserInfo(user)
}

func ChangeUserAvatar(user *model.User, avatar string) (err *errcode.Error) {
	defer func() {
		if err != nil {
			deleteOssObjects([]string{avatar})
		}
	}()

	user.Avatar = avatar
	return UpdateUserInfo(user)
}

func GetUserCollections(userAddress string, offset, limit int) ([]*model.PostFormatted, int64, error) {
	collections, err := ds.GetUserPostCollections(userAddress, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	totalRows, err := ds.GetUserPostCollectionCount(userAddress)
	if err != nil {
		return nil, 0, err
	}
	var posts []*model.Post
	for _, collection := range collections {
		posts = append(posts, collection.Post)
	}
	postsFormatted, err := ds.MergePosts(posts)
	if err != nil {
		return nil, 0, err
	}

	return postsFormatted, totalRows, nil
}

func GetUserStars(userAddress string, offset, limit int) ([]*model.PostFormatted, int64, error) {
	stars, err := ds.GetUserPostStars(userAddress, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	totalRows, err := ds.GetUserPostStarCount(userAddress)
	if err != nil {
		return nil, 0, err
	}

	var posts []*model.Post
	for _, star := range stars {
		posts = append(posts, star.Post)
	}
	postsFormatted, err := ds.MergePosts(posts)
	if err != nil {
		return nil, 0, err
	}

	return postsFormatted, totalRows, nil
}

// GetSuggestUsers Get user recommendations based on keywords
func GetSuggestUsers(keyword string) ([]string, error) {
	users, err := ds.GetUsersByKeyword(keyword)
	if err != nil {
		return nil, err
	}

	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user.Nickname)
	}

	return usernames, nil
}

// GetSuggestTags Get tag recommendations based on keywords
func GetSuggestTags(keyword string) ([]string, error) {
	tags, err := ds.GetTagsByKeyword(keyword)
	if err != nil {
		return nil, err
	}

	var ts []string
	for _, t := range tags {
		ts = append(ts, t.Tag)
	}

	return ts, nil
}

func IsFriend(userAddress, friendAddress string) bool {
	return ds.IsFriend(userAddress, friendAddress)
}

func checkPermission(user *model.User, targetUserAddress string) *errcode.Error {
	if user == nil || (user.Address != targetUserAddress) {
		return errcode.NoPermission
	}
	return nil
}

func GetMyPostStartCount(address string) int64 {
	return ds.GetMyPostStartCount(address)
}

func GetMyDaoMarkCount(address string) int64 {
	return ds.GetMyDaoMarkCount(address)
}

func GetMyCommentCount(address string) int64 {
	return ds.GetMyCommentCount(address)
}

func Cancellation(address string) (err error) {
	ctx := context.TODO()
	// delete user for chat
	err = DeleteChatUser(ctx, address)
	if err != nil {
		return err
	}
	// cancel follow DAO
	daoBookmarks := GetDaoBookmarkByAddress(address)
	for _, v := range daoBookmarks {
		err = DeleteDaoBookmark(v, func(ctx context.Context, dao *model.Dao) (string, error) {
			return GetGroupID(dao.ID.Hex()), nil
		})
		if err != nil {
			return err
		}
	}

	// delete DAO
	err = ds.RealDeleteDAO(address, func(ctx context.Context, dao *model.Dao) (string, error) {
		daoId := dao.ID.Hex()
		gid := GetGroupID(daoId)
		err = DeleteGroup(ctx, daoId)
		if err != nil {
			logrus.Errorf("Cancellation chat.DeleteGroup daoID %s: %s", daoId, err)
			return gid, err
		}
		err = DeleteSearchDao(dao)
		if err != nil {
			logrus.Errorf("Cancellation delete daoID %s from search err: %v", daoId, err)
		}
		return gid, err
	})
	if err != nil {
		return err
	}
	// delete post
	err = ds.RealDeletePosts(address, func(ctx context.Context, post *model.Post) (string, error) {
		err = DeleteSearchPost(post)
		if err != nil {
			logrus.Errorf("Cancellation delete postID %s from search err: %v", post.ID.Hex(), err)
		}
		return "", err
	})
	if err != nil {
		return err
	}
	return ds.Cancellation(ctx, address)
}

func CancellationTask() {
	interval := conf.ServerSetting.CancellationTimeInterval
	if interval.Minutes() < 1 {
		return
	}
	tick := time.NewTicker(interval)
	for {
		select {
		case <-tick.C:
			addresses, err := ds.GetCancellationUsers()
			if err != nil {
				logrus.Errorf("CancellationTask GetCancellationUsers %s", err)
				break
			}
			for _, v := range addresses {
				err = Cancellation(v)
				if err != nil {
					logrus.Errorf("CancellationTask Cancellation address %s %s", v, err)
				}
			}
			tick.Reset(interval)
		}
	}
}

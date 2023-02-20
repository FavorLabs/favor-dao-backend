package service

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PhoneCaptchaReq struct {
	ImgCaptcha   string `json:"img_captcha" form:"img_captcha" binding:"required"`
	ImgCaptchaID string `json:"img_captcha_id" form:"img_captcha_id" binding:"required"`
}

type AuthRequest struct {
	UserAddress string `json:"user_address" form:"user_address" binding:"required"`
}

type AuthByWalletRequest struct {
	Timestamp  int64  `json:"timestamp" binding:"required"`
	WalletAddr string `json:"wallet_addr" binding:"required"`
	Signature  string `json:"signature" binding:"required"`
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
	signature, err := hex.DecodeString(param.Signature)
	if err != nil {
		return nil, errcode.InvalidParams
	}
	walletBytes, err := hex.DecodeString(param.WalletAddr)
	if err != nil {
		return nil, errcode.InvalidParams
	}

	created := false

	user, err := ds.GetUserByAddress(param.WalletAddr)
	if err != nil {
		// return nil, errcode.UnauthorizedAuthFailed
		// if not exists
		created = true
	}

	if created || !user.ID.IsZero() {
		if errTimes, err := conf.Redis.Get(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr)).Result(); err == nil {
			if convert.StrTo(errTimes).MustInt() >= MAX_LOGIN_ERR_TIMES {
				return nil, errcode.TooManyLoginError
			}
		}

		// check valid timestamp
		if time.Now().After(time.Unix(param.Timestamp, 0).Add(time.Minute)) {
			return nil, errcode.UnauthorizedTokenTimeout
		}

		// parse message
		guessMessage := fmt.Sprintf("%s login FavorTube at %d", param.WalletAddr, param.Timestamp)
		ethMessage := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(guessMessage), guessMessage))
		pubkey, err := crypto.Ecrecover(crypto.Keccak256(ethMessage), signature)
		if err == nil {
			var signer common.Address
			copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
			if bytes.Equal(walletBytes, signer.Bytes()) {
				if created {
					user := &model.User{
						Nickname: param.WalletAddr[:10],
						Address:  param.WalletAddr,
						Avatar:   GetRandomAvatar(),
					}

					user, err := ds.CreateUser(user)
					if err != nil {
						return nil, err
					}
				}

				// Clear Login Count
				conf.Redis.Del(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr))
				return user, nil
			}
		}
		// Clear Login Err Count
		_, err = conf.Redis.Incr(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr)).Result()
		if err == nil {
			conf.Redis.Expire(ctx, fmt.Sprintf("%s:%s", LOGIN_ERR_KEY, param.WalletAddr), time.Hour).Result()
		}

		return nil, errcode.UnauthorizedAuthFailed
	}

	return nil, errcode.UnauthorizedAuthNotExist
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
	if err := ds.UpdateUser(user); err != nil {
		return errcode.ServerError
	}
	return nil
}

func ChangeUserAvatar(user *model.User, avatar string) (err *errcode.Error) {
	defer func() {
		if err != nil {
			deleteOssObjects([]string{avatar})
		}
	}()

	if err := ds.CheckAttachment(avatar); err != nil {
		return errcode.InvalidParams
	}

	if err := oss.PersistObject(oss.ObjectKey(avatar)); err != nil {
		logrus.Errorf("service.ChangeUserAvatar persist object failed: %s", err)
		return errcode.ServerError
	}

	user.Avatar = avatar
	return UpdateUserInfo(user)
}

func GetUserCollections(userAddress string, offset, limit int) ([]*model.PostFormated, int64, error) {
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

func GetUserStars(userAddress string, offset, limit int) ([]*model.PostFormated, int64, error) {
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

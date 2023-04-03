package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	unipass_sigverify "github.com/unipassid/unipass-sigverify-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type PhoneCaptchaReq struct {
	ImgCaptcha   string `json:"img_captcha" form:"img_captcha" binding:"required"`
	ImgCaptchaID string `json:"img_captcha_id" form:"img_captcha_id" binding:"required"`
}

type AuthRequest struct {
	UserAddress string `json:"user_address" form:"user_address" binding:"required"`
}

type WalletType string

const (
	WalletConnect WalletType = "wallet_connect"
	MetaMask      WalletType = "meta_mask"
	OKX           WalletType = "okx"
	Unipass_Std   WalletType = "unipass_std"
	Unipass_eth   WalletType = "unipass_eth"
)

type AuthByWalletRequest struct {
	Timestamp  int64      `json:"timestamp" binding:"required"`
	WalletAddr string     `json:"wallet_addr" binding:"required"`
	Signature  string     `json:"signature" binding:"required"`
	Type       WalletType `json:"type" binding:"required"`
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
	walletBytes, err := hexutil.Decode(param.WalletAddr)
	if err != nil {
		return nil, errcode.InvalidParams
	}
	signature, err := hexutil.Decode(param.Signature)
	if err != nil {
		return nil, errcode.InvalidParams
	}

	created := false

	user, err := ds.GetUserByAddress(param.WalletAddr)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errcode.ServerError
		}
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
		if time.Now().After(time.UnixMilli(param.Timestamp).Add(time.Minute)) {
			return nil, errcode.UnauthorizedTokenTimeout
		}

		var ok bool

		// parse message
		guessMessage := fmt.Sprintf("%s login FavorDAO at %d", param.WalletAddr, param.Timestamp)
		switch param.Type {
		case WalletConnect, MetaMask, OKX:
			ethMessage := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(guessMessage), guessMessage))
			// Convert to btcec input format with 'recovery id' v at the beginning.
			btcsig := make([]byte, 65)
			btcsig[0] = signature[64]
			copy(btcsig[1:], signature)
			rawKey, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, crypto.Keccak256(ethMessage))
			if err == nil {
				pubkey := (*ecdsa.PublicKey)(rawKey)
				pubBytes := elliptic.Marshal(btcec.S256(), pubkey.X, pubkey.Y)
				var signer common.Address
				copy(signer[:], crypto.Keccak256(pubBytes[1:])[12:])
				ok = bytes.Equal(walletBytes, signer.Bytes())
			}
		case Unipass_Std:
			ok, _ = unipass_sigverify.VerifyMessageSignature(ctx, common.BytesToAddress(walletBytes), []byte(guessMessage), signature, false, eth)
		case Unipass_eth:
			ok, _ = unipass_sigverify.VerifyMessageSignature(ctx, common.BytesToAddress(walletBytes), []byte(guessMessage), signature, true, eth)
		default:
			return nil, errcode.InvalidParams
		}
		if ok {
			if created {
				user, err = ds.CreateUser(&model.User{
					Nickname: param.WalletAddr[:10],
					Address:  param.WalletAddr,
					Avatar:   GetRandomAvatar(),
				}, func(ctx context.Context, user *model.User) error {
					return CreateChatUser(ctx, user.Address, user.Nickname, user.Avatar)
				})
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

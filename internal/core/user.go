package core

import (
	"context"

	"favor-dao-backend/internal/model"
)

type UserManageService interface {
	GetUserByAddress(address string) (*model.User, error)
	GetUsersByAddresses(addresses []string) ([]*model.User, error)
	GetUsersByKeyword(keyword string) ([]*model.User, error)
	CreateUser(user *model.User, chatAction func(context.Context, *model.User) error) (*model.User, error)
	UpdateUser(user *model.User, chatAction func(context.Context, *model.User) error) error
	IsFriend(userAddress, friendAddress string) bool
	GetMyPostStartCount(address string) int64
	GetMyDaoMarkCount(address string) int64
	GetMyCommentCount(address string) int64
}

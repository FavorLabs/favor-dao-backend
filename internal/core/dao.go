package core

import "favor-dao-backend/internal/model"

type DaoManageService interface {
	GetUserByAddress(address string) (*model.User, error)
	GetUsersByAddresses(addresses []string) ([]*model.User, error)
	GetUsersByKeyword(keyword string) ([]*model.User, error)
	CreateUser(user *model.User) (*model.User, error)
	UpdateUser(user *model.User) error
	IsFriend(userAddress, friendAddress string) bool
}

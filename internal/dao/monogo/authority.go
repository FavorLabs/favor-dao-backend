package monogo

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.AuthorizationManageService = (*authorizationManageServant)(nil)
)

type authorizationManageServant struct {
	db *mongo.Database
}

func (s *authorizationManageServant) IsAllow(user *model.User, action *core.Action) bool {
	isFriend := s.isFriend(user.Address, action.UserAddress)
	// TODO: just use defaut act authorization chek rule now
	return action.Act.IsAllow(user, action.UserAddress, isFriend, true)
}

func (s *authorizationManageServant) BeFriendFilter(userAddress string) core.FriendFilter {
	// just empty now
	return core.FriendFilter{}
}

func (s *authorizationManageServant) BeFriendIds(userAddress string) ([]string, error) {
	// just empty now
	return []string{}, nil
}

func (s *authorizationManageServant) isFriend(userAddress, friendAddress string) bool {
	// just true now
	return true
}

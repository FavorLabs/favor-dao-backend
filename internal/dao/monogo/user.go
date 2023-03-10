package monogo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.UserManageService = (*userManageServant)(nil)
)

type userManageServant struct {
	db *mongo.Database
}

func newUserManageService(db *mongo.Database) core.UserManageService {
	return &userManageServant{
		db: db,
	}
}

func (s *userManageServant) GetUserByAddress(address string) (*model.User, error) {
	user := &model.User{
		Address: address,
	}
	return user.Get(context.TODO(), s.db)
}

func (s *userManageServant) GetUsersByAddresses(addresses []string) ([]*model.User, error) {
	user := &model.User{}
	return user.List(s.db, &model.ConditionsT{
		"query": bson.M{"address": bson.M{"$in": addresses}},
	}, 0, 0)
}

func (s *userManageServant) GetUsersByKeyword(keyword string) ([]*model.User, error) {
	user := &model.User{}
	keyword = strings.Trim(keyword, " ")
	return user.FindListByKeyword(context.TODO(), s.db, keyword, 0, 6)
}

func (s *userManageServant) GetTagsByKeyword(keyword string) ([]*model.Tag, error) {
	tag := &model.Tag{}
	keyword = strings.Trim(keyword, " ")
	return tag.FindListByKeyword(context.TODO(), s.db, keyword, 0, 6)
}

func (s *userManageServant) CreateUser(user *model.User) (*model.User, error) {
	return user.Create(context.TODO(), s.db)
}

func (s *userManageServant) UpdateUser(user *model.User) error {
	return user.Update(context.TODO(), s.db)
}

func (s *userManageServant) IsFriend(userAddress string, friendAddress string) bool {
	// just true now
	return true
}

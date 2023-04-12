package monogo

import (
	"context"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/util"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
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

func (s *userManageServant) CreateUser(user *model.User, chatAction func(context.Context, *model.User) error) (*model.User, error) {
	err := util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		newUser, err := user.Create(ctx, s.db)
		if err != nil {
			return err
		}
		err = chatAction(ctx, newUser)
		if err != nil {
			return err
		}
		user = newUser
		return nil
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userManageServant) UpdateUser(user *model.User, chatAction func(context.Context, *model.User) error) error {
	return util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		err := user.Update(ctx, s.db)
		if err != nil {
			return err
		}
		return chatAction(ctx, user)
	})
}

func (s *userManageServant) DeleteUser(user *model.User) error {
	return user.Delete(context.TODO(), s.db)
}

func (s *userManageServant) IsFriend(userAddress string, friendAddress string) bool {
	// just true now
	return true
}

func (s *userManageServant) GetMyPostStartCount(address string) int64 {
	start := &model.PostStar{}
	total, err := start.CountByAddress(s.db, address)
	if err != nil {
		logrus.Errorf("userManageServant.GetMyPostStartCount err: %s", err)
		return 0
	}
	return total
}

func (s *userManageServant) GetMyDaoMarkCount(address string) int64 {
	dao := &model.DaoBookmark{Address: address}
	total := dao.CountMark(context.TODO(), s.db)
	return total
}

func (s *userManageServant) GetMyCommentCount(address string) int64 {
	conditions := &model.ConditionsT{
		"query": bson.M{"address": address},
	}

	count, err := (&model.Comment{}).Count(s.db, conditions)
	if err != nil {
		logrus.Errorf("userManageServant.GetMyCommentCount err: %s", err)
		return 0
	}
	return count
}

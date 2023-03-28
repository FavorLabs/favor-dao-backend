package monogo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"

	corechat "favor-dao-backend/internal/core/chat"
	"favor-dao-backend/internal/model/chat"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ corechat.ManageService = (*chatManageServant)(nil)
)

type chatManageServant struct {
	db *mongo.Database
}

func newChatManageServant(db *mongo.Database) corechat.ManageService {
	return &chatManageServant{
		db: db,
	}
}

func (c *chatManageServant) LinkDao(m *chat.Group) (*chat.Group, error) {
	return m.Create(context.TODO(), c.db)
}

func (c *chatManageServant) UnlinkDao(m *chat.Group) error {
	return m.Delete(context.TODO(), c.db)
}

func (c *chatManageServant) FindGroupByDao(daoId string) (*chat.Group, error) {
	daoIdHex, err := primitive.ObjectIDFromHex(daoId)
	if err != nil {
		return nil, err
	}
	group := &chat.Group{
		DaoID: daoIdHex,
	}
	return group.RelatedDao(context.TODO(), c.db)
}

package monogo

import (
	"context"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.MsgMangerService = (*msgManageService)(nil)
)

type msgManageService struct {
	db *mongo.Database
}

func newMsgManageService(db *mongo.Database) core.MsgMangerService {
	return &msgManageService{
		db: db,
	}
}

func (m msgManageService) GetMsgById(id primitive.ObjectID) (*model.Msg, error) {
	msg := &model.Msg{
		ID: id,
	}
	return msg.Get(context.TODO(), m.db)
}

func (m msgManageService) ListMsg(from, to primitive.ObjectID, pageSize, pageNum int) (*[]model.Msg, error) {
	conditions := &model.ConditionsT{
		"query": bson.M{
			"send.from": from,
			"send.to":   to,
		},
		"ORDER": bson.M{"_id": -1},
	}
	msg := &model.Msg{}
	return msg.List(m.db, conditions, pageNum, pageSize)
}

func (m msgManageService) DeleteMsg(id primitive.ObjectID) (bool, error) {
	conditions := &model.ConditionsT{
		"query": bson.M{
			"_id": id,
		},
	}
	msg := &model.Msg{}
	return true, msg.Delete(m.db, conditions)
}

func (m msgManageService) CountMsg(from, to primitive.ObjectID) (int64, error) {
	conditions := &model.ConditionsT{
		"query": bson.M{
			"send.from": from,
			"send.to":   to,
		},
	}
	msg := &model.Msg{}
	return msg.Count(m.db, conditions)
}

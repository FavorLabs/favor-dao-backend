package monogo

import (
	"context"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

var (
	_ core.MsgReadMangerService = (*msgReadManageService)(nil)
)

type msgReadManageService struct {
	db *mongo.Database
}

func newMsgReadMangerService(db *mongo.Database) core.MsgReadMangerService {
	return &msgReadManageService{
		db: db,
	}
}

func (m msgReadManageService) CreateMsgRead(from, to primitive.ObjectID) (*model.MsgRead, error) {
	mr := model.MsgRead{From: from, To: to, ReadAt: time.Now().Unix(), CreatedAt: time.Now().Unix()}
	return mr.Create(context.TODO(), m.db)
}

func (m msgReadManageService) UpdateReadAt(mr *model.MsgRead) (bool, error) {
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": mr.From,
			"to":   mr.To,
		},
	}

	return true, mr.Update(m.db, conditions)
}

func (m msgReadManageService) DeleteMsgRead(from, to primitive.ObjectID) (bool, error) {
	mr := model.MsgRead{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
			"to":   to,
		},
	}
	return true, mr.Delete(m.db, conditions)
}

func (m msgReadManageService) GetMsgRead(from, to primitive.ObjectID) (*model.MsgRead, error) {
	mr := model.MsgRead{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
			"to":   to,
		},
	}
	return mr.Get(m.db, conditions)
}

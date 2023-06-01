package monogo

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.MsgSendMangerService = (*msgSendManageService)(nil)
)

type msgSendManageService struct {
	db *mongo.Database
}

func newMsgSendMangerService(db *mongo.Database) core.MsgSendMangerService {
	return &msgSendManageService{
		db: db,
	}
}

func (m msgSendManageService) GetMsgSendByMsgId(msgId primitive.ObjectID) (*model.MsgSend, error) {
	ms := &model.MsgSend{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"msgID": msgId,
		},
	}
	return ms.Get(m.db, conditions)
}

func (m msgSendManageService) GetMsgSend(from, to primitive.ObjectID) (*model.MsgSend, error) {
	ms := &model.MsgSend{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
			"to":   to,
		},
	}

	return ms.Get(m.db, conditions)

}

func (m msgSendManageService) GetLastMsg(from, to primitive.ObjectID) (*model.MsgSend, error) {
	ms := &model.MsgSend{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
			"to":   to,
		},
		"ORDER": bson.M{"_id": -1},
	}
	return ms.GetLast(m.db, conditions)
}

func (m msgSendManageService) ListMsgSend(to primitive.ObjectID, froms *[]primitive.ObjectID,
	pageSize, pageNum int) (*[]model.MsgSendGroup, error) {
	ms := &model.MsgSend{}
	return ms.List(m.db, to, froms, pageNum, pageSize)
}

func (m msgSendManageService) DeleteMsgSendByMsgId(msgId primitive.ObjectID) (bool, error) {
	ms := &model.MsgSend{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"msgID": msgId,
		},
	}
	return true, ms.Delete(m.db, conditions)
}

func (m msgSendManageService) CountMsgSend(to primitive.ObjectID, froms *[]primitive.ObjectID) (int64, error) {
	ms := &model.MsgSend{}
	return ms.CountGroup(m.db, to, froms)
}

func (m msgSendManageService) CountUnreadMsg(from, to primitive.ObjectID, date int64) (int64, error) {
	ms := &model.MsgSend{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from":      from,
			"to":        to,
			"createdAt": bson.M{"$gt": date},
		},
	}
	return ms.Count(m.db, conditions)
}

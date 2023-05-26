package monogo

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.MsgSysMangerService = (*msgSysManageService)(nil)
)

type msgSysManageService struct {
	db *mongo.Database
}

func newMsgSysMangerService(db *mongo.Database) core.MsgSysMangerService {
	return &msgSysManageService{
		db: db,
	}
}

func (m msgSysManageService) GetMsgSysByMsgId(msgId primitive.ObjectID) (*model.MsgSys, error) {

	msgSys := &model.MsgSys{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"_id": msgId,
		},
	}
	return msgSys.Get(m.db, conditions)
}

func (m msgSysManageService) ListMsgSys(from primitive.ObjectID, pageSize, pageNum int) (*[]model.MsgSys, error) {
	msgSys := &model.MsgSys{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
		},
	}
	return msgSys.List(m.db, conditions, pageNum, pageSize)
}

func (m msgSysManageService) CountMsgSys(from primitive.ObjectID) (int64, error) {
	msgSys := &model.MsgSys{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"from": from,
		},
	}
	return msgSys.Count(m.db, conditions)
}

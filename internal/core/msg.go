package core

import (
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MsgMangerService interface {
	GetMsgById(id primitive.ObjectID) (*model.Msg, error)
	ListMsg(from, to primitive.ObjectID, pageSize, pageNum int) (*[]model.Msg, error)
	DeleteMsg(id primitive.ObjectID) (bool, error)
	CountMsg(from, to primitive.ObjectID) (int64, error)
}

type MsgReadMangerService interface {
	UpdateReadAt(from, to primitive.ObjectID) (bool, error)
	DeleteMsgRead(from, to primitive.ObjectID) (bool, error)
	GetMsgRead(from, to primitive.ObjectID) (*model.MsgRead, error)
	CreateMsgRead(from, to primitive.ObjectID) (*model.MsgRead, error)
}

type MsgSendMangerService interface {
	GetMsgSendByMsgId(msgId primitive.ObjectID) (*model.MsgSend, error)
	GetMsgSend(from, to primitive.ObjectID) (*model.MsgSend, error)
	GetLastMsg(from, to primitive.ObjectID) (*model.MsgSend, error)
	ListMsgSend(to primitive.ObjectID, froms *[]primitive.ObjectID, pageSize, pageNum int) (*[]model.MsgSendGroup, error)
	DeleteMsgSendByMsgId(msgId primitive.ObjectID) (bool, error)
	CountMsgSend(to primitive.ObjectID, froms *[]primitive.ObjectID) (int64, error)
	CountUnreadMsg(from, to primitive.ObjectID, date int64) (int64, error)
}

type MsgSysMangerService interface {
	GetMsgSysByMsgId(msgId primitive.ObjectID) (*model.MsgSys, error)
	ListMsgSys(from primitive.ObjectID, pageSize, pageNum int) (*[]model.MsgSys, error)
	CountMsgSys(from primitive.ObjectID) (int64, error)
}

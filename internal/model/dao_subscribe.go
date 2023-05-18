package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DaoSubscribeT uint8

const (
	DaoSubscribeSubmit DaoSubscribeT = iota
	DaoSubscribeSuccess
	DaoSubscribeFailed
	DaoSubscribeRefund
)

type DaoSubscribe struct {
	DefaultModel `bson:",inline"`
	Address      string             `json:"address"          bson:"address"`
	DaoID        primitive.ObjectID `json:"dao_id"           bson:"dao_id"`
	TxID         string             `json:"tx_id"            bson:"tx_id"`
	PayAmount    string             `json:"pay_amount"       bson:"pay_amount"`
	Status       DaoSubscribeT      `json:"status"           bson:"status"`
}

func (m *DaoSubscribe) Table() string {
	return "dao_subscribe"
}

func (m *DaoSubscribe) Create(ctx context.Context, db *mongo.Database) error {
	return create(ctx, db, m)
}

func (m *DaoSubscribe) Update(ctx context.Context, db *mongo.Database) error {
	return update(ctx, db, m)
}

func (m *DaoSubscribe) Get(ctx context.Context, db *mongo.Database) error {
	filter := bson.M{ID: m.GetID()}

	return findOne(ctx, db, m, filter)
}

func (m *DaoSubscribe) FindOne(ctx context.Context, db *mongo.Database, filter interface{}) error {
	return findOne(ctx, db, m, filter)
}

func (m *DaoSubscribe) FindList(ctx context.Context, db *mongo.Database, filter interface{}) (list []*DaoSubscribe) {
	cursor, err := db.Collection(m.Table()).Find(ctx, filter)
	if err != nil {
		return
	}
	list = []*DaoSubscribe{}
	for cursor.Next(context.TODO()) {
		var t DaoSubscribe
		if cursor.Decode(&t) != nil {
			return
		}
		list = append(list, &t)
	}
	return
}

package model

import (
	"context"
	"time"

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
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	Address    string             `json:"address"          bson:"address"`
	DaoID      primitive.ObjectID `json:"dao_id"           bson:"dao_id"`
	TxID       string             `json:"tx_id"            bson:"tx_id"`
	PayAmount  int64              `json:"pay_amount"       bson:"pay_amount"`
	Status     DaoSubscribeT      `json:"status"           bson:"status"`
}

func (m *DaoSubscribe) Table() string {
	return "dao_subscribe"
}

func (m *DaoSubscribe) Create(ctx context.Context, db *mongo.Database) (*DaoSubscribe, error) {
	now := time.Now().Unix()
	m.CreatedOn = now
	m.ModifiedOn = now
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *DaoSubscribe) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndReplace(ctx, filter, &m)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *DaoSubscribe) Get(ctx context.Context, db *mongo.Database) (*DaoSubscribe, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var dao DaoSubscribe
	err := res.Decode(&dao)
	if err != nil {
		return nil, err
	}
	return &dao, nil
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

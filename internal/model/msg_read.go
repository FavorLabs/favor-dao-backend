package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MsgRead struct {
	ID        primitive.ObjectID `json:"id"   bson:"_id,omitempty"`
	From      primitive.ObjectID `json:"from" bson:"from"`
	To        primitive.ObjectID `json:"to" bson:"to"`
	ReadAt    int64              `json:"readAt" bson:"readAt"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

func (m *MsgRead) Table() string {
	return "msg_read"
}

func (m *MsgRead) Create(ctx context.Context, db *mongo.Database) (*MsgRead, error) {
	now := time.Now().Unix()
	m.CreatedAt = now

	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *MsgRead) Update(db *mongo.Database, conditions *ConditionsT) error {
	m.ReadAt = time.Now().Unix()
	var filter bson.M
	for k, v := range *conditions {
		if k != "ORDER" {
			if filter != nil {
				filter = findQuery1([]bson.M{filter, v})
			} else {
				filter = findQuery1([]bson.M{v})
			}
		}
	}
	update := bson.M{"$set": m}
	if _, err := db.Collection(m.Table()).UpdateMany(context.TODO(), filter, update); err != nil {
		return err
	}
	return nil
}

func (m *MsgRead) Get(db *mongo.Database, conditions *ConditionsT) (*MsgRead, error) {
	var query bson.M
	for k, v := range *conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery1([]bson.M{query, v})
			} else {
				query = findQuery1([]bson.M{v})
			}
		}
	}

	res := db.Collection(m.Table()).FindOne(context.TODO(), query)
	if res.Err() != nil {
		return nil, res.Err()
	}

	var msgRead MsgRead
	err := res.Decode(&msgRead)
	if err != nil {
		return nil, err
	}
	return &msgRead, nil
}

func (m *MsgRead) Delete(db *mongo.Database, conditions *ConditionsT) error {
	var filter bson.M

	for _, v := range *conditions {
		if filter != nil {
			filter = findQuery1([]bson.M{filter, v})
		} else {
			filter = findQuery1([]bson.M{v})
		}
	}

	_, err := db.Collection(m.Table()).DeleteMany(context.TODO(), filter)

	return err
}

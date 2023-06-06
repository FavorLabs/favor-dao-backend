package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MsgSys struct {
	ID        primitive.ObjectID `json:"id"   bson:"_id,omitempty"`
	From      primitive.ObjectID `json:"from" bson:"from"`
	Title     string             `json:"title"   bson:"title"`
	Content   string             `json:"content" bson:"content"`
	Links     string             `json:"links" bson:"links"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

func (m *MsgSys) Table() string {
	return "msg_sys"
}

func (m *MsgSys) Create(ctx context.Context, db *mongo.Database) (*MsgSys, error) {
	now := time.Now().Unix()
	m.CreatedAt = now
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *MsgSys) Get(db *mongo.Database, conditions *ConditionsT) (*MsgSys, error) {
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

	var msgSys MsgSys
	err := res.Decode(&msgSys)
	if err != nil {
		return nil, err
	}
	return &msgSys, nil
}

func (m *MsgSys) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) (*[]MsgSys, error) {
	var (
		msgs   []MsgSys
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)

	finds := make([]*options.FindOptions, 0, 3)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	for k, v := range *conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery1([]bson.M{query, v})
			} else {
				query = findQuery1([]bson.M{v})
			}
		} else {
			finds = append(finds, options.Find().SetSort(v))
		}
	}
	if cursor, err = db.Collection(m.Table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var msg MsgSys
		if cursor.Decode(&msg) != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return &msgs, nil
}

func (m *MsgSys) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
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
	count, err := db.Collection(m.Table()).CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

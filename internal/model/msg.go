package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Msg struct {
	ID        primitive.ObjectID `json:"id"   bson:"_id,omitempty"`
	Title     string             `json:"title"   bson:"title"`
	Content   string             `json:"content" bson:"content"`
	Links     string             `json:"links" bson:"links"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

func (m *Msg) Table() string {
	return "msg"
}

func (m *Msg) Get(ctx context.Context, db *mongo.Database) (*Msg, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}}

	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var msg Msg
	err := res.Decode(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *Msg) Create(ctx context.Context, db *mongo.Database) (*Msg, error) {
	now := time.Now().Unix()
	m.CreatedAt = now

	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *Msg) Delete(db *mongo.Database, conditions *ConditionsT) error {
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

func (m *Msg) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) (*[]Msg, error) {
	var (
		msgs   []Msg
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)

	for k, v := range *conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery1([]bson.M{query, v})
			} else {
				query = findQuery1([]bson.M{v})
			}
		}
	}

	sortStage := bson.D{{"$sort", bson.D{{"_id", -1}}}}
	limitStage := bson.D{{"$limit", limit}}
	skipStage := bson.D{{"$skip", offset}}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "msg_send",
			"localField":   "_id",
			"foreignField": "msg_id",
			"as":           "send",
		}}},
		{{"$match", query}},
		sortStage,
		skipStage,
		limitStage,
	}
	ctx := context.TODO()
	cursor, err = db.Collection(m.Table()).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var msg Msg
		if cursor.Decode(&msg) != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return &msgs, nil
}

func (m *Msg) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
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
	countStage := bson.D{{"$count", "countGroup"}}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "msg_send",
			"localField":   "_id",
			"foreignField": "msg_id",
			"as":           "send",
		}}},
		{{"$match", query}},
		countStage,
	}
	cursor, err := db.Collection(m.Table()).Aggregate(context.TODO(), pipeline)
	if err != nil {
		return 0, err
	}
	for cursor.Next(context.TODO()) {
		var msgSend map[string]int64
		if cursor.Decode(&msgSend) != nil {
			return 0, err
		}
		return msgSend["countGroup"], nil
	}
	return 0, nil
}

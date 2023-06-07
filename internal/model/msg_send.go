package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FromTypeEnum uint8

const (
	DAO_TYPE FromTypeEnum = iota
	ORANGE
	USER
)

type MsgSend struct {
	ID        primitive.ObjectID `json:"id"   bson:"_id,omitempty"`
	MsgID     primitive.ObjectID `json:"msgID" bson:"msg_id"`
	From      primitive.ObjectID `json:"from" bson:"from"`
	To        primitive.ObjectID `json:"to" bson:"to"`
	FromType  FromTypeEnum       `json:"fromType" bson:"fromType"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type MsgSendGroup struct {
	From     primitive.ObjectID `json:"from" bson:"from"`
	To       primitive.ObjectID `json:"to" bson:"to"`
	FromType FromTypeEnum       `json:"fromType" bson:"fromType"`
}

func (m *MsgSend) Table() string {
	return "msg_send"
}

func (m *MsgSend) Create(ctx context.Context, db *mongo.Database) (*MsgSend, error) {
	now := time.Now().Unix()
	m.CreatedAt = now
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *MsgSend) Get(db *mongo.Database, conditions *ConditionsT) (*MsgSend, error) {
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

	var msgSend MsgSend
	err := res.Decode(&msgSend)
	if err != nil {
		return nil, err
	}
	return &msgSend, nil
}

func (m *MsgSend) Delete(db *mongo.Database, conditions *ConditionsT) error {
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

func (m *MsgSend) List(db *mongo.Database, conditions *ConditionsT) (*[]MsgSend, error) {
	var (
		mss    []MsgSend
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)

	for _, v := range *conditions {
		if query != nil {
			query = findQuery1([]bson.M{query, v})
		} else {
			query = findQuery1([]bson.M{v})
		}
	}
	if cursor, err = db.Collection(m.Table()).Find(context.TODO(), query); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	mss = make([]MsgSend, 0)
	for cursor.Next(context.TODO()) {
		var ms MsgSend
		if cursor.Decode(&ms) != nil {
			return nil, err
		}
		mss = append(mss, ms)
	}
	return &mss, nil
}

func (m *MsgSend) ListGroup(db *mongo.Database, to primitive.ObjectID,
	froms *[]primitive.ObjectID, offset, limit int) (*[]MsgSendGroup, error) {
	var matchStage bson.D
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", bson.D{
				{"to", "$to"},
				{"from", "$from"},
				{"fromType", "$fromType"},
			}}}}}

	if froms != nil && len(*froms) > 0 {
		matchStage = bson.D{{"$match", bson.D{{"to", to},
			{"from", bson.M{"$nin": *froms}}}}}
	} else {
		matchStage = bson.D{{"$match", bson.D{{"to", to}}}}
	}
	sortStage := bson.D{{"$sort", bson.D{{"_id.from", -1}}}}
	limitStage := bson.D{{"$limit", limit}}
	skipStage := bson.D{{"$skip", offset}}

	cursor, err := db.Collection(m.Table()).Aggregate(context.TODO(),
		mongo.Pipeline{matchStage, groupStage, sortStage, skipStage, limitStage})
	if err != nil {
		return nil, err
	}
	msgSends := make([]MsgSendGroup, 0, limit)
	for cursor.Next(context.TODO()) {
		var ms map[string]MsgSendGroup
		if cursor.Decode(&ms) != nil {
			return nil, err
		}
		msgSends = append(msgSends, ms["_id"])
	}
	return &msgSends, nil
}

func (m *MsgSend) CountGroup(db *mongo.Database, to primitive.ObjectID, froms *[]primitive.ObjectID) (int64, error) {
	var matchStage bson.D
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", bson.D{
				{"to", "$to"},
				{"from", "$from"},
			}}}}}
	if froms != nil && len(*froms) > 0 {
		matchStage = bson.D{{"$match", bson.D{{"to", to},
			{"from", bson.M{"$nin": *froms}}}}}
	} else {
		matchStage = bson.D{{"$match", bson.D{{"to", to}}}}
	}
	countStage := bson.D{{"$count", "countGroup"}}

	cursor, err := db.Collection(m.Table()).Aggregate(context.TODO(),
		mongo.Pipeline{matchStage, groupStage, countStage})
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

func (m *MsgSend) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
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

func (m *MsgSend) GetLast(db *mongo.Database, conditions *ConditionsT) (*MsgSend, error) {
	var (
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)

	finds := make([]*options.FindOptions, 0, 2)
	finds = append(finds, options.Find().SetLimit(1))
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
		var msg MsgSend
		if cursor.Decode(&msg) != nil {
			return nil, err
		}
		return &msg, nil
	}
	return nil, err
}

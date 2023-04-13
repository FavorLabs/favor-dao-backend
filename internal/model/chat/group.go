package chat

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GroupID struct {
	DaoID   primitive.ObjectID `json:"dao_id"   bson:"dao_id"`
	GroupID string             `json:"group_id" bson:"group_id"`
	OwnerID string             `json:"owner_id" bson:"owner_id"`
}

type Group struct {
	ID GroupID `json:"id"       bson:"_id,omitempty"`
}

func (m *Group) Table() string {
	return "chat_group"
}

func (m *Group) Create(ctx context.Context, db *mongo.Database) (*Group, error) {
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	var g Group
	resMap := res.InsertedID.(primitive.D).Map()
	g.ID.DaoID = resMap["dao_id"].(primitive.ObjectID)
	g.ID.GroupID = resMap["group_id"].(string)
	g.ID.OwnerID = resMap["owner_id"].(string)
	return &g, nil
}

func (m *Group) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	_, err := db.Collection(m.Table()).DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

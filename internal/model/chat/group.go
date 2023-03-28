package chat

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Group struct {
	ID      primitive.ObjectID `json:"id"       bson:"_id,omitempty"`
	DaoID   primitive.ObjectID `json:"dao_id"   bson:"dao_id"`
	OwnerID string             `json:"owner_id" bson:"owner_id"`
	GroupID string
}

func (m *Group) Table() string {
	return "chat_group"
}

func (m *Group) Create(ctx context.Context, db *mongo.Database) (*Group, error) {
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *Group) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndDelete(ctx, filter)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Group) RelatedDao(ctx context.Context, db *mongo.Database) (*Group, error) {
	if m.DaoID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"dao_id", m.DaoID}}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var dao Group
	err := res.Decode(&dao)
	if err != nil {
		return nil, err
	}
	return &dao, nil
}

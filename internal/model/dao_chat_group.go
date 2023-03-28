package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DaoChatGroup struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	Address    string             `json:"address"          bson:"address"`
	DaoID      primitive.ObjectID `json:"dao_id"           bson:"dao_id"`
	Guid       string             `json:"guid"             bson:"guid"`
	Name       string             `json:"name"             bson:"name"`
	Type       string             `json:"type"             bson:"type"` // public private password
	Icon       string             `json:"icon"             bson:"icon"`
}

func (m *DaoChatGroup) Table() string {
	return "dao_chat_group"
}

func (m *DaoChatGroup) Create(ctx context.Context, db *mongo.Database) (*DaoChatGroup, error) {
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

func (m *DaoChatGroup) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndDelete(ctx, filter)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *DaoChatGroup) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndReplace(ctx, filter, &m)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *DaoChatGroup) Get(ctx context.Context, db *mongo.Database) (*DaoChatGroup, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var dao DaoChatGroup
	err := res.Decode(&dao)
	if err != nil {
		return nil, err
	}
	return &dao, nil
}

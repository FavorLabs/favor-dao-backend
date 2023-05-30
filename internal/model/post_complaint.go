package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostComplaint struct {
	DefaultModel `bson:",inline"`
	Address      string             `json:"address"          bson:"address"`
	PostID       primitive.ObjectID `json:"post_id"          bson:"post_id"`
	Reason       string             `json:"reason"           bson:"reason"`
}

func (m *PostComplaint) Table() string {
	return "post_complaint"
}

func (m *PostComplaint) Create(ctx context.Context, db *mongo.Database) error {
	return create(ctx, db, m)
}

func (m *PostComplaint) Update(ctx context.Context, db *mongo.Database) error {
	return update(ctx, db, m)
}

func (m *PostComplaint) Get(ctx context.Context, db *mongo.Database) error {
	filter := bson.M{ID: m.GetID()}

	return findOne(ctx, db, m, filter)
}

func (m *PostComplaint) FindOne(ctx context.Context, db *mongo.Database, filter interface{}) error {
	return findOne(ctx, db, m, filter)
}

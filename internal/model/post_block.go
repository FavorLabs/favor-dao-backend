package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BlockModel int

const (
	BlockModelPost BlockModel = iota
	BlockModelDAO
)

type PostBlock struct {
	DefaultModel `bson:",inline"`
	Address      string             `json:"address"          bson:"address"`
	BlockId      primitive.ObjectID `json:"block_id"         bson:"block_id"`
	Model        BlockModel         `json:"model"            bson:"model"`
}

func (m *PostBlock) Table() string {
	return "post_block"
}

func (m *PostBlock) Create(ctx context.Context, db *mongo.Database) error {
	return create(ctx, db, m)
}

func (m *PostBlock) Update(ctx context.Context, db *mongo.Database) error {
	return update(ctx, db, m)
}

func (m *PostBlock) Get(ctx context.Context, db *mongo.Database) error {
	filter := bson.M{ID: m.GetID()}

	return findOne(ctx, db, m, filter)
}

func (m *PostBlock) FindOne(ctx context.Context, db *mongo.Database, filter interface{}) error {
	return findOne(ctx, db, m, filter)
}

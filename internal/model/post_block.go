package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (m *PostBlock) FindIDs(ctx context.Context, db *mongo.Database, filter interface{}, options ...*options.FindOptions) (list []string) {
	cursor, err := db.Collection(m.Table()).Find(ctx, filter, options...)
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(context.TODO()) {
		var tmp PostBlock
		if cursor.Decode(&tmp) != nil {
			return
		}
		list = append(list, tmp.BlockId.Hex())
	}
	return
}

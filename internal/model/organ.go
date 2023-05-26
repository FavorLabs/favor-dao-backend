package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Organ struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name   string             `json:"name" bson:"name"`
	Avatar string             `json:"avatar" bson:"avatar"`
}

func (o *Organ) Table() string {
	return "organ"
}

func (o *Organ) Get(ctx context.Context, db *mongo.Database) (*Organ, error) {
	var (
		organ Organ
		res   *mongo.SingleResult
	)
	filter := bson.D{{"_id", o.ID}}
	res = db.Collection(o.Table()).FindOne(ctx, filter)
	err := res.Err()
	if err != nil {
		return &organ, err
	}
	err = res.Decode(&organ)
	if err != nil {
		return &organ, err
	}
	return &organ, nil
}

package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Organ struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Key    string             `json:"key" bson:"key"`
	Name   string             `json:"name" bson:"name"`
	Avatar string             `json:"avatar" bson:"avatar"`
	IsShow bool               `json:"isShow" bson:"isShow"`
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

func (o *Organ) GetByKey(ctx context.Context, db *mongo.Database) (*Organ, error) {
	var (
		organ Organ
		res   *mongo.SingleResult
	)
	filter := bson.D{{"key", o.Key}}
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

func (o *Organ) List(db *mongo.Database, conditions *ConditionsT) (*[]Organ, error) {
	var (
		organs []Organ
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)

	finds := make([]*options.FindOptions, 0, 1)
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
	if cursor, err = db.Collection(o.Table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var organ Organ
		if cursor.Decode(&organ) != nil {
			return nil, err
		}
		organs = append(organs, organ)

	}
	return &organs, nil
}

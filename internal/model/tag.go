package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Tag struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn  int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel      int                `json:"is_del"           bson:"is_del"`
	Address    string             `json:"address"          bson:"address"`
	Tag        string             `json:"tag"              bson:"tag"`
	QuoteNum   int64              `json:"quote_num"        bson:"quote_num"`
}

type TagFormatted struct {
	ID       string         `json:"id"`
	Address  string         `json:"address"`
	User     *UserFormatted `json:"user"`
	Tag      string         `json:"tag"`
	QuoteNum int64          `json:"quote_num"`
}

func (m *Tag) Format() *TagFormatted {
	return &TagFormatted{
		ID:       m.ID.Hex(),
		Address:  m.Address,
		User:     &UserFormatted{},
		Tag:      m.Tag,
		QuoteNum: m.QuoteNum,
	}
}

func (m *Tag) table() string {
	return "tag"
}

func (m *Tag) Get(ctx context.Context, db *mongo.Database) (*Tag, error) {
	var (
		tag Tag
		res *mongo.SingleResult
	)
	if !m.ID.IsZero() {
		filter := bson.D{{"_id", m.ID}, {"is_del", 0}}
		res = db.Collection(m.table()).FindOne(ctx, filter)
	} else {
		filter := bson.D{{"tag", m.Tag}, {"is_del", 0}}
		res = db.Collection(m.table()).FindOne(ctx, filter)
	}
	err := res.Err()
	if err != nil {
		return &tag, err
	}
	err = res.Decode(&tag)
	if err != nil {
		return &tag, err
	}

	return &tag, nil
}

func (m *Tag) Create(ctx context.Context, db *mongo.Database) (*Tag, error) {
	res, err := db.Collection(m.table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *Tag) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	res := db.Collection(m.table()).FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Tag) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.table()).FindOneAndReplace(ctx, filter, &m)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Tag) Aggregate(ctx context.Context, db *mongo.Database, pipeline bson.A) (res []*Tag, err error) {
	cur, err := db.Collection(m.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	err = cur.All(ctx, &res)
	return
}

func (m *Tag) TagsFrom(ctx context.Context, db *mongo.Database, tags []string) (res []*Tag, err error) {
	filter := bson.D{{"tag", bson.M{"$in": tags}}}
	cur, err := db.Collection(m.table()).Find(ctx, filter)
	if err != nil {
		return
	}
	err = cur.All(ctx, &res)
	return
}

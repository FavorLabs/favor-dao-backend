package model

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (m *Tag) Table() string {
	return "tag"
}

func (m *Tag) Get(ctx context.Context, db *mongo.Database) (*Tag, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}, {"is_del", 0}}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var tmp Tag
	err := res.Decode(&tmp)
	if err != nil {
		return nil, err
	}
	return &tmp, nil
}

func (m *Tag) Create(ctx context.Context, db *mongo.Database) (*Tag, error) {
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

func (m *Tag) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndReplace(ctx, filter, &m)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Tag) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	res := db.Collection(m.Table()).FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Tag) FindListByKeyword(ctx context.Context, db *mongo.Database, keyword string, offset, limit int) (list []*Tag, err error) {
	var filter bson.M
	if keyword != "" {
		filter = bson.M{
			"tags": fmt.Sprintf("/%s/", keyword),
		}
	}
	finds := make([]*options.FindOptions, 0, 3)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	finds = append(finds, options.Find().SetSort(bson.M{"quote_num": 1}))
	cur, err := db.Collection(m.Table()).Find(ctx, filter, finds...)
	if err != nil {
		return
	}
	var res []*Tag
	err = cur.All(ctx, &res)
	if err != nil {
		return
	}
	return res, nil
}

func (m *Tag) TagsFrom(ctx context.Context, db *mongo.Database, tags []string) (res []*Tag, err error) {
	filter := bson.D{{"tag", bson.M{"$in": tags}}}
	cur, err := db.Collection(m.Table()).Find(ctx, filter)
	if err != nil {
		return
	}
	err = cur.All(ctx, &res)
	return
}

func (t *Tag) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*Tag, error) {

	var (
		tags   []*Tag
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)
	if t.Address != "" {
		query = bson.M{"address": t.Address}
	}
	finds := make([]*options.FindOptions, 0, 3)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	for k, v := range *conditions {
		if k != "ORDER" {
			query = findQuery([]bson.M{query, v})
		} else {
			finds = append(finds, options.Find().SetSort(v))
		}
	}
	if cursor, err = db.Collection(t.Table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var tag Tag
		if cursor.Decode(&tag) != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

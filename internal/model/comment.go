package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Comment struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn  int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel      int                `json:"is_del"           bson:"is_del"`
	PostID     primitive.ObjectID `json:"post_id"          bson:"post_id"`
	Address    string             `json:"address"          bson:"address"`
}

type CommentFormatted struct {
	ID         primitive.ObjectID       `json:"id"`
	PostID     primitive.ObjectID       `json:"post_id"`
	Address    string                   `json:"address"`
	User       *UserFormatted           `json:"user"`
	Contents   []*CommentContent        `json:"contents"`
	Replies    []*CommentReplyFormatted `json:"replies"`
	CreatedOn  int64                    `json:"created_on"`
	ModifiedOn int64                    `json:"modified_on"`
}

func (c *Comment) Format() *CommentFormatted {
	return &CommentFormatted{
		ID:         c.ID,
		PostID:     c.PostID,
		Address:    c.Address,
		User:       &UserFormatted{},
		Contents:   []*CommentContent{},
		Replies:    []*CommentReplyFormatted{},
		CreatedOn:  c.CreatedOn,
		ModifiedOn: c.ModifiedOn,
	}
}

func (c *Comment) Table() string {
	return "comment"
}

func (c *Comment) Get(ctx context.Context, db *mongo.Database) (*Comment, error) {
	if c.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", c.ID}, {"is_del", 0}}
	res := db.Collection(c.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var out Comment
	err := res.Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Comment) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*Comment, error) {
	var comments []*Comment
	var err error
	var query bson.M
	var cursor *mongo.Cursor

	finds := make([]*options.FindOptions, 0)
	if offset >= 0 && limit > 0 {
		finds = append(finds, options.Find().SetSkip(int64(offset)))
		finds = append(finds, options.Find().SetLimit(int64(limit)))
	}
	if !c.PostID.IsZero() {
		query = bson.M{"post_id": c.PostID}
	}

	for k, v := range *conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery([]bson.M{query, v})
			} else {
				query = findQuery([]bson.M{v})
			}
		} else {
			finds = append(finds, options.Find().SetSort(v))
		}
	}

	if cursor, err = db.Collection(c.Table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var tmp Comment
		if cursor.Decode(&tmp) != nil {
			return nil, err
		}
		comments = append(comments, &tmp)
	}
	return comments, nil
}

func (c *Comment) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
	var count int64
	var query bson.M
	if !c.PostID.IsZero() {
		query = bson.M{"post_id": c.PostID}
	}
	for k, v := range *conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery([]bson.M{query, v})
			} else {
				query = findQuery([]bson.M{v})
			}
		}
	}
	count, err := db.Collection(c.Table()).CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Comment) Create(db *mongo.Database) (*Comment, error) {
	now := time.Now().Unix()
	c.CreatedOn = now
	c.ModifiedOn = now
	res, err := db.Collection(c.Table()).InsertOne(context.TODO(), &c)
	if err != nil {
		return nil, err
	}
	c.ID = res.InsertedID.(primitive.ObjectID)
	return c, err
}

func (c *Comment) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", c.ID}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	res := db.Collection(c.Table()).FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (c *Comment) CommentIdsByPostId(db *mongo.Database, postId string) (ids []primitive.ObjectID, err error) {
	hex, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		return nil, err
	}
	filter := bson.D{{"post_id", hex}, {"_id", 1}}
	cursor, err := db.Collection(c.Table()).Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	type out struct {
		Id primitive.ObjectID `bson:"_id"`
	}
	for cursor.Next(context.TODO()) {
		var tmp out
		if cursor.Decode(&tmp) != nil {
			return nil, err
		}
		ids = append(ids, tmp.Id)
	}
	return
}

func (c *Comment) DeleteByPostId(db *mongo.Database, postId string) error {
	hex, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		return err
	}
	filter := bson.D{{"post_id", hex}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	_, err = db.Collection(c.Table()).UpdateMany(context.TODO(), filter, update)
	return err
}

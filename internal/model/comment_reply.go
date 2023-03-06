package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommentReply struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn  int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel      int                `json:"is_del"           bson:"is_del"`
	CommentID  primitive.ObjectID `json:"comment_id"       bson:"comment_id"`
	Address    string             `json:"address"          bson:"address"`
	Content    string             `json:"content"          bson:"content"`
}

type CommentReplyFormatted struct {
	ID         primitive.ObjectID `json:"id"`
	CommentID  primitive.ObjectID `json:"comment_id"`
	Address    string             `json:"address"`
	User       *UserFormatted     `json:"user"`
	Content    string             `json:"content"`
	CreatedOn  int64              `json:"created_on"`
	ModifiedOn int64              `json:"modified_on"`
}

func (c *CommentReply) Format() *CommentReplyFormatted {
	return &CommentReplyFormatted{
		ID:         c.ID,
		CommentID:  c.CommentID,
		Address:    c.Address,
		User:       &UserFormatted{},
		Content:    c.Content,
		CreatedOn:  c.CreatedOn,
		ModifiedOn: c.ModifiedOn,
	}
}

func (c *CommentReply) Table() string {
	return "comment_reply"
}

func (c *CommentReply) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*CommentReply, error) {
	var comments []*CommentReply
	var err error
	var query bson.M
	var cursor *mongo.Cursor

	finds := make([]*options.FindOptions, 0)
	if offset >= 0 && limit > 0 {
		finds = append(finds, options.Find().SetSkip(int64(offset)))
		finds = append(finds, options.Find().SetLimit(int64(limit)))
	}
	if !c.CommentID.IsZero() {
		query = bson.M{"comment_id": c.CommentID}
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
		var tmp CommentReply
		if cursor.Decode(&tmp) != nil {
			return nil, err
		}
		comments = append(comments, &tmp)
	}
	return comments, nil
}

func (c *CommentReply) Create(db *mongo.Database) (*CommentReply, error) {
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

func (c *CommentReply) Get(db *mongo.Database) (*CommentReply, error) {
	if c.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", c.ID}, {"is_del", 0}}
	res := db.Collection(c.Table()).FindOne(context.TODO(), filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var out CommentReply
	err := res.Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CommentReply) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", c.ID}, {"is_del", 0}}
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

func (c *CommentReply) DeleteByCommentIds(db *mongo.Database, commentIds []primitive.ObjectID) error {
	filter := bson.D{
		{"comment_id", bson.D{{"$in", commentIds}}},
		{"is_del", 0},
	}
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

package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommentContent struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn  int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel      int                `json:"is_del"           bson:"is_del"`
	CommentID  primitive.ObjectID `json:"comment_id"       bson:"comment_id"`
	Address    string             `json:"address"          bson:"address"`
	Content    string             `json:"content"          bson:"content"`
	Type       PostContentT       `json:"type"             bson:"type"`
	Sort       int64              `json:"sort"             bson:"sort"`
}

func (c *CommentContent) PostFormat() *PostContentFormatted {
	return &PostContentFormatted{
		ID:      c.ID,
		Address: c.Address,
		Content: c.Content,
		Type:    c.Type,
		Sort:    c.Sort,
	}
}

func (c *CommentContent) Table() string {
	return "comment_content"
}

func (c *CommentContent) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*CommentContent, error) {
	var comments []*CommentContent
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
	if len(*conditions) == 0 {
		query = findQuery([]bson.M{query})
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
		var tmp CommentContent
		if cursor.Decode(&tmp) != nil {
			return nil, err
		}
		comments = append(comments, &tmp)
	}
	return comments, nil
}

func (c *CommentContent) Create(db *mongo.Database) (*CommentContent, error) {
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

func (c *CommentContent) MediaContentsByCommentId(db *mongo.Database, commentIds []primitive.ObjectID) (contents []string, err error) {
	filter := bson.D{
		{"comment_id", bson.D{{"$in", commentIds}}},
		{"type", CONTENT_TYPE_IMAGE},
	}
	projection := bson.D{
		{"content", 1},
	}
	cursor, err := db.Collection(c.Table()).Find(context.TODO(), filter, options.Find().SetProjection(projection))

	for cursor.Next(context.TODO()) {
		var tmp CommentContent
		if cursor.Decode(&tmp) != nil {
			return nil, err
		}
		contents = append(contents, tmp.Content)
	}
	return contents, nil
}

func (c *CommentContent) DeleteByCommentIds(db *mongo.Database, commentIds []primitive.ObjectID) error {
	filter := bson.D{
		{"comment_id", bson.D{{"$in", commentIds}}},
	}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	_, err := db.Collection(c.Table()).UpdateMany(context.TODO(), filter, update)
	return err
}

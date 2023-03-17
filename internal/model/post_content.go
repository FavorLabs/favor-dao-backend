package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Type, 1 title, 2 text paragraph, 3 picture address, 4 video address, 5 voice address, 6 link address, 7 attachment resource

type PostContentT int

const (
	CONTENT_TYPE_TITLE PostContentT = iota + 1
	CONTENT_TYPE_TEXT
	CONTENT_TYPE_IMAGE
	CONTENT_TYPE_VIDEO
	CONTENT_TYPE_FAVOR
	CONTENT_TYPE_AUDIO
	CONTENT_TYPE_LINK
)

var (
	mediaContentType = []PostContentT{
		CONTENT_TYPE_IMAGE,
		CONTENT_TYPE_VIDEO,
		CONTENT_TYPE_AUDIO,
	}
)

type PostType int

const (
	DAO PostType = iota - 1
	SMS
	VIDEO
	Retweet
	RetweetComment
)

type PostRefType int

const (
	RefPost PostRefType = iota
	RefComment
	RefCommentReply
)

type PostContent struct {
	ID      primitive.ObjectID `json:"id"      bson:"_id,omitempty"`
	PostID  primitive.ObjectID `json:"post_id" bson:"post_id"`
	Address string             `json:"address" bson:"address"`
	Content string             `json:"content" bson:"content"`
	Type    PostContentT       `json:"type"    bson:"type"`
	Sort    int64              `json:"sort"    bson:"sort"`
	IsDel   int                `json:"is_del"  bson:"is_del"`
}

type PostContentFormatted struct {
	ID      primitive.ObjectID `json:"id"`
	PostID  primitive.ObjectID `json:"post_id"`
	Address string             `json:"address"`
	Content string             `json:"content"`
	Type    PostContentT       `json:"type"`
	Sort    int64              `json:"sort"`
}

func (p *PostContent) table() string {
	return "post_content"
}

func (p *PostContent) DeleteByPostId(db *mongo.Database, postId primitive.ObjectID) error {
	filter := bson.D{{"post_id", postId}}
	update := bson.D{{"$set", bson.D{{"is_del", 1}}}}
	res := db.Collection(p.table()).FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (p *PostContent) MediaContentsByPostId(db *mongo.Database, postId primitive.ObjectID) (contents []string, err error) {
	filter := bson.D{
		{"is_del", 0},
		{"post_id", postId},
		{"type", bson.D{{"$in", bson.A{CONTENT_TYPE_IMAGE, CONTENT_TYPE_VIDEO, CONTENT_TYPE_AUDIO, CONTENT_TYPE_FAVOR}}}}}
	cursor, err := db.Collection(p.table()).Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var post PostContent
		if cursor.Decode(&post) != nil {
			return nil, err
		}
		contents = append(contents, post.Content)
	}
	return contents, nil
}

func (p *PostContent) Create(db *mongo.Database) (*PostContent, error) {
	res, err := db.Collection(p.table()).InsertOne(context.TODO(), &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

func (p *PostContent) Format() *PostContentFormatted {
	return &PostContentFormatted{
		ID:      p.ID,
		PostID:  p.PostID,
		Address: p.Address,
		Content: p.Content,
		Type:    p.Type,
		Sort:    p.Sort,
	}
}

func (p *PostContent) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*PostContent, error) {
	var (
		contents []*PostContent
		err      error
		cursor   *mongo.Cursor
		query    bson.M
	)
	if !p.PostID.IsZero() {
		query = bson.M{"post_id": p.PostID}
	}
	finds := make([]*options.FindOptions, 0, 3)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	if len(*conditions) == 0 {
		if query != nil {
			query = findQuery([]bson.M{query})
		} else {
			query = bson.M{"is_del": 0}
		}
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
	if cursor, err = db.Collection(p.table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var content PostContent
		if cursor.Decode(&content) != nil {
			return nil, err
		}
		contents = append(contents, &content)
	}
	return contents, nil
}

func (p *PostContent) Get(db *mongo.Database) (*PostContent, error) {
	var content PostContent
	if p.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", p.ID}, {"is_del", 0}}
	res := db.Collection(p.table()).FindOne(context.TODO(), filter)
	if res.Err() != nil {
		return nil, res.Err()
	}

	err := res.Decode(&content)
	if err != nil {
		return nil, err
	}

	return &content, nil
}

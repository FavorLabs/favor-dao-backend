package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
)

// PostVisibleT Accessible type, 0 public, 1 private, 2 friends
type PostVisibleT uint8

const (
	PostVisitDraft PostVisibleT = iota
	PostVisitPublic
	PostVisitPrivate
	//PostVisitSecret
	//PostVisitFriend
	//PostVisitInvalid
)

type Post struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Address         string             `json:"address" bson:"address"`
	DaoId           primitive.ObjectID `json:"dao_id" bson:"dao_id"`
	ViewCount       int64              `json:"view_count" bson:"view_count"`
	CollectionCount int64              `json:"collection_count" bson:"collection_count"`
	UpvoteCount     int64              `json:"upvote_count" bson:"upvote_count"`
	Member          int                `json:"member" bson:"member"`
	Visibility      PostVisibleT       `json:"visibility" bson:"visibility"`
	IsTop           int                `json:"is_top" bson:"is_top"`
	IsEssence       int                `json:"is_essence" bson:"is_essence"`
	Tags            string             `json:"tags" bson:"tags"`
	Type            PostType           `json:"type" bson:"type"`
}

type PostFormated struct {
	ID              primitive.ObjectID     `json:"id"`
	DaoId           primitive.ObjectID     `json:"daoId"`
	Address         string                 `json:"address"`
	User            *UserFormated          `json:"user"`
	Contents        []*PostContentFormated `json:"contents"`
	Member          int                    `json:"member"`
	ViewCount       int64                  `json:"view_count"`
	CollectionCount int64                  `json:"collection_count"`
	UpvoteCount     int64                  `json:"upvote_count"`
	Visibility      PostVisibleT           `json:"visibility"`
	IsTop           int                    `json:"is_top"`
	IsEssence       int                    `json:"is_essence"`
	Tags            map[string]int8        `json:"tags"`
	Type            PostType               `json:"type"`
}

func (p *Post) table() string {
	return "post"
}

func (p *Post) Format() *PostFormated {
	tagsMap := map[string]int8{}
	for _, tag := range strings.Split(p.Tags, ",") {
		tagsMap[tag] = 1
	}
	return &PostFormated{
		ID:              p.ID,
		DaoId:           p.DaoId,
		Address:         p.Address,
		User:            &UserFormated{},
		Contents:        []*PostContentFormated{},
		Member:          p.Member,
		ViewCount:       p.ViewCount,
		CollectionCount: p.CollectionCount,
		UpvoteCount:     p.UpvoteCount,
		Visibility:      p.Visibility,
		IsTop:           p.IsTop,
		IsEssence:       p.IsEssence,
		Tags:            tagsMap,
		Type:            p.Type,
	}
}

func (p *Post) Create(db *mongo.Database) (*Post, error) {
	res, err := db.Collection(p.table()).InsertOne(context.TODO(), &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

func (p *Post) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", p.ID}}
	update := bson.D{{"$set", bson.D{{"is_del", 1}}}}
	res := db.Collection(p.table()).FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (p *Post) Get(db *mongo.Database) (*Post, error) {
	if p.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", p.ID}, {"is_del", 0}}
	res := db.Collection(p.table()).FindOne(context.TODO(), filter)
	if res.Err() != nil {
		return nil, res.Err()
	}

	var post Post
	err := res.Decode(&post)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (p *Post) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*Post, error) {

	var (
		posts  []*Post
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)
	if p.Address != "" {
		query = bson.M{"address": p.Address}
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
	if cursor, err = db.Collection(p.table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var post Post
		if cursor.Decode(&post) != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	return posts, nil
}

func (p *Post) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {

	var query bson.M
	if p.Address != "" {
		query = bson.M{"address": p.Address}
	}
	for k, v := range *conditions {
		if k != "ORDER" {
			query = findQuery([]bson.M{query, v})
		}
	}
	count, err := db.Collection(p.table()).CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *Post) Update(db *mongo.Database) error {
	filter := bson.D{{"_id", p.ID}, {"is_del", 0}}
	update := bson.M{"$set": p}
	if _, err := db.Collection(p.table()).UpdateMany(context.TODO(), filter, update); err != nil {
		return err
	}
	return nil
}

func (p PostVisibleT) String() string {
	switch p {
	case PostVisitPublic:
		return "public"
	case PostVisitPrivate:
		return "private"
	case PostVisitDraft:
		return "draft"
	default:
		return "unknow"
	}
}

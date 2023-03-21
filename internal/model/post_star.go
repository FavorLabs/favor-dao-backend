package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostStar struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Post    *Post              `json:"-" bson:"posts"`
	PostID  primitive.ObjectID `json:"post_id" bson:"post_id"`
	Address string             `json:"address" bson:"address"`
	IsDel   int                `json:"is_del" bson:"is_del"`
}

type PostStarFormatted struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Post    *Post              `json:"post" bson:"post"`
	PostID  primitive.ObjectID `json:"post_id" bson:"post_id"`
	Address string             `json:"address" bson:"address"`
}

func (p *PostStar) table() string {
	return "post_star"
}
func (p *PostStar) Get(db *mongo.Database) (*PostStar, error) {
	var star PostStar
	var queries []bson.M

	if !p.ID.IsZero() {
		queries = append(queries, bson.M{"_id": p.ID})
	}
	if !p.PostID.IsZero() {
		queries = append(queries, bson.M{"post_id": p.PostID})
	}
	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}

	queries = append(queries, bson.M{"post.visibility": bson.M{"$ne": PostVisitPrivate}})

	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$match", findQuery(queries)}},
		{{"$unwind", "$post"}},
	}

	ctx := context.TODO()
	cursor, err := db.Collection(p.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	cursor.Next(ctx)
	err = cursor.Decode(&star)
	if err != nil {
		return nil, err
	}

	return &star, nil
}

func (p *PostStar) Create(db *mongo.Database) (*PostStar, error) {
	res, err := db.Collection(p.table()).InsertOne(context.TODO(), &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

func (p *PostStar) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", p.ID}}
	_, err := db.Collection(p.table()).DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}
	return nil
}

func (p *PostStar) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*PostStarFormatted, error) {
	var (
		stars   []*PostStarFormatted
		queries []bson.M
		query   bson.M
		sort    bson.M
		cursor  *mongo.Cursor
		err     error
	)

	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}
	queries = append(queries, bson.M{"post.visibility": bson.M{"$ne": PostVisitPrivate}})
	if len(*conditions) == 0 {
		if query != nil {
			query = findQuery([]bson.M{query})
		} else {
			query = bson.M{"is_del": 0}
		}
	}
	for k, v := range *conditions {
		if k != "ORDER" {
			queries = append(queries, v)
		} else {
			sort = v
		}
	}
	query = findQuery(queries)
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$match", query}},
		{{"$sort", sort}},
		{{"$skip", offset}},
		{{"$limit", limit}},
		{{"$unwind", "$post"}},
	}

	ctx := context.TODO()
	cursor, err = db.Collection(p.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(context.TODO()) {
		var star PostStarFormatted
		if cursor.Decode(&star) != nil {
			return nil, err
		}
		stars = append(stars, &star)
	}

	return stars, nil
}

func (p *PostStar) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
	var (
		queries []bson.M
		query   bson.M
		cursor  *mongo.Cursor
		err     error
	)

	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}
	queries = append(queries, bson.M{"post.visibility": bson.M{"$ne": PostVisitPrivate}})

	for k, v := range *conditions {
		if k != "ORDER" {
			queries = append(queries, v)
		}
	}
	query = findQuery(queries)
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$match", query}},
		{{"$count", "counted_documents"}},
	}

	ctx := context.TODO()
	cursor, err = db.Collection(p.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var count struct {
		Count int64 `bson:"counted_documents"`
	}
	cursor.Next(ctx)
	if err = cursor.Decode(&count); err != nil {
		panic(err)
	}
	return count.Count, err
}

func (p *PostStar) CountByAddress(db *mongo.Database, address string) (int64, error) {
	var count int64
	count, err := db.Collection(p.table()).CountDocuments(context.TODO(), bson.M{"address": address})
	if err != nil {
		return 0, err
	}

	return count, nil
}

package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostCollection struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Post    *Post              `json:"-" bson:"-"`
	PostID  primitive.ObjectID `json:"post_id" bson:"post_id"`
	Address string             `json:"address" bson:"address"`
}

func (p *PostCollection) table() string {
	return "post_collection"
}

func (p *PostCollection) Get(db *mongo.Database) (*PostCollection, error) {
	var star PostCollection
	var queries []bson.M

	if !p.ID.IsZero() {
		queries = append(queries, bson.M{"_id": p.ID, "is_del": 0})
	}
	if !p.PostID.IsZero() {
		queries = append(queries, bson.M{"post_id": p.PostID})
	}
	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}
	queries = append(queries, bson.M{"post.visibility": PostVisitPrivate})

	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$unwind", "$post"}},
		{{"$match", findQuery(queries)}}}

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

func (p *PostCollection) Create(db *mongo.Database) (*PostCollection, error) {
	res, err := db.Collection(p.table()).InsertOne(context.TODO(), &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

func (p *PostCollection) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", p.ID}}
	update := bson.D{{"$set", bson.D{{"is_del", 1}}}}
	res := db.Collection(p.table()).FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (p *PostCollection) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*PostCollection, error) {
	var (
		collections []*PostCollection
		queries     []bson.M
		query       bson.M
		sort        bson.M
		cursor      *mongo.Cursor
		err         error
	)

	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}
	queries = append(queries, bson.M{"post.visibility": PostVisitPrivate})

	for k, v := range *conditions {
		if k != "ORDER" {
			queries = append(queries, v)
			query = findQuery(queries)
		} else {
			sort = v
		}
	}

	pipeline := mongo.Pipeline{
		{{"$match", query}},
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$unwind", "$post"}},
		{{"$limit", limit}},
		{{"$skip", offset}},
		{{"$sort", sort}},
	}

	ctx := context.TODO()
	cursor, err = db.Collection(p.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(context.TODO()) {
		var collection PostCollection
		if cursor.Decode(&collection) != nil {
			return nil, err
		}
		collections = append(collections, &collection)
	}

	return collections, nil
}

func (p *PostCollection) Count(db *mongo.Database, conditions *ConditionsT) (int64, error) {
	var (
		queries []bson.M
		query   bson.M
		cursor  *mongo.Cursor
		err     error
	)

	if p.Address != "" {
		queries = append(queries, bson.M{"address": p.Address})
	}
	queries = append(queries, bson.M{"post.visibility": PostVisitPrivate})

	for k, v := range *conditions {
		if k != "ORDER" {
			queries = append(queries, v)
			query = findQuery(queries)
		}
	}

	pipeline := mongo.Pipeline{
		{{"$match", query}},
		{{"$lookup", bson.M{
			"from":         "post",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "post",
		}}},
		{{"$unwind", "$post"}},
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

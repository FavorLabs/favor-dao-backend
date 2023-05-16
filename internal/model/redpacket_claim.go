package model

import (
	"context"
	"math/big"

	"favor-dao-backend/pkg/convert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RedpacketClaim struct {
	DefaultModel `bson:",inline"`
	RedpacketId  primitive.ObjectID `json:"redpacket_id" bson:"redpacket_id"`
	Address      string             `json:"address" bson:"address"`
	Amount       string             `json:"amount" bson:"amount"`
	TxID         string             `json:"tx_id"    bson:"tx_id"`
	PayStatus    PayStatus          `json:"pay_status" bson:"pay_status"`
}

type RedpacketClaimFormatted struct {
	RedpacketClaim `bson:",inline"`
	Title          string `json:"title" bson:"title"`
	UserAvatar     string `json:"user_avatar" bson:"user_avatar"`
}

func (a *RedpacketClaim) Table() string {
	return "redpacket_claim"
}

func (a *RedpacketClaim) Create(ctx context.Context, db *mongo.Database) error {
	return create(ctx, db, a)
}

func (a *RedpacketClaim) Update(ctx context.Context, db *mongo.Database) error {
	return update(ctx, db, a)
}

func (a *RedpacketClaim) First(ctx context.Context, db *mongo.Database) error {
	filter := bson.M{ID: a.GetID()}

	return findOne(ctx, db, a, filter)
}

func (a *RedpacketClaim) FindOne(ctx context.Context, db *mongo.Database, filter interface{}) error {
	return findOne(ctx, db, a, filter)
}

func (a *RedpacketClaim) Find(ctx context.Context, db *mongo.Database, filter interface{}, opts ...*options.FindOptions) ([]RedpacketClaim, error) {
	cursor, err := find(ctx, db, a, filter, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]RedpacketClaim, 0, 1)
	for cursor.Next(ctx) {
		var t RedpacketClaim
		if err = cursor.Decode(&t); err != nil {
			return nil, err
		}
		list = append(list, t)
	}

	return list, nil
}

func (a *RedpacketClaim) FindList(ctx context.Context, db *mongo.Database, match interface{}, limit, offset int) (list []*RedpacketClaimFormatted) {
	red := Redpacket{}
	user := User{}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from": red.Table(),
			"let":  bson.M{"rpd": "$redpacket_id", "user": "$address"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$rpd"}}}},
				bson.M{"$lookup": bson.M{
					"from": user.Table(),
					"pipeline": bson.A{
						bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$address", "$$user"}}}},
						bson.M{"$project": bson.M{"_id": 0, "avatar": 1}},
					},
					"as": "user",
				}},
				bson.M{"$unwind": "$user"},
				bson.M{"$project": bson.M{"_id": 0, "title": 1, "user": 1}},
			},
			"as": "ext",
		}}},
		{{"$match", match}},
		{{"$sort", bson.M{"created_on": -1}}},
		{{"$skip", offset}},
		{{"$limit", limit}},
		{{"$unwind", "$ext"}},
	}
	list = []*RedpacketClaimFormatted{}
	cursor, err := db.Collection(a.Table()).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	type tmp struct {
		RedpacketClaim `bson:",inline"`
		Ext            struct {
			Title string `bson:"title"`
			User  struct {
				Avatar string `bson:"avatar"`
			} `bson:"user"`
		} `bson:"ext"`
	}
	for cursor.Next(context.TODO()) {
		var t tmp
		if cursor.Decode(&t) != nil {
			return
		}
		list = append(list, &RedpacketClaimFormatted{
			RedpacketClaim: t.RedpacketClaim,
			Title:          t.Ext.Title,
			UserAvatar:     t.Ext.User.Avatar,
		})
	}
	return
}

func (a *RedpacketClaim) Count(ctx context.Context, db *mongo.Database, match interface{}) int64 {
	documents, err := db.Collection(a.Table()).CountDocuments(ctx, match)
	if err != nil {
		return 0
	}
	return documents
}

func (a *RedpacketClaim) CountAmount(ctx context.Context, db *mongo.Database, match interface{}) (total string) {
	cur, err := db.Collection(a.Table()).Find(ctx, match)
	if err != nil {
		return
	}
	totalAmount := new(big.Int)
	for cur.Next(ctx) {
		var t RedpacketClaim
		if err = cur.Decode(&t); err != nil {
			return
		}
		amount := convert.StrTo(t.Amount).MustBigInt()
		totalAmount.Add(totalAmount, amount)
	}
	return totalAmount.String()
}

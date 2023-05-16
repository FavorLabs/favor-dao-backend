package model

import (
	"context"
	"math/big"

	"favor-dao-backend/pkg/convert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RedpacketType int

const (
	RedpacketTypeLucked RedpacketType = iota
	RedpacketTypeAverage
)

type Redpacket struct {
	DefaultModel `bson:",inline"`
	Address      string        `json:"address"  bson:"address"`
	Title        string        `json:"title"    bson:"title"`
	AvgAmount    string        `json:"avg_amount"   bson:"avg_amount"`
	Amount       string        `json:"amount"   bson:"amount"`
	Type         RedpacketType `json:"type"     bson:"type"`
	Total        int64         `json:"total"    bson:"total"`
	Balance      string        `json:"balance"  bson:"balance"`
	ClaimCount   int64         `json:"claim_count" bson:"claim_count"`
	TxID         string        `json:"tx_id"    bson:"tx_id"`
	PayStatus    PayStatus     `json:"pay_status" bson:"pay_status"`
	RefundTxID   string        `json:"refund_tx_id" bson:"refund_tx_id"`
	RefundStatus PayStatus     `json:"refund_status" bson:"refund_status"`
}

type RedpacketSendFormatted struct {
	Redpacket  `bson:",inline"`
	UserAvatar string `json:"user_avatar" bson:"user_avatar"`
}

type RedpacketViewFormatted struct {
	Redpacket   `bson:",inline"`
	UserAvatar  string `json:"user_avatar" bson:"user_avatar"`
	ClaimAmount string `json:"claim_amount" bson:"claim_amount"`
}

func (a *Redpacket) Table() string {
	return "redpacket"
}

func (a *Redpacket) Create(ctx context.Context, db *mongo.Database) error {
	return create(ctx, db, a)
}

func (a *Redpacket) Update(ctx context.Context, db *mongo.Database) error {
	return update(ctx, db, a)
}

func (a *Redpacket) First(ctx context.Context, db *mongo.Database) error {
	filter := bson.M{ID: a.GetID()}

	return findOne(ctx, db, a, filter)
}

func (a *Redpacket) FindAndUpdate(ctx context.Context, db *mongo.Database, update interface{}) error {
	filter := bson.M{ID: a.GetID()}

	return findAndUpdate(ctx, db, a, filter, update)
}

func (a *Redpacket) FindList(ctx context.Context, db *mongo.Database, match interface{}, limit, offset int) (list []*RedpacketSendFormatted) {
	user := User{}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from": user.Table(),
			"let":  bson.M{"user": "$address"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$address", "$$user"}}}},
				bson.M{"$project": bson.M{"_id": 0, "avatar": 1}},
			},
			"as": "ext",
		}}},
		{{"$match", match}},
		{{"$sort", bson.M{"created_on": -1}}},
		{{"$skip", offset}},
		{{"$limit", limit}},
		{{"$unwind", "$ext"}},
	}
	list = []*RedpacketSendFormatted{}
	cursor, err := db.Collection(a.Table()).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	type tmp struct {
		Redpacket `bson:",inline"`
		Ext       struct {
			Avatar string `bson:"avatar"`
		} `bson:"ext"`
	}
	for cursor.Next(context.TODO()) {
		var t tmp
		if cursor.Decode(&t) != nil {
			return
		}
		list = append(list, &RedpacketSendFormatted{
			Redpacket:  t.Redpacket,
			UserAvatar: t.Ext.Avatar,
		})
	}
	return
}

func (a *Redpacket) Count(ctx context.Context, db *mongo.Database, match interface{}) int64 {
	documents, err := db.Collection(a.Table()).CountDocuments(ctx, match)
	if err != nil {
		return 0
	}
	return documents
}

func (a *Redpacket) CountAmount(ctx context.Context, db *mongo.Database, match interface{}) (total string) {
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

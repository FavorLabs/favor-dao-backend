package model

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PostAttachmentBill struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PostID  primitive.ObjectID `json:"post_id" bson:"post_id"`
	Address string             `json:"address" bson:"address"`
}

func (m *PostAttachmentBill) table() string {
	return "post_attachment_bill"
}
func (p *PostAttachmentBill) Get(db *mongo.Database) (*PostAttachmentBill, error) {
	var pas PostAttachmentBill
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

	res := db.Collection(p.table()).FindOne(context.TODO(), findQuery(queries))
	err := res.Decode(&pas)
	if err != nil {
		return nil, err
	}

	return &pas, nil
}

func (p *PostAttachmentBill) Create(db *mongo.Database) (*PostAttachmentBill, error) {
	res, err := db.Collection(p.table()).InsertOne(context.TODO(), &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

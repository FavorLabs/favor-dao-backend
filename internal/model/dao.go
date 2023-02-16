package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// DaoVisibleT Accessible type, 0 public, 1 private
type DaoVisibleT uint8

const (
	DaoVisitPublic DaoVisibleT = iota
	DaoVisitPrivate
)

type Dao struct {
	ID           primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn    int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn   int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn    int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel        int                `json:"is_del"           bson:"is_del"`
	Address      string             `json:"address"          bson:"address"`
	Name         string             `json:"name"             bson:"name"`
	Visibility   DaoVisibleT        `json:"visibility"       bson:"visibility"`
	Introduction string             `json:"introduction"     bson:"introduction"`
}

func (m *Dao) table() string {
	return "dao"
}

func (m *Dao) Create(ctx context.Context, db *mongo.Database) (*Dao, error) {
	res, err := db.Collection(m.table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *Dao) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	update := bson.D{{"$set", bson.D{{"is_del", 1}}}}
	res := db.Collection(m.table()).FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Dao) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.table()).FindOneAndReplace(ctx, filter, &m)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Dao) Get(ctx context.Context, db *mongo.Database) (*Dao, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}, {"is_del", 0}}
	res := db.Collection(m.table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var dao Dao
	err := res.Decode(&dao)
	if err != nil {
		return nil, err
	}
	return &dao, nil
}

func (m *Dao) FindList(ctx context.Context, db *mongo.Database, filter interface{}) (list []*Dao, err error) {
	cur, err := db.Collection(m.table()).Find(ctx, filter)
	if err != nil {
		return
	}
	var res []*Dao
	err = cur.All(ctx, &res)
	if err != nil {
		return
	}
	return res, nil
}

package model

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

type DaoFormatted struct {
	ID           string      `json:"id"`
	Address      string      `json:"address"`
	Name         string      `json:"name"`
	Introduction string      `json:"introduction"`
	Visibility   DaoVisibleT `json:"visibility"`
	Avatar       string      `json:"avatar"`
}

func (m *Dao) Format() *DaoFormatted {
	return &DaoFormatted{
		ID:           m.ID.Hex(),
		Address:      m.Address,
		Name:         m.Name,
		Introduction: m.Introduction,
		Visibility:   m.Visibility,
	}
}

func (m *Dao) Table() string {
	return "dao"
}

func (m *Dao) Create(ctx context.Context, db *mongo.Database) (*Dao, error) {
	now := time.Now().Unix()
	m.CreatedOn = now
	m.ModifiedOn = now
	res, err := db.Collection(m.Table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *Dao) Delete(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	res := db.Collection(m.Table()).FindOneAndUpdate(ctx, filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (m *Dao) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{{"_id", m.ID}}
	res := db.Collection(m.Table()).FindOneAndReplace(ctx, filter, &m)
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
	res := db.Collection(m.Table()).FindOne(ctx, filter)
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

func (m *Dao) GetListByAddress(ctx context.Context, db *mongo.Database) (list []*DaoFormatted, err error) {
	filter := bson.M{"address": m.Address, "is_del": 0}
	cur, err := db.Collection(m.Table()).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var dao []*Dao
	err = cur.All(ctx, &dao)
	if err != nil {
		return nil, err
	}
	list = []*DaoFormatted{}
	for _, v := range dao {
		list = append(list, v.Format())
	}
	return list, nil
}

func (m *Dao) FindListByKeyword(ctx context.Context, db *mongo.Database, keyword string, offset, limit int) (list []*Dao, err error) {
	var filter bson.M
	if keyword != "" {
		filter = bson.M{
			"name": fmt.Sprintf("/%s/", keyword),
		}
	}
	finds := make([]*options.FindOptions, 0, 2)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	cur, err := db.Collection(m.Table()).Find(ctx, filter, finds...)
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

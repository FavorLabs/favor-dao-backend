package model

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	ID         primitive.ObjectID `json:"id"               bson:"_id,omitempty"`
	CreatedOn  int64              `json:"created_on"       bson:"created_on"`
	ModifiedOn int64              `json:"modified_on"      bson:"modified_on"`
	DeletedOn  int64              `json:"deleted_on"       bson:"deleted_on"`
	IsDel      int                `json:"is_del"           bson:"is_del"`
	Nickname   string             `json:"nickname"         bson:"nickname"`
	Address    string             `json:"address"          bson:"address"`
	Avatar     string             `json:"avatar"           bson:"avatar"`
	Role       string             `json:"role"             bson:"role"`
}

type UserFormatted struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Address  string `json:"address"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
}

func (m *User) Format() *UserFormatted {
	return &UserFormatted{
		ID:       m.ID.Hex(),
		Nickname: m.Nickname,
		Address:  m.Address,
		Avatar:   m.Avatar,
		Role:     m.Role,
	}
}

func (m *User) table() string {
	return "user"
}

func (m *User) Get(ctx context.Context, db *mongo.Database) (*User, error) {
	var (
		user User
		res  *mongo.SingleResult
	)
	if !m.ID.IsZero() {
		filter := bson.D{{"_id", m.ID}, {"is_del", 0}}
		res = db.Collection(m.table()).FindOne(ctx, filter)
	} else if m.Address != "" {
		filter := bson.D{{"address", m.Address}, {"is_del", 0}}
		res = db.Collection(m.table()).FindOne(ctx, filter)
	}
	err := res.Err()
	if err != nil {
		return &user, err
	}
	err = res.Decode(&user)
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (m *User) List(ctx context.Context, db *mongo.Database, pipeline bson.A) (users []*User, err error) {
	cur, err := db.Collection(m.table()).Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	err = cur.All(ctx, &users)
	return
}

func (m *User) Create(ctx context.Context, db *mongo.Database) (*User, error) {
	res, err := db.Collection(m.table()).InsertOne(ctx, &m)
	if err != nil {
		return nil, err
	}
	m.ID = res.InsertedID.(primitive.ObjectID)
	return m, nil
}

func (m *User) Update(ctx context.Context, db *mongo.Database) error {
	filter := bson.D{
		{"$or", bson.A{
			bson.M{"_id": m.ID},
			bson.M{"address": m.Address},
		},
		},
	}
	res := db.Collection(m.table()).FindOneAndReplace(ctx, filter, &m)
	return res.Err()
}

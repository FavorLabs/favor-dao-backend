package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	chatModel "favor-dao-backend/internal/model/chat"
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

type DaoType uint8

const (
	DaoDefault DaoType = iota
	DaoWithURL
)

var (
	ErrDuplicateDAOName = errors.New("DAO name duplicate")
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
	Avatar       string             `json:"avatar"           bson:"avatar"`
	Banner       string             `json:"banner"           bson:"banner"`
	HomePage     string             `json:"home_page,omitempty"        bson:"homePage,omitempty"`
	FollowCount  int64              `json:"follow_count"     bson:"follow_count"`
	Price        string             `json:"price"            bson:"price"`
	Tags         string             `json:"tags"             bson:"tags"`
	Type         DaoType            `json:"type"             bson:"type,omitempty"`
}

type DaoFormatted struct {
	ID           string           `json:"id"`
	Address      string           `json:"address"`
	Name         string           `json:"name"`
	Introduction string           `json:"introduction"`
	Visibility   DaoVisibleT      `json:"visibility"`
	Avatar       string           `json:"avatar"`
	Banner       string           `json:"banner"`
	HomePage     string           `json:"home_page,omitempty"`
	FollowCount  int64            `json:"follow_count"`
	Price        string           `json:"price"`
	Tags         map[string]int8  `json:"tags"`
	Type         DaoType          `json:"type"`
	LastPosts    []*PostFormatted `json:"last_posts"`
	IsJoined     bool             `json:"is_joined"`
	IsSubscribed bool             `json:"is_subscribed"`
}

func (m *Dao) Format() *DaoFormatted {
	tagsMap := map[string]int8{}
	for _, tag := range strings.Split(m.Tags, ",") {
		tagsMap[tag] = 1
	}
	return &DaoFormatted{
		ID:           m.ID.Hex(),
		Address:      m.Address,
		Name:         m.Name,
		Introduction: m.Introduction,
		Visibility:   m.Visibility,
		Avatar:       m.Avatar,
		Banner:       m.Banner,
		HomePage:     m.HomePage,
		FollowCount:  m.FollowCount,
		Price:        m.Price,
		Tags:         tagsMap,
		Type:         m.Type,
		LastPosts:    []*PostFormatted{},
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

func (m *Dao) Count(db *mongo.Database, conditions ConditionsT) (int64, error) {

	var query bson.M
	if m.Address != "" {
		query = bson.M{"address": m.Address}
	}
	for k, v := range conditions {
		if k != "ORDER" {
			if query != nil {
				query = findQuery([]bson.M{query, v})
			} else {
				query = findQuery([]bson.M{v})
			}
		}
	}
	count, err := db.Collection(m.Table()).CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (m *Dao) List(db *mongo.Database, conditions ConditionsT, offset, limit int) ([]*Dao, error) {
	var (
		posts  []*Dao
		err    error
		cursor *mongo.Cursor
		query  bson.M
	)
	if m.Address != "" {
		query = bson.M{"address": m.Address}
	}
	finds := make([]*options.FindOptions, 0, 3)
	finds = append(finds, options.Find().SetSkip(int64(offset)))
	finds = append(finds, options.Find().SetLimit(int64(limit)))
	if len(conditions) == 0 {
		if query != nil {
			query = findQuery([]bson.M{query})
		} else {
			query = bson.M{"is_del": 0}
		}
	}
	for k, v := range conditions {
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
	if cursor, err = db.Collection(m.Table()).Find(context.TODO(), query, finds...); err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var post Dao
		if cursor.Decode(&post) != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	return posts, nil
}

func (m *Dao) Get(ctx context.Context, db *mongo.Database) (*Dao, error) {
	if m.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", m.ID}}
	if m.IsDel == 0 {
		filter = append(filter, bson.E{Key: "is_del", Value: 0})
	}
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

func (m *Dao) GetByName(ctx context.Context, db *mongo.Database) (*DaoFormatted, error) {
	filter := bson.M{"name": m.Name}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var dao *Dao
	err := res.Decode(&dao)
	if err != nil {
		return nil, err
	}
	return dao.Format(), nil
}

func (m *Dao) CheckNameDuplication(ctx context.Context, db *mongo.Database) bool {
	filter := bson.M{"name": m.Name, ID: bson.M{"$ne": m.ID}}
	res := db.Collection(m.Table()).FindOne(ctx, filter)
	if res.Err() == nil {
		return true
	}
	return false
}

func (m *Dao) GetListByAddress(ctx context.Context, db *mongo.Database) (list []*DaoFormatted, err error) {
	filter := bson.M{"address": m.Address, "is_del": 0}
	cursor, err := db.Collection(m.Table()).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.TODO()) {
		var t Dao
		if cursor.Decode(&t) != nil {
			return
		}
		list = append(list, t.Format())
	}
	return
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
	cursor, err := db.Collection(m.Table()).Find(ctx, filter, finds...)
	if err != nil {
		return
	}
	for cursor.Next(context.TODO()) {
		var t Dao
		if cursor.Decode(&t) != nil {
			return
		}
		list = append(list, &t)
	}
	return
}

func (m *Dao) RealDelete(ctx context.Context, db *mongo.Database, gid string) error {
	if !m.ID.IsZero() {
		table := []string{
			new(DaoBookmark).Table(),
		}
		for _, v := range table {
			_, err := db.Collection(v).DeleteMany(ctx, bson.M{"dao_id": m.ID})
			if err != nil {
				return err
			}
		}
		_, err := db.Collection(new(PostBlock).Table()).DeleteMany(ctx, bson.M{"block_id": m.ID, "model": BlockModelDAO})
		if err != nil {
			return err
		}
		group := &chatModel.Group{
			ID: chatModel.GroupID{
				DaoID:   m.ID,
				GroupID: gid,
				OwnerID: m.Address,
			},
		}
		err = group.Delete(ctx, db)
		if err != nil {
			return err
		}
		_, err = db.Collection(m.Table()).DeleteOne(ctx, bson.M{"_id": m.ID})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Dao) GetUser(ctx context.Context, db *mongo.Database) (*User, error) {
	if m.ID.IsZero() {
		return nil, errors.New("NO DAO ID")
	}
	user := &User{}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         user.Table(),
			"localField":   "address",
			"foreignField": "address",
			"as":           "user",
		}}},
		{{"$match", bson.M{ID: m.ID}}},
		{{"$unwind", "$user"}},
	}
	cursor, err := db.Collection(m.Table()).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	type tmp struct {
		User *User `bson:"user"`
	}

	for cursor.Next(context.TODO()) {
		var t tmp
		if err = cursor.Decode(&t); err != nil {
			return nil, err
		}
		return t.User, nil
	}
	return nil, mongo.ErrNoDocuments
}

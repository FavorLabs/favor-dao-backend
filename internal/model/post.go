package model

import (
	"context"
	"strings"
	"time"

	"favor-dao-backend/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PostVisibleT Accessible type, 0 public, 1 private, 2 friends
type PostVisibleT uint8

const (
	PostVisitDraft PostVisibleT = iota
	PostVisitPublic
	PostVisitPrivate
	// PostVisitSecret
	// PostVisitFriend
	// PostVisitInvalid
)

type Post struct {
	ID              primitive.ObjectID `json:"id"                bson:"_id,omitempty"`
	CreatedOn       int64              `json:"created_on"        bson:"created_on"`
	ModifiedOn      int64              `json:"modified_on"       bson:"modified_on"`
	DeletedOn       int64              `json:"deleted_on"        bson:"deleted_on"`
	IsDel           int                `json:"is_del"            bson:"is_del"`
	LatestRepliedOn int64              `json:"latest_replied_on" bson:"latest_replied_on"`
	Address         string             `json:"address"           bson:"address"`
	DaoId           primitive.ObjectID `json:"dao_id"            bson:"dao_id"`
	ViewCount       int64              `json:"view_count"        bson:"view_count"`
	CollectionCount int64              `json:"collection_count"  bson:"collection_count"`
	UpvoteCount     int64              `json:"upvote_count"      bson:"upvote_count"`
	CommentCount    int64              `json:"comment_count"     bson:"comment_count"`
	RefCount        int64              `json:"ref_count"         bson:"ref_count"`
	Member          int                `json:"member"            bson:"member"`
	Visibility      PostVisibleT       `json:"visibility"        bson:"visibility"`
	IsTop           int                `json:"is_top"            bson:"is_top"`
	IsEssence       int                `json:"is_essence"        bson:"is_essence"`
	Tags            string             `json:"tags"              bson:"tags"`
	Type            PostType           `json:"type"              bson:"type"`
	OrigType        PostType           `json:"orig_type"         bson:"orig_type"`
	AuthorId        string             `json:"author_id"         bson:"author_id"`
	AuthorDaoId     primitive.ObjectID `json:"author_dao_id"     bson:"author_dao_id"`
	RefId           primitive.ObjectID `json:"ref_id"            bson:"ref_id"`
	RefType         PostRefType        `json:"ref_type"          bson:"ref_type"`
}

type PostFormatted struct {
	ID              primitive.ObjectID      `json:"id"`
	CreatedOn       int64                   `json:"created_on"`
	ModifiedOn      int64                   `json:"modified_on"`
	LatestRepliedOn int64                   `json:"latest_replied_on"`
	DaoId           primitive.ObjectID      `json:"dao_id"`
	Dao             *DaoFormatted           `json:"dao"`
	Address         string                  `json:"address"`
	User            *UserFormatted          `json:"user"`
	Contents        []*PostContentFormatted `json:"contents"`
	OrigContents    []*PostContentFormatted `json:"orig_contents"`
	Member          int                     `json:"member"`
	ViewCount       int64                   `json:"view_count"`
	CollectionCount int64                   `json:"collection_count"`
	UpvoteCount     int64                   `json:"upvote_count"`
	CommentCount    int64                   `json:"comment_count"`
	RefCount        int64                   `json:"ref_count"`
	Visibility      PostVisibleT            `json:"visibility"`
	IsTop           int                     `json:"is_top"`
	IsEssence       int                     `json:"is_essence"`
	Tags            map[string]int8         `json:"tags"`
	Type            PostType                `json:"type"`
	OrigType        PostType                `json:"orig_type"`
	AuthorId        string                  `json:"author_id"`
	AuthorDaoId     primitive.ObjectID      `json:"author_dao_id"`
	Author          *UserFormatted          `json:"author"`
	AuthorDao       *DaoFormatted           `json:"author_dao"`
	RefId           primitive.ObjectID      `json:"ref_id"`
	RefType         PostRefType             `json:"ref_type"`
}

func (p *Post) Table() string {
	return "post"
}

func (p *Post) Format() *PostFormatted {
	tagsMap := map[string]int8{}
	for _, tag := range strings.Split(p.Tags, ",") {
		tagsMap[tag] = 1
	}
	return &PostFormatted{
		ID:              p.ID,
		DaoId:           p.DaoId,
		Dao:             &DaoFormatted{},
		Address:         p.Address,
		User:            &UserFormatted{},
		Contents:        []*PostContentFormatted{},
		OrigContents:    []*PostContentFormatted{},
		Member:          p.Member,
		ViewCount:       p.ViewCount,
		CollectionCount: p.CollectionCount,
		UpvoteCount:     p.UpvoteCount,
		CommentCount:    p.CommentCount,
		RefCount:        p.RefCount,
		Visibility:      p.Visibility,
		IsTop:           p.IsTop,
		IsEssence:       p.IsEssence,
		Tags:            tagsMap,
		Type:            p.Type,
		OrigType:        p.OrigType,
		CreatedOn:       p.CreatedOn,
		AuthorId:        p.AuthorId,
		AuthorDaoId:     p.AuthorDaoId,
		Author:          &UserFormatted{},
		AuthorDao:       &DaoFormatted{},
		RefId:           p.RefId,
		RefType:         p.RefType,
	}
}

func (p *Post) Create(ctx context.Context, db *mongo.Database) (*Post, error) {
	now := time.Now().Unix()
	p.CreatedOn = now
	p.ModifiedOn = now
	res, err := db.Collection(p.Table()).InsertOne(ctx, &p)
	if err != nil {
		return nil, err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return p, err
}

func (p *Post) Delete(db *mongo.Database) error {
	filter := bson.D{{"_id", p.ID}}
	update := bson.D{{"$set", bson.D{
		{"is_del", 1},
		{"deleted_on", time.Now().Unix()},
	}}}
	res := db.Collection(p.Table()).FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (p *Post) Get(ctx context.Context, db *mongo.Database) (*Post, error) {
	if p.ID.IsZero() {
		return nil, mongo.ErrNoDocuments
	}
	filter := bson.D{{"_id", p.ID}, {"is_del", 0}}
	res := db.Collection(p.Table()).FindOne(ctx, filter)
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
	if len(*conditions) == 0 {
		if query != nil {
			query = findQuery([]bson.M{query})
		} else {
			query = bson.M{"is_del": 0}
		}
	}
	for k, v := range *conditions {
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
	if cursor, err = db.Collection(p.Table()).Find(context.TODO(), query, finds...); err != nil {
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
			if query != nil {
				query = findQuery([]bson.M{query, v})
			} else {
				query = findQuery([]bson.M{v})
			}
		}
	}
	count, err := db.Collection(p.Table()).CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *Post) Update(ctx context.Context, db *mongo.Database) error {
	p.ModifiedOn = time.Now().Unix()
	filter := bson.D{{"_id", p.ID}, {"is_del", 0}}
	update := bson.M{"$set": p}
	if _, err := db.Collection(p.Table()).UpdateMany(ctx, filter, update); err != nil {
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

func (p *Post) RealDelete(ctx context.Context, db *mongo.Database) error {
	return util.MongoTransaction(ctx, db, func(ctx context.Context) error {
		if !p.ID.IsZero() {
			table := []string{
				new(PostContent).Table(),
				new(PostCollection).Table(),
				new(PostStar).Table(),
				new(Comment).Table(),
			}
			for _, v := range table {
				_, err := db.Collection(v).DeleteMany(ctx, bson.M{"post_id": p.ID})
				if err != nil {
					return err
				}
			}
			_, err := db.Collection(p.Table()).DeleteOne(ctx, bson.M{"_id": p.ID})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

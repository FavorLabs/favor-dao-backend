package core

import (
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TweetService interface {
	GetPostByID(id primitive.ObjectID) (*model.Post, error)
	GetPosts(conditions *model.ConditionsT, offset, limit int) ([]*model.Post, error)
	GetPostCount(conditions *model.ConditionsT) (int64, error)
	GetUserPostStar(postID primitive.ObjectID, userID string) (*model.PostStar, error)
	GetUserPostStars(address string, offset, limit int) ([]*model.PostStarFormatted, error)
	GetUserPostStarCount(address string) (int64, error)
	GetUserPostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error)
	GetUserPostCollections(address string, offset, limit int) ([]*model.PostCollection, error)
	GetUserPostCollectionCount(address string) (int64, error)
	GetPostContentsByIDs(ids []primitive.ObjectID) ([]*model.PostContent, error)
	GetPostContentByID(id primitive.ObjectID) ([]*model.PostContent, error)
}

type TweetManageService interface {
	CreatePost(post *model.Post, contents []*model.PostContent) (*model.Post, error)
	DeletePost(post *model.Post) ([]string, error)
	StickPost(post *model.Post) error
	VisiblePost(post *model.Post, visibility model.PostVisibleT) error
	UpdatePost(post *model.Post) error
	CreatePostStar(postID primitive.ObjectID, address string) (*model.PostStar, error)
	DeletePostStar(p *model.PostStar) error
	CreatePostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error)
	DeletePostCollection(p *model.PostCollection) error
}

type TweetHelpService interface {
	RevampPosts(posts []*model.PostFormatted) ([]*model.PostFormatted, error)
	MergePosts(posts []*model.Post) ([]*model.PostFormatted, error)
}

type IndexPostsService interface {
	IndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error)
}

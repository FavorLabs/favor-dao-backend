package core

import (
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
)

type TweetService interface {
	GetPostByID(id int64) (*model.Post, error)
	GetPosts(conditions *model.ConditionsT, offset, limit int) ([]*model.Post, error)
	GetPostCount(conditions *model.ConditionsT) (int64, error)
	GetUserPostStar(postID, userID int64) (*model.PostStar, error)
	GetUserPostStars(userID int64, offset, limit int) ([]*model.PostStar, error)
	GetUserPostStarCount(userID int64) (int64, error)
	GetUserPostCollection(postID, userID int64) (*model.PostCollection, error)
	GetUserPostCollections(userID int64, offset, limit int) ([]*model.PostCollection, error)
	GetUserPostCollectionCount(userID int64) (int64, error)
	GetPostAttatchmentBill(postID, userID int64) (*model.PostAttachmentBill, error)
	GetPostContentsByIDs(ids []int64) ([]*model.PostContent, error)
	GetPostContentByID(id int64) (*model.PostContent, error)
}

type TweetManageService interface {
	CreatePost(post *model.Post) (*model.Post, error)
	DeletePost(post *model.Post) ([]string, error)
	StickPost(post *model.Post) error
	VisiblePost(post *model.Post, visibility model.PostVisibleT) error
	UpdatePost(post *model.Post) error
	CreatePostStar(postID, userID int64) (*model.PostStar, error)
	DeletePostStar(p *model.PostStar) error
	CreatePostCollection(postID, userID int64) (*model.PostCollection, error)
	DeletePostCollection(p *model.PostCollection) error
	CreatePostContent(content *model.PostContent) (*model.PostContent, error)
}

type TweetHelpService interface {
	RevampPosts(posts []*model.PostFormated) ([]*model.PostFormated, error)
	MergePosts(posts []*model.Post) ([]*model.PostFormated, error)
}

type IndexPostsService interface {
	IndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error)
}

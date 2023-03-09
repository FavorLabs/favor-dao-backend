package monogo

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.IndexPostsService = (*indexPostsServant)(nil)
	_ core.IndexPostsService = (*simpleIndexPostsServant)(nil)
)

type indexPostsServant struct {
	ams core.AuthorizationManageService
	ths core.TweetHelpService
	db  *mongo.Database
}

type simpleIndexPostsServant struct {
	ths core.TweetHelpService
	db  *mongo.Database
}

func newIndexPostsService(db *mongo.Database) core.IndexPostsService {
	return &indexPostsServant{
		ams: NewAuthorizationManageService(),
		ths: newTweetHelpService(db),
		db:  db,
	}
}

func newSimpleIndexPostsService(db *mongo.Database) core.IndexPostsService {
	return &simpleIndexPostsServant{
		ths: newTweetHelpService(db),
		db:  db,
	}
}

// IndexPosts querying the list of square tweets according to userId, simply so that the home pages are different for different users.
func (s *indexPostsServant) IndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error) {
	predicates := model.ConditionsT{
		"ORDER": bson.M{"is_top": -1},
	}
	if user == nil {
		predicates["query"] = bson.M{"visibility": model.PostVisitPublic}
	} else {
		predicates["query"] = bson.M{"visibility": model.PostVisitPublic,
			"$or": bson.M{"visibility": model.PostVisitPrivate, "address": user.Address}}
	}

	posts, err := (&model.Post{}).List(s.db, &predicates, offset, limit)
	if err != nil {
		logrus.Debugf("gormIndexPostsServant.IndexPosts err: %v", err)
		return nil, err
	}
	formatPosts, err := s.ths.MergePosts(posts)
	if err != nil {
		return nil, err
	}

	total, err := (&model.Post{}).Count(s.db, &predicates)
	if err != nil {
		return nil, err
	}

	return &rest.IndexTweetsResp{
		Tweets: formatPosts,
		Total:  total,
	}, nil
}

// IndexPosts simpleCacheIndexGetPosts simpleCacheIndex Proprietary get square tweet list function
func (s *simpleIndexPostsServant) IndexPosts(_user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error) {
	predicates := model.ConditionsT{
		"query": bson.M{"visibility": model.PostVisitPublic},
		"ORDER": bson.M{"is_top": -1},
	}

	posts, err := (&model.Post{}).List(s.db, &predicates, offset, limit)
	if err != nil {
		logrus.Debugf("gormSimpleIndexPostsServant.IndexPosts err: %v", err)
		return nil, err
	}

	formatPosts, err := s.ths.MergePosts(posts)
	if err != nil {
		return nil, err
	}

	total, err := (&model.Post{}).Count(s.db, &predicates)
	if err != nil {
		return nil, err
	}

	return &rest.IndexTweetsResp{
		Tweets: formatPosts,
		Total:  total,
	}, nil
}

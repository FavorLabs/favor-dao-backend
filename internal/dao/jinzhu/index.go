package jinzhu

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"favor-dao-backend/pkg/types"
	"github.com/sirupsen/logrus"
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
	predicates := model.Predicates{
		"ORDER": types.AnySlice{"is_top DESC, latest_replied_on DESC"},
	}
	if user == nil {
		predicates["visibility = ?"] = []types.Any{model.PostVisitPublic}
	} else if !user.IsAdmin {
		friendIds, _ := s.ams.BeFriendIds(user.ID)
		friendIds = append(friendIds, user.ID)
		args := types.AnySlice{model.PostVisitPublic, model.PostVisitPrivate, user.ID, model.PostVisitFriend, friendIds}
		predicates["visibility = ? OR (visibility = ? AND user_id = ?) OR (visibility = ? AND user_id IN ?)"] = args
	}

	posts, err := (&model.Post{}).Fetch(s.db, predicates, offset, limit)
	if err != nil {
		logrus.Debugf("gormIndexPostsServant.IndexPosts err: %v", err)
		return nil, err
	}
	formatPosts, err := s.ths.MergePosts(posts)
	if err != nil {
		return nil, err
	}

	total, err := (&model.Post{}).CountBy(s.db, predicates)
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
	predicates := model.Predicates{
		"visibility = ?": []types.Any{model.PostVisitPublic},
		"ORDER":          []types.Any{"is_top DESC, latest_replied_on DESC"},
	}

	posts, err := (&model.Post{}).Fetch(s.db, predicates, offset, limit)
	if err != nil {
		logrus.Debugf("gormSimpleIndexPostsServant.IndexPosts err: %v", err)
		return nil, err
	}

	formatPosts, err := s.ths.MergePosts(posts)
	if err != nil {
		return nil, err
	}

	total, err := (&model.Post{}).CountBy(s.db, predicates)
	if err != nil {
		return nil, err
	}

	return &rest.IndexTweetsResp{
		Tweets: formatPosts,
		Total:  total,
	}, nil
}

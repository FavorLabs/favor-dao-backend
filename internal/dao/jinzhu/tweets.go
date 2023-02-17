package jinzhu

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.TweetService       = (*tweetServant)(nil)
	_ core.TweetManageService = (*tweetManageServant)(nil)
	_ core.TweetHelpService   = (*tweetHelpServant)(nil)
)

type tweetServant struct {
	db *mongo.Database
}

type tweetManageServant struct {
	cacheIndex core.CacheIndexService
	db         *mongo.Database
}

type tweetHelpServant struct {
	db *mongo.Database
}

func newTweetService(db *mongo.Database) core.TweetService {
	return &tweetServant{
		db: db,
	}
}

func newTweetManageService(db *mongo.Database, cacheIndex core.CacheIndexService) core.TweetManageService {
	return &tweetManageServant{
		cacheIndex: cacheIndex,
		db:         db,
	}
}

func newTweetHelpService(db *mongo.Database) core.TweetHelpService {
	return &tweetHelpServant{
		db: db,
	}
}

// MergePosts post data integration
func (s *tweetHelpServant) MergePosts(posts []*model.Post) ([]*model.PostFormated, error) {
	postIds := make([]int64, 0, len(posts))
	userIds := make([]int64, 0, len(posts))
	for _, post := range posts {
		postIds = append(postIds, post.ID)
		userIds = append(userIds, post.UserID)
	}

	postContents, err := s.getPostContentsByIDs(postIds)
	if err != nil {
		return nil, err
	}

	users, err := s.getUsersByIDs(userIds)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*model.UserFormated, len(users))
	for _, user := range users {
		userMap[user.ID] = user.Format()
	}

	contentMap := make(map[int64][]*model.PostContentFormated, len(postContents))
	for _, content := range postContents {
		contentMap[content.PostID] = append(contentMap[content.PostID], content.Format())
	}

	// data integration
	postsFormated := make([]*model.PostFormated, 0, len(posts))
	for _, post := range posts {
		postFormated := post.Format()
		postFormated.User = userMap[post.UserID]
		postFormated.Contents = contentMap[post.ID]
		postsFormated = append(postsFormated, postFormated)
	}
	return postsFormated, nil
}

// RevampPosts post data shaping repair
func (s *tweetHelpServant) RevampPosts(posts []*model.PostFormated) ([]*model.PostFormated, error) {
	postIds := make([]int64, 0, len(posts))
	userIds := make([]int64, 0, len(posts))
	for _, post := range posts {
		postIds = append(postIds, post.ID)
		userIds = append(userIds, post.UserID)
	}

	postContents, err := s.getPostContentsByIDs(postIds)
	if err != nil {
		return nil, err
	}

	users, err := s.getUsersByIDs(userIds)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*model.UserFormated, len(users))
	for _, user := range users {
		userMap[user.ID] = user.Format()
	}

	contentMap := make(map[int64][]*model.PostContentFormated, len(postContents))
	for _, content := range postContents {
		contentMap[content.PostID] = append(contentMap[content.PostID], content.Format())
	}

	// data integration
	for _, post := range posts {
		post.User = userMap[post.UserID]
		post.Contents = contentMap[post.ID]
	}
	return posts, nil
}

func (s *tweetHelpServant) getPostContentsByIDs(ids []int64) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"post_id IN ?": ids,
		"ORDER":        "sort ASC",
	}, 0, 0)
}

func (s *tweetHelpServant) getUsersByIDs(ids []int64) ([]*model.User, error) {
	user := &model.User{}

	return user.List(s.db, &model.ConditionsT{
		"id IN ?": ids,
	}, 0, 0)
}

func (s *tweetManageServant) CreatePostCollection(postID, userID int64) (*model.PostCollection, error) {
	collection := &model.PostCollection{
		PostID: postID,
		UserID: userID,
	}

	return collection.Create(s.db)
}

func (s *tweetManageServant) DeletePostCollection(p *model.PostCollection) error {
	return p.Delete(s.db)
}

func (s *tweetManageServant) CreatePostContent(content *model.PostContent) (*model.PostContent, error) {
	return content.Create(s.db)
}

func (s *tweetManageServant) CreatePost(post *model.Post) (*model.Post, error) {
	post.LatestRepliedOn = time.Now().Unix()
	p, err := post.Create(s.db)
	if err != nil {
		return nil, err
	}
	s.cacheIndex.SendAction(core.IdxActCreatePost, post)
	return p, nil
}

func (s *tweetManageServant) DeletePost(post *model.Post) ([]string, error) {
	var mediaContents []string

	postId := post.ID
	postContent := &model.PostContent{}
	err := s.db.Transaction(
		func(tx *mongo.Database) error {
			if contents, err := postContent.MediaContentsByPostId(tx, postId); err == nil {
				mediaContents = contents
			} else {
				return err
			}

			// delete post
			if err := post.Delete(tx); err != nil {
				return err
			}

			// delete post content
			if err := postContent.DeleteByPostId(tx, postId); err != nil {
				return err
			}

			// delete post comment
			if contents, err := s.deleteCommentByPostId(tx, postId); err == nil {
				mediaContents = append(mediaContents, contents...)
			} else {
				return err
			}

			if tags := strings.Split(post.Tags, ","); len(tags) > 0 {
				// Delete tag, handle errors loosely, no rollback with errors
				deleteTags(tx, tags)
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	s.cacheIndex.SendAction(core.IdxActDeletePost, post)
	return mediaContents, nil
}

func (s *tweetManageServant) deleteCommentByPostId(db *mongo.Database, postId int64) ([]string, error) {
	comment := &model.Comment{}
	commentContent := &model.CommentContent{}

	// Get all comment ids for a tweet
	commentIds, err := comment.CommentIdsByPostId(db, postId)
	if err != nil {
		return nil, err
	}

	// Get media content for reviews
	mediaContents, err := commentContent.MediaContentsByCommentId(db, commentIds)
	if err != nil {
		return nil, err
	}

	// delete comment
	if err = comment.DeleteByPostId(db, postId); err != nil {
		return nil, err
	}

	// delete comment content
	if err = commentContent.DeleteByCommentIds(db, commentIds); err != nil {
		return nil, err
	}

	// Delete comments of comments
	if err = (&model.CommentReply{}).DeleteByCommentIds(db, commentIds); err != nil {
		return nil, err
	}

	return mediaContents, nil
}

func (s *tweetManageServant) StickPost(post *model.Post) error {
	post.IsTop = 1 - post.IsTop
	if err := post.Update(s.db); err != nil {
		return err
	}
	s.cacheIndex.SendAction(core.IdxActStickPost, post)
	return nil
}

func (s *tweetManageServant) VisiblePost(post *model.Post, visibility model.PostVisibleT) error {
	oldVisibility := post.Visibility
	post.Visibility = visibility
	// TODO: Can this judgment be left out?
	if oldVisibility == visibility {
		return nil
	}
	// Private tweets Special handling
	if visibility == model.PostVisitPrivate {
		// Forced untopping
		// TODO: Do users of top tweets have the right to set them to private? Follow up
		post.IsTop = 0
	}
	db := s.db.Begin()
	err := post.Update(db)
	if err != nil {
		db.Rollback()
		return err
	}

	// tag processing
	tags := strings.Split(post.Tags, ",")
	for _, t := range tags {
		tag := &model.Tag{
			Tag: t,
		}
		// TODO: Temporary leniency does not deal with errors, here perhaps there can be optimization, the subsequent refinement
		if oldVisibility == model.PostVisitPrivate {
			// You need to recreate the tag only when you go from private to non-private
			createTag(db, tag)
		} else if visibility == model.PostVisitPrivate {
			// You need to delete the tag only when you go from non-private to private
			deleteTag(db, tag)
		}
	}
	db.Commit()
	s.cacheIndex.SendAction(core.IdxActVisiblePost, post)
	return nil
}

func (s *tweetManageServant) UpdatePost(post *model.Post) error {
	if err := post.Update(s.db); err != nil {
		return err
	}
	s.cacheIndex.SendAction(core.IdxActUpdatePost, post)
	return nil
}

func (s *tweetManageServant) CreatePostStar(postID, userID int64) (*model.PostStar, error) {
	star := &model.PostStar{
		PostID: postID,
		UserID: userID,
	}
	return star.Create(s.db)
}

func (s *tweetManageServant) DeletePostStar(p *model.PostStar) error {
	return p.Delete(s.db)
}

func (s *tweetServant) GetPostByID(id int64) (*model.Post, error) {
	post := &model.Post{
		Model: &model.Model{
			ID: id,
		},
	}
	return post.Get(s.db)
}

func (s *tweetServant) GetPosts(conditions *model.ConditionsT, offset, limit int) ([]*model.Post, error) {
	return (&model.Post{}).List(s.db, conditions, offset, limit)
}

func (s *tweetServant) GetPostCount(conditions *model.ConditionsT) (int64, error) {
	return (&model.Post{}).Count(s.db, conditions)
}

func (s *tweetServant) GetUserPostStar(postID, userID int64) (*model.PostStar, error) {
	star := &model.PostStar{
		PostID: postID,
		UserID: userID,
	}
	return star.Get(s.db)
}

func (s *tweetServant) GetUserPostStars(userID int64, offset, limit int) ([]*model.PostStar, error) {
	star := &model.PostStar{
		UserID: userID,
	}

	return star.List(s.db, &model.ConditionsT{
		"ORDER": s.db.NamingStrategy.TableName("PostStar") + ".id DESC",
	}, offset, limit)
}

func (s *tweetServant) GetUserPostStarCount(userID int64) (int64, error) {
	star := &model.PostStar{
		UserID: userID,
	}
	return star.Count(s.db, &model.ConditionsT{})
}

func (s *tweetServant) GetUserPostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error) {
	star := &model.PostCollection{
		PostID:  postID,
		Address: address,
	}
	star, err := star.Get(s.db)
	post := &model.Post{ID: postID}
	post, err = post.Get(s.db)
	star.Post = post
	return star, err
}

func (s *tweetServant) GetUserPostCollections(address string, offset, limit int) ([]*model.PostCollection, error) {
	collection := &model.PostCollection{
		Address: address,
	}

	pc, err := collection.List(s.db, &model.ConditionsT{
		"ORDER": bson.M{"_id": -1},
	}, offset, limit)

	for _, p := range pc {
		post := &model.Post{ID: p.PostID}
		post, err = post.Get(s.db)
		p.Post = post
	}

	return pc, err
}

func (s *tweetServant) GetUserPostCollectionCount(userID int64) (int64, error) {
	collection := &model.PostCollection{
		UserID: userID,
	}
	return collection.Count(s.db, &model.ConditionsT{})
}

func (s *tweetServant) GetUserWalletBills(userID int64, offset, limit int) ([]*model.WalletStatement, error) {
	statement := &model.WalletStatement{
		UserID: userID,
	}

	return statement.List(s.db, &model.ConditionsT{
		"ORDER": "id DESC",
	}, offset, limit)
}

func (s *tweetServant) GetUserWalletBillCount(userID int64) (int64, error) {
	statement := &model.WalletStatement{
		UserID: userID,
	}
	return statement.Count(s.db, &model.ConditionsT{})
}

func (s *tweetServant) GetPostAttatchmentBill(postID, userID int64) (*model.PostAttachmentBill, error) {
	bill := &model.PostAttachmentBill{
		PostID: postID,
		UserID: userID,
	}

	return bill.Get(s.db)
}

func (s *tweetServant) GetPostContentsByIDs(ids []int64) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"post_id IN ?": ids,
		"ORDER":        "sort ASC",
	}, 0, 0)
}

func (s *tweetServant) GetPostContentByID(id int64) (*model.PostContent, error) {
	return (&model.PostContent{
		Model: &model.Model{
			ID: id,
		},
	}).Get(s.db)
}

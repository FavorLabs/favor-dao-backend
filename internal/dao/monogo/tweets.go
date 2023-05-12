package monogo

import (
	"context"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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
func (s *tweetHelpServant) MergePosts(user string, posts []*model.Post) ([]*model.PostFormatted, error) {
	postIds := make([]primitive.ObjectID, len(posts))
	refItems := make(map[primitive.ObjectID]model.PostRefType, 0)
	addresses := make([]string, len(posts))
	daoIds := make([]primitive.ObjectID, len(posts))
	for i, post := range posts {
		if post.Type == model.Retweet || post.Type == model.RetweetComment {
			refItems[post.RefId] = post.RefType
		}
		daoIds[i] = post.DaoId
		postIds[i] = post.ID
		addresses[i] = post.Address
	}

	postContents, err := s.getPostContentsByIDs(postIds)
	if err != nil {
		return nil, err
	}

	users, err := s.getUsersByAddress(addresses)
	if err != nil {
		return nil, err
	}

	daoS, err := s.getDAOsByIds(daoIds)
	if err != nil {
		return nil, err
	}

	joined := s.getDAOsByIdsWithJoined(user, daoIds)
	joinedMap := make(map[string]struct{}, len(joined))
	for _, v := range joined {
		joinedMap[v.DaoID.Hex()] = struct{}{}
	}
	subscribed := s.getDAOsByIdsWithSubscribed(user, daoIds)
	subscribedMap := make(map[string]struct{}, len(subscribed))
	for _, v := range subscribed {
		subscribedMap[v.DaoID.Hex()] = struct{}{}
	}

	userMap := make(map[string]*model.UserFormatted, len(users))
	for _, user := range users {
		userMap[user.Address] = user.Format()
	}

	daoMap := make(map[string]*model.DaoFormatted, len(daoS))
	for _, dao := range daoS {
		daoMap[dao.ID.Hex()] = dao.Format()
		if _, ok := joinedMap[dao.ID.Hex()]; ok {
			daoMap[dao.ID.Hex()].IsJoined = true
		}
		if _, ok := subscribedMap[dao.ID.Hex()]; ok {
			daoMap[dao.ID.Hex()].IsSubscribed = true
		}
	}

	contentMap := make(map[primitive.ObjectID][]*model.PostContentFormatted, len(postContents))
	for _, content := range postContents {
		contentMap[content.PostID] = append(contentMap[content.PostID], content.Format())
	}

	// data integration
	postsFormatted := make([]*model.PostFormatted, len(posts))
	for i, post := range posts {
		postFormatted := post.Format()
		postFormatted.User = userMap[post.Address]
		postFormatted.Dao = daoMap[post.DaoId.Hex()]

		if content, ok := contentMap[post.ID]; ok {
			postFormatted.Contents = content
		}

		if postFormatted.Type == model.Retweet || postFormatted.Type == model.RetweetComment {
			switch post.RefType {
			case model.RefPost:
				refContents, err := s.getPostContentsByID(post.RefId)
				if err != nil {
					return nil, err
				}
				refContentsFormatted := make([]*model.PostContentFormatted, len(refContents))
				for i := range refContentsFormatted {
					refContentsFormatted[i] = refContents[i].Format()
				}
				postFormatted.OrigContents = append(postFormatted.OrigContents, refContentsFormatted...)
			case model.RefComment:
				refComments, err := s.getCommentContentsByID(post.RefId)
				if err != nil {
					return nil, err
				}
				refCommentsFormatted := make([]*model.PostContentFormatted, len(refComments))
				for i := range refCommentsFormatted {
					refCommentsFormatted[i] = refComments[i].PostFormat()
				}
				postFormatted.OrigContents = append(postFormatted.OrigContents, refCommentsFormatted...)
			case model.RefCommentReply:
				refReplies, err := s.getCommentRepliesByID(post.RefId)
				if err != nil {
					return nil, err
				}
				postFormatted.OrigContents = append(postFormatted.OrigContents, refReplies.PostFormat())
			}
		}
		// filter member content
		postsFormatted[i] = s.filterMemberContent(user, postFormatted)
	}
	return postsFormatted, nil
}

// RevampPosts post data shaping repair
func (s *tweetHelpServant) RevampPosts(user string, posts []*model.PostFormatted) ([]*model.PostFormatted, error) {
	postIds := make([]primitive.ObjectID, 0, len(posts))
	refItems := make(map[primitive.ObjectID]model.PostRefType, 0)
	addresses := make([]string, 0, len(posts))
	daoIds := make([]primitive.ObjectID, 0, len(posts))
	for _, post := range posts {
		if post.Type == model.Retweet || post.Type == model.RetweetComment {
			refItems[post.RefId] = post.RefType
		}
		if post.Type != model.DAO {
			postIds = append(postIds, post.ID)
		}
		addresses = append(addresses, post.Address, post.AuthorId)
		daoIds = append(daoIds, post.DaoId, post.AuthorDaoId)
	}

	postContents, err := s.getPostContentsByIDs(postIds)
	if err != nil {
		return nil, err
	}

	users, err := s.getUsersByAddress(addresses)
	if err != nil {
		return nil, err
	}

	daoS, err := s.getDAOsByIds(daoIds)
	if err != nil {
		return nil, err
	}
	joined := s.getDAOsByIdsWithJoined(user, daoIds)
	joinedMap := make(map[string]struct{}, len(joined))
	for _, v := range joined {
		joinedMap[v.DaoID.Hex()] = struct{}{}
	}
	subscribed := s.getDAOsByIdsWithSubscribed(user, daoIds)
	subscribedMap := make(map[string]struct{}, len(subscribed))
	for _, v := range subscribed {
		subscribedMap[v.DaoID.Hex()] = struct{}{}
	}
	userMap := make(map[string]*model.UserFormatted, len(users))
	for _, user := range users {
		userMap[user.Address] = user.Format()
	}

	daoMap := make(map[string]*model.DaoFormatted, len(daoS))
	for _, dao := range daoS {
		daoMap[dao.ID.Hex()] = dao.Format()
		if _, ok := joinedMap[dao.ID.Hex()]; ok {
			daoMap[dao.ID.Hex()].IsJoined = true
		}
		if _, ok := subscribedMap[dao.ID.Hex()]; ok {
			daoMap[dao.ID.Hex()].IsSubscribed = true
		}
	}

	contentMap := make(map[primitive.ObjectID][]*model.PostContentFormatted, len(postContents))
	for _, content := range postContents {
		contentMap[content.PostID] = append(contentMap[content.PostID], content.Format())
	}

	// data integration
	for _, post := range posts {
		post.User = userMap[post.Address]
		post.Dao = daoMap[post.DaoId.Hex()]

		if post.AuthorId != "" {
			post.Author = userMap[post.AuthorId]
		}

		if !post.AuthorDaoId.IsZero() {
			post.AuthorDao = daoMap[post.AuthorDaoId.Hex()]
		}

		if content, ok := contentMap[post.ID]; ok {
			post.Contents = content
		}

		if post.Type == model.Retweet || post.Type == model.RetweetComment {
			switch post.RefType {
			case model.RefPost:
				refContents, err := s.getPostContentsByID(post.RefId)
				if err != nil {
					return nil, err
				}
				refContentsFormatted := make([]*model.PostContentFormatted, len(refContents))
				for i := range refContentsFormatted {
					refContentsFormatted[i] = refContents[i].Format()
				}
				post.OrigContents = append(post.OrigContents, refContentsFormatted...)
			case model.RefComment:
				refComments, err := s.getCommentContentsByID(post.RefId)
				if err != nil {
					return nil, err
				}
				refCommentsFormatted := make([]*model.PostContentFormatted, len(refComments))
				for i := range refCommentsFormatted {
					refCommentsFormatted[i] = refComments[i].PostFormat()
				}
				post.OrigContents = append(post.OrigContents, refCommentsFormatted...)
			case model.RefCommentReply:
				refReplies, err := s.getCommentRepliesByID(post.RefId)
				if err != nil {
					return nil, err
				}
				post.OrigContents = append(post.OrigContents, refReplies.PostFormat())
			}
		}

		// filter member content
		post = s.filterMemberContent(user, post)
	}
	return posts, nil
}

func (s *tweetHelpServant) filterMemberContent(user string, post *model.PostFormatted) *model.PostFormatted {
	// Warning, Other related places Service.FilterMemberContent
	if post.Member == model.PostMemberNothing {
		return post
	}
	if post.Type == model.VIDEO {
		if user == "" || (user != post.Address && !post.Dao.IsSubscribed) {
			for k, v := range post.Contents {
				if v.Type == model.CONTENT_TYPE_VIDEO {
					post.Contents[k].Content = ""
				}
			}
		}
	} else if post.OrigType == model.VIDEO {
		if user == "" || (user != post.AuthorId && !post.AuthorDao.IsSubscribed) {
			for k, v := range post.OrigContents {
				if v.Type == model.CONTENT_TYPE_VIDEO {
					post.OrigContents[k].Content = ""
				}
			}
		}
	}
	return post
}

func (s *tweetHelpServant) getPostContentsByIDs(ids []primitive.ObjectID) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"query": bson.M{"post_id": bson.M{"$in": ids}},
		"ORDER": bson.M{"sort": 1},
	}, 0, 0)
}

func (s *tweetHelpServant) getPostContentsByID(id primitive.ObjectID) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"query": bson.M{"post_id": id},
		"ORDER": bson.M{"sort": 1},
	}, 0, 0)
}

func (s *tweetHelpServant) getCommentContentsByID(id primitive.ObjectID) ([]*model.CommentContent, error) {
	return (&model.CommentContent{}).List(s.db, &model.ConditionsT{
		"query": bson.M{"comment_id": id},
		"ORDER": bson.M{"sort": 1},
	}, 0, 0)
}

func (s *tweetHelpServant) getCommentRepliesByID(id primitive.ObjectID) (*model.CommentReply, error) {
	return (&model.CommentReply{ID: id}).Get(context.TODO(), s.db)
}

func (s *tweetHelpServant) getUsersByAddress(addresses []string) ([]*model.User, error) {
	user := &model.User{}
	return user.List(s.db, &model.ConditionsT{
		"query": bson.M{"address": bson.M{"$in": addresses}},
	}, 0, 0)
}

func (s *tweetHelpServant) getDAOsByIds(ids []primitive.ObjectID) ([]*model.Dao, error) {
	dao := &model.Dao{}
	return dao.List(s.db, &model.ConditionsT{
		"query": bson.M{"_id": bson.M{"$in": ids}},
	}, 0, 0)
}

func (s *tweetHelpServant) getDAOsByIdsWithJoined(address string, ids []primitive.ObjectID) []*model.DaoBookmark {
	book := &model.DaoBookmark{}
	return book.FindList(context.TODO(), s.db, bson.M{
		"address": address,
		"is_del":  0,
		"dao_id":  bson.M{"$in": ids},
	})
}

func (s *tweetHelpServant) getDAOsByIdsWithSubscribed(address string, ids []primitive.ObjectID) []*model.DaoSubscribe {
	book := &model.DaoSubscribe{}
	return book.FindList(context.TODO(), s.db, bson.M{
		"address": address,
		"status":  core.DaoSubscribeSuccess,
		"dao_id":  bson.M{"$in": ids},
	})
}

func (s *tweetManageServant) CreatePostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error) {
	collection := &model.PostCollection{
		PostID:  postID,
		Address: address,
	}

	return collection.Create(s.db)
}

func (s *tweetManageServant) DeletePostCollection(p *model.PostCollection) error {
	return p.Delete(s.db)
}

func (s *tweetManageServant) CreatePost(post *model.Post, contents []*model.PostContent) (*model.Post, error) {
	var newPost *model.Post

	err := util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		if !post.RefId.IsZero() {
			// check orig content exists
			switch post.RefType {
			case model.RefPost:
				// retweet post MUST exist
				origPost, err := (&model.Post{ID: post.RefId}).Get(ctx, s.db)
				if err != nil {
					return err
				}

				// update ref count
				origPost.RefCount++
				err = origPost.Update(ctx, s.db)
				if err != nil {
					return err
				}

				switch origPost.Type {
				case model.Retweet:
					// if re-post type is retweet, we'll find original post id
					post.RefId = origPost.RefId
					post.AuthorId = origPost.AuthorId
					post.AuthorDaoId = origPost.AuthorDaoId
				default:
					post.AuthorId = origPost.Address
					post.AuthorDaoId = origPost.DaoId
				}

				post.Tags = origPost.Tags
				post.OrigType = origPost.OrigType
			case model.RefComment:
				comment, err := (&model.Comment{ID: post.RefId}).Get(ctx, s.db)
				if err != nil {
					return err
				}

				post.AuthorId = comment.Address
			case model.RefCommentReply:
				reply, err := (&model.CommentReply{ID: post.RefId}).Get(ctx, s.db)
				if err != nil {
					return err
				}

				post.AuthorId = reply.Address
			}
		}

		p, err := post.Create(ctx, s.db)
		if err != nil {
			return err
		}

		for _, content := range contents {
			content.PostID = p.ID
			_, err = content.Create(ctx, s.db)
			if err != nil {
				return err
			}
		}

		newPost = p
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cacheIndex.SendAction(core.IdxActCreatePost, post)
	return newPost, nil
}

func (s *tweetManageServant) DeletePost(post *model.Post) ([]string, error) {
	var mediaContents []string

	postId := post.ID
	postContent := &model.PostContent{}
	session, err := s.db.Client().StartSession()
	if err != nil {
		return nil, err
	}
	wc := writeconcern.New(writeconcern.WMajority())
	txnOptions := options.Transaction().SetWriteConcern(wc)
	defer session.EndSession(context.TODO())
	_, err = session.WithTransaction(context.TODO(),
		func(ctx mongo.SessionContext) (interface{}, error) {
			if contents, err := postContent.MediaContentsByPostId(s.db, postId); err == nil {
				mediaContents = contents
			} else {
				return nil, err
			}

			// delete post
			if err := post.Delete(s.db); err != nil {
				return nil, err
			}

			if mediaContents != nil {
				// delete post content
				if err := postContent.DeleteByPostId(s.db, postId); err != nil {
					return nil, err
				}
			}

			if tags := strings.Split(post.Tags, ","); len(tags) > 0 {
				// Delete tag, handle errors loosely, no rollback with errors
				deleteTags(s.db, tags)
			}

			return nil, nil
		}, txnOptions)

	if err != nil {
		return nil, err
	}

	s.cacheIndex.SendAction(core.IdxActDeletePost, post)
	return mediaContents, nil
}

func (s *tweetManageServant) RealDeletePosts(address string, fn func(context.Context, *model.Post) (string, error)) error {
	return util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		cursor, err := s.db.Collection(new(model.Post).Table()).Find(ctx, bson.M{"address": address})
		for cursor.Next(ctx) {
			var t model.Post
			if cursor.Decode(&t) != nil {
				break
			}
			err = t.RealDelete(ctx, s.db)
			if err != nil {
				return err
			}
			_, err = fn(ctx, &t)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *tweetManageServant) StickPost(post *model.Post) error {
	post.IsTop = 1 - post.IsTop
	if err := post.Update(context.TODO(), s.db); err != nil {
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

	session, err := s.db.Client().StartSession()
	if err != nil {
		return err
	}
	wc := writeconcern.New(writeconcern.WMajority())
	txnOptions := options.Transaction().SetWriteConcern(wc)
	defer session.EndSession(context.TODO())

	_, err = session.WithTransaction(context.TODO(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		err := post.Update(sessCtx, s.db)
		if err != nil {
			return nil, err
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
				createTag(s.db, tag)
			} else if visibility == model.PostVisitPrivate {
				// You need to delete the tag only when you go from non-private to private
				deleteTag(s.db, tag)
			}
		}
		return nil, err
	}, txnOptions)
	if err != nil {
		return err
	}

	s.cacheIndex.SendAction(core.IdxActVisiblePost, post)
	return nil
}

func (s *tweetManageServant) UpdatePost(post *model.Post) error {
	if err := post.Update(context.TODO(), s.db); err != nil {
		return err
	}
	s.cacheIndex.SendAction(core.IdxActUpdatePost, post)
	return nil
}

func (s *tweetManageServant) CreatePostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	star := &model.PostStar{
		PostID:  postID,
		Address: address,
	}
	return star.Create(s.db)
}

func (s *tweetManageServant) DeletePostStar(p *model.PostStar) error {
	return p.Delete(s.db)
}

func (s *tweetServant) GetPostByID(id primitive.ObjectID) (*model.Post, error) {
	post := &model.Post{
		ID: id,
	}
	return post.Get(context.TODO(), s.db)
}

func (s *tweetServant) GetPosts(conditions *model.ConditionsT, offset, limit int) ([]*model.Post, error) {
	return (&model.Post{}).List(s.db, conditions, offset, limit)
}

func (s *tweetServant) GetPostCount(conditions *model.ConditionsT) (int64, error) {
	return (&model.Post{}).Count(s.db, conditions)
}

func (s *tweetServant) GetUserPostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	star := &model.PostStar{
		PostID:  postID,
		Address: address,
	}
	return star.Get(s.db)
}

func (s *tweetServant) GetUserPostStars(address string, offset, limit int) ([]*model.PostStarFormatted, error) {
	star := &model.PostStar{
		Address: address,
	}

	return star.List(s.db, &model.ConditionsT{
		"ORDER": bson.M{"_id": -1},
	}, offset, limit)
}

func (s *tweetServant) GetUserPostStarCount(address string) (int64, error) {
	star := &model.PostStar{
		Address: address,
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
	post, err = post.Get(context.TODO(), s.db)
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
		post, err = post.Get(context.TODO(), s.db)
		p.Post = post
	}

	return pc, err
}

func (s *tweetServant) GetUserPostCollectionCount(address string) (int64, error) {
	collection := &model.PostCollection{
		Address: address,
	}
	return collection.Count(s.db, &model.ConditionsT{})
}

func (s *tweetServant) GetPostContentsByIDs(ids []primitive.ObjectID) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"query": bson.M{"post_id": bson.M{"$in": ids}},
		"ORDER": bson.M{"sort": 1},
	}, 0, 0)
}

func (s *tweetServant) GetPostContentByID(id primitive.ObjectID) ([]*model.PostContent, error) {
	return (&model.PostContent{}).List(s.db, &model.ConditionsT{
		"query": bson.M{"post_id": id},
		"ORDER": bson.M{"sort": 1},
	}, 0, 0)
}

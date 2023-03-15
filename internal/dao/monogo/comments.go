package monogo

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
)

var (
	_ core.CommentService       = (*commentServant)(nil)
	_ core.CommentManageService = (*commentManageServant)(nil)
)

type commentServant struct {
	db *mongo.Database
}

type commentManageServant struct {
	db *mongo.Database
}

func newCommentService(db *mongo.Database) core.CommentService {
	return &commentServant{
		db: db,
	}
}

func newCommentManageService(db *mongo.Database) core.CommentManageService {
	return &commentManageServant{
		db: db,
	}
}

func (s *commentServant) GetComments(conditions *model.ConditionsT, offset, limit int) ([]*model.Comment, error) {
	return (&model.Comment{}).List(s.db, conditions, offset, limit)
}

func (s *commentServant) GetCommentByID(id primitive.ObjectID) (*model.Comment, error) {
	comment := &model.Comment{
		ID: id,
	}
	return comment.Get(context.TODO(), s.db)
}

func (s *commentServant) GetCommentReplyByID(id primitive.ObjectID) (*model.CommentReply, error) {
	reply := &model.CommentReply{
		ID: id,
	}
	return reply.Get(s.db)
}

func (s *commentServant) GetCommentCount(conditions *model.ConditionsT) (int64, error) {
	return (&model.Comment{}).Count(s.db, conditions)
}

func (s *commentServant) GetCommentContentsByIDs(ids []primitive.ObjectID) ([]*model.CommentContent, error) {
	commentContent := &model.CommentContent{}
	return commentContent.List(s.db, &model.ConditionsT{
		"query": bson.M{"comment_id": bson.M{"$in": ids}},
	}, 0, 0)
}

func (s *commentServant) GetCommentRepliesByID(ids []primitive.ObjectID) ([]*model.CommentReplyFormatted, error) {
	CommentReply := &model.CommentReply{}
	replies, err := CommentReply.List(s.db, &model.ConditionsT{
		"query": bson.M{"comment_id": bson.M{"$in": ids}},
	}, 0, 0)

	if err != nil {
		return nil, err
	}

	addresses := make([]string, len(replies))
	for i, reply := range replies {
		addresses[i] = reply.Address
	}

	var users []*model.User
	if len(addresses) != 0 {
		user := &model.User{}
		users, err = user.List(s.db, &model.ConditionsT{
			"query": bson.M{"address": bson.M{"$in": addresses}},
		}, 0, 0)
		if err != nil {
			return nil, err
		}
	}

	repliesFormatted := make([]*model.CommentReplyFormatted, len(replies))
	for i, reply := range replies {
		replyFormatted := reply.Format()
		for _, user := range users {
			if reply.Address == user.Address {
				replyFormatted.User = user.Format()
			}
		}

		repliesFormatted[i] = replyFormatted
	}

	return repliesFormatted, nil
}

func (s *commentManageServant) DeleteComment(comment *model.Comment) error {
	return comment.Delete(s.db)
}

func (s *commentManageServant) CreateComment(comment *model.Comment) (*model.Comment, error) {
	return comment.Create(s.db)
}

func (s *commentManageServant) CreateCommentReply(reply *model.CommentReply) (*model.CommentReply, error) {
	return reply.Create(s.db)
}

func (s *commentManageServant) DeleteCommentReply(reply *model.CommentReply) error {
	return reply.Delete(s.db)
}

func (s *commentManageServant) CreateCommentContent(content *model.CommentContent) (*model.CommentContent, error) {
	return content.Create(s.db)
}

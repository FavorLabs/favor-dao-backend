package core

import (
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CommentService 评论检索服务
type CommentService interface {
	GetComments(conditions *model.ConditionsT, offset, limit int) ([]*model.Comment, error)
	GetCommentByID(id primitive.ObjectID) (*model.Comment, error)
	GetCommentCount(conditions *model.ConditionsT) (int64, error)
	GetCommentReplyByID(id primitive.ObjectID) (*model.CommentReply, error)
	GetCommentContentsByIDs(ids []primitive.ObjectID) ([]*model.CommentContent, error)
	GetCommentRepliesByID(ids []primitive.ObjectID) ([]*model.CommentReplyFormatted, error)
}

// CommentManageService 评论管理服务
type CommentManageService interface {
	DeleteComment(comment *model.Comment) error
	CreateComment(comment *model.Comment) (*model.Comment, error)
	CreateCommentReply(reply *model.CommentReply) (*model.CommentReply, error)
	DeleteCommentReply(reply *model.CommentReply) error
	CreateCommentContent(content *model.CommentContent) (*model.CommentContent, error)
}

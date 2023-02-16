package core

import (
	"favor-dao-backend/internal/model"
)

type CommentService interface {
	GetComments(conditions *model.ConditionsT, offset, limit int) ([]*model.Comment, error)
	GetCommentByID(id int64) (*model.Comment, error)
	GetCommentCount(conditions *model.ConditionsT) (int64, error)
	GetCommentReplyByID(id int64) (*model.CommentReply, error)
	GetCommentContentsByIDs(ids []int64) ([]*model.CommentContent, error)
	GetCommentRepliesByID(ids []int64) ([]*model.CommentReplyFormated, error)
}

type CommentManageService interface {
	DeleteComment(comment *model.Comment) error
	CreateComment(comment *model.Comment) (*model.Comment, error)
	CreateCommentReply(reply *model.CommentReply) (*model.CommentReply, error)
	DeleteCommentReply(reply *model.CommentReply) error
	CreateCommentContent(content *model.CommentContent) (*model.CommentContent, error)
}

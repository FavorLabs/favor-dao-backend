package core

import "favor-dao-backend/internal/core/chat"

type DataService interface {
	TopicService
	IndexPostsService

	TweetService
	TweetManageService
	TweetHelpService

	CommentService
	CommentManageService

	UserManageService

	DaoManageService
	chat.ManageService
}

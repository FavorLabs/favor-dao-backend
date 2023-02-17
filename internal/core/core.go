package core

type DataService interface {
	TopicService
	IndexPostsService

	TweetService
	TweetManageService
	TweetHelpService

	AttachmentCheckService

	UserManageService
}

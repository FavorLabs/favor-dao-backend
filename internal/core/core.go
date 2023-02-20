package core

type DataService interface {
	TopicService
	IndexPostsService

	TweetService
	TweetManageService
	TweetHelpService

	UserManageService

	DaoManageService
}

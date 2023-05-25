package errcode

var (
	NoPermission         = NewError(20007, "No Permission")
	ErrorCaptchaPassword = NewError(20012, "Error Captcha Password")
	TooManyLoginError    = NewError(20014, "Too Many Login Error，Please try again later")
	NicknameLengthLimit  = NewError(20020, "Nickname length 2~12")
	NoExistUserAddress   = NewError(20021, "No Exist User Address")

	CreatePostFailed  = NewError(30002, "Create Post Failed")
	GetPostFailed     = NewError(30003, "Get Post Failed")
	DeletePostFailed  = NewError(30004, "Delete Post Failed")
	LockPostFailed    = NewError(30005, "Lock Post Failed")
	GetPostTagsFailed = NewError(30006, "Get Post Tags Failed")
	VisiblePostFailed = NewError(30012, "Visible Post Failed")

	GetCommentsFailed   = NewError(40001, "Get Comments Failed")
	CreateCommentFailed = NewError(40002, "Create Comment Failed")
	GetCommentFailed    = NewError(40003, "Get Comment Failed")
	DeleteCommentFailed = NewError(40004, "Delete Comment Failed")
	CreateReplyFailed   = NewError(40005, "Create Reply Failed")
	GetReplyFailed      = NewError(40006, "Get Reply Failed")
	MaxCommentCount     = NewError(40007, "Max Comment Count")

	RedpacketHasBeenCollectedCompletely = NewError(50001, "It has been collected completely")
	RedpacketAlreadyClaim               = NewError(50002, "Already claim")

	GetCollectionsFailed = NewError(60001, "Get Collections Failed")

	CreateDaoFailed          = NewError(80001, "Create Dao Failed")
	GetDaoFailed             = NewError(80002, "Get Dao Failed")
	UpdateDaoFailed          = NewError(80003, "Update Dao Failed")
	CreateDaoNameDuplication = NewError(80004, "DAO name duplication")
	NoExistDao               = NewError(80005, "DAO not found")
	SubscribeDAO             = NewError(80006, "Subscribe DAO Failed")
	DAONothingChange         = NewError(80007, "DAO Nothing Change")
	AlreadySubscribedDAO     = NewError(80008, "Already Subscribed DAO")
	CreateChatGroupFailed    = NewError(80009, "Create Chat Group Failed")

	PayNotifyError = NewError(90001, "Pay notify Failed")
)

package errcode

var (
	Success                   = NewError(0, "Success")
	ServerError               = NewError(10000, "Server Error")
	InvalidParams             = NewError(10001, "Invalid Params")
	NotFound                  = NewError(10002, "Not Found")
	UnauthorizedAuthNotExist  = NewError(10003, "Unauthorized Auth Not Exist")
	UnauthorizedAuthFailed    = NewError(10004, "Unauthorized Auth Failed")
	UnauthorizedTokenError    = NewError(10005, "Unauthorized Token Error")
	UnauthorizedTokenTimeout  = NewError(10006, "Unauthorized Token Timeout")
	UnauthorizedTokenGenerate = NewError(10007, "Unauthorized Token Generate")
	TooManyRequests           = NewError(10008, "Too Many Requests")
)

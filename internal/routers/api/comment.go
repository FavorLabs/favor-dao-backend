package api

import (
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetPostComments(c *gin.Context) {
	id := c.Query("id")
	response := app.NewResponse(c)

	postId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("service.GetPostComments err: %v\n", err)
		response.ToErrorResponse(errcode.GetCommentsFailed)
		return
	}

	offset, limit := app.GetPageOffset(c)
	contents, totalRows, err := service.GetPostComments(postId, "_id", 1, offset, limit)

	if err != nil {
		logrus.Errorf("service.GetPostComments err: %v\n", err)
		response.ToErrorResponse(errcode.GetCommentsFailed)
		return
	}

	response.ToResponseList(contents, totalRows)
}

func CreatePostComment(c *gin.Context) {
	param := service.CommentCreationReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	address, _ := c.Get("address")
	comment, err := service.CreatePostComment(address.(string), param)

	if err != nil {
		if err == errcode.MaxCommentCount {
			response.ToErrorResponse(errcode.MaxCommentCount)
		} else {
			logrus.Errorf("service.CreatePostComment err: %v\n", err)
			response.ToErrorResponse(errcode.CreateCommentFailed)
		}
		return
	}

	response.ToResponse(comment)
}

func DeletePostComment(c *gin.Context) {
	param := service.CommentDelReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	user, _ := c.Get("USER")

	comment, err := service.GetPostComment(param.ID)
	if err != nil {
		logrus.Errorf("service.GetPostComment err: %v\n", err)
		response.ToErrorResponse(errcode.GetCommentFailed)
		return
	}

	if user.(*model.User).Address != comment.Address {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	// 执行删除
	err = service.DeletePostComment(comment)
	if err != nil {
		logrus.Errorf("service.DeletePostComment err: %v\n", err)
		response.ToErrorResponse(errcode.DeleteCommentFailed)
		return
	}

	response.ToResponse(nil)
}

func CreatePostCommentReply(c *gin.Context) {
	param := service.CommentReplyCreationReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}
	user, _ := c.Get("USER")

	comment, err := service.CreatePostCommentReply(param.CommentID, param.Content, user.(*model.User).Address)
	if err != nil {
		logrus.Errorf("service.CreatePostCommentReply err: %v\n", err)
		response.ToErrorResponse(errcode.CreateReplyFailed)
		return
	}

	response.ToResponse(comment)
}

func DeletePostCommentReply(c *gin.Context) {
	param := service.ReplyDelReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	user, _ := c.Get("USER")

	reply, err := service.GetPostCommentReply(param.ID)
	if err != nil {
		logrus.Errorf("service.GetPostCommentReply err: %v\n", err)
		response.ToErrorResponse(errcode.GetReplyFailed)
		return
	}

	if user.(*model.User).Address != reply.Address {
		response.ToErrorResponse(errcode.NoPermission)
		return
	}

	// 执行删除
	err = service.DeletePostCommentReply(reply)
	if err != nil {
		logrus.Errorf("service.DeletePostCommentReply err: %v\n", err)
		response.ToErrorResponse(errcode.DeleteCommentFailed)
		return
	}

	response.ToResponse(nil)
}

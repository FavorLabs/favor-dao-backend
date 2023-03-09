package service

import (
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/errcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CommentCreationReq struct {
	PostID   primitive.ObjectID `json:"post_id" binding:"required"`
	Contents []*PostContentItem `json:"contents" binding:"required"`
}

type CommentReplyCreationReq struct {
	CommentID primitive.ObjectID `json:"comment_id" binding:"required"`
	Content   string             `json:"content" binding:"required"`
}

type CommentDelReq struct {
	ID primitive.ObjectID `json:"id" binding:"required"`
}
type ReplyDelReq struct {
	ID primitive.ObjectID `json:"id" binding:"required"`
}

func GetPostComments(postID primitive.ObjectID, sort string, sortVal, offset, limit int) ([]*model.CommentFormatted, int64, error) {
	conditions := &model.ConditionsT{
		"query": bson.M{"post_id": postID},
		"ORDER": bson.M{sort: sortVal},
	}
	comments, err := ds.GetComments(conditions, offset, limit)

	if err != nil {
		return nil, 0, err
	}

	var addresses []string
	var commentIDs []primitive.ObjectID
	for _, comment := range comments {
		addresses = append(addresses, comment.Address)
		commentIDs = append(commentIDs, comment.ID)
	}

	users, err := ds.GetUsersByAddresses(addresses)
	if err != nil {
		return nil, 0, err
	}

	contents, err := ds.GetCommentContentsByIDs(commentIDs)
	if err != nil {
		return nil, 0, err
	}

	replies, err := ds.GetCommentRepliesByID(commentIDs)
	if err != nil {
		return nil, 0, err
	}

	var commentsFormatted []*model.CommentFormatted
	for _, comment := range comments {
		commentFormatted := comment.Format()
		for _, content := range contents {
			if content.CommentID == comment.ID {
				commentFormatted.Contents = append(commentFormatted.Contents, content)
			}
		}
		for _, reply := range replies {
			if reply.CommentID == commentFormatted.ID {
				commentFormatted.Replies = append(commentFormatted.Replies, reply)
			}
		}
		for _, user := range users {
			if user.Address == comment.Address {
				commentFormatted.User = user.Format()
			}
		}

		commentsFormatted = append(commentsFormatted, commentFormatted)
	}

	// 获取总量
	totalRows, _ := ds.GetCommentCount(conditions)

	return commentsFormatted, totalRows, nil
}

func CreatePostComment(address string, param CommentCreationReq) (comment *model.Comment, err error) {
	var mediaContents []string

	defer func() {
		if err != nil {
			deleteOssObjects(mediaContents)
		}
	}()

	if mediaContents, err = persistMediaContents(param.Contents); err != nil {
		return
	}

	// 加载Post
	post, err := ds.GetPostByID(param.PostID)
	if err != nil {
		return nil, err
	}

	if post.CommentCount >= conf.AppSetting.MaxCommentCount {
		return nil, errcode.MaxCommentCount
	}

	comment = &model.Comment{
		PostID:  post.ID,
		Address: address,
	}
	comment, err = ds.CreateComment(comment)
	if err != nil {
		return nil, err
	}

	for _, item := range param.Contents {
		postContent := &model.CommentContent{
			CommentID: comment.ID,
			Address:   address,
			Content:   item.Content,
			Type:      item.Type,
			Sort:      item.Sort,
		}
		_, err = ds.CreateCommentContent(postContent)
		if err != nil {
			return nil, err
		}
	}

	// 更新Post回复数
	post.CommentCount++
	post.LatestRepliedOn = time.Now().Unix()
	if err := ds.UpdatePost(post); err != nil {
		return nil, err
	}

	// 更新索引
	PushPostToSearch(post)

	return comment, nil
}

func GetPostComment(id primitive.ObjectID) (*model.Comment, error) {
	return ds.GetCommentByID(id)
}

func DeletePostComment(comment *model.Comment) error {
	// 加载post
	post, err := ds.GetPostByID(comment.PostID)
	if err == nil {
		// 更新post回复数
		post.CommentCount--
		if err := ds.UpdatePost(post); err != nil {
			return err
		}
	}

	return ds.DeleteComment(comment)
}

func createPostPreHandler(commentID primitive.ObjectID) (*model.Post, error) {
	// 加载Comment
	comment, err := ds.GetCommentByID(commentID)
	if err != nil {
		return nil, err
	}

	// 加载comment的post
	post, err := ds.GetPostByID(comment.PostID)
	if err != nil {
		return nil, err
	}

	if post.CommentCount >= conf.AppSetting.MaxCommentCount {
		return nil, errcode.MaxCommentCount
	}

	return post, nil
}

func CreatePostCommentReply(commentID primitive.ObjectID, content string, address string) (*model.CommentReply, error) {
	var (
		post *model.Post
		err  error
	)
	if post, err = createPostPreHandler(commentID); err != nil {
		return nil, err
	}

	// 创建评论
	reply := &model.CommentReply{
		CommentID: commentID,
		Address:   address,
		Content:   content,
	}

	reply, err = ds.CreateCommentReply(reply)
	if err != nil {
		return nil, err
	}

	// 更新Post回复数
	post.CommentCount++
	post.LatestRepliedOn = time.Now().Unix()
	if err := ds.UpdatePost(post); err != nil {
		return nil, err
	}

	// 更新索引
	PushPostToSearch(post)

	return reply, nil
}

func GetPostCommentReply(id primitive.ObjectID) (*model.CommentReply, error) {
	return ds.GetCommentReplyByID(id)
}

func DeletePostCommentReply(reply *model.CommentReply) error {
	err := ds.DeleteCommentReply(reply)
	if err != nil {
		return err
	}

	// 加载Comment
	comment, err := ds.GetCommentByID(reply.CommentID)
	if err != nil {
		return err
	}

	// 加载comment的post
	post, err := ds.GetPostByID(comment.PostID)
	if err != nil {
		return err
	}

	// 更新Post回复数
	post.CommentCount--
	post.LatestRepliedOn = time.Now().Unix()
	if err := ds.UpdatePost(post); err != nil {
		return err
	}

	// 更新索引
	PushPostToSearch(post)

	return nil
}

package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TagType string

const TagTypeHot TagType = "hot"
const TagTypeNew TagType = "new"

type PostListReq struct {
	Conditions *model.ConditionsT
	Offset     int
	Limit      int
}
type PostTagsReq struct {
	Type TagType `json:"type" form:"type" binding:"required"`
	Num  int     `json:"num" form:"num" binding:"required"`
}

type PostCreationReq struct {
	Contents   []*PostContentItem `json:"contents"`
	DaoId      primitive.ObjectID `json:"dao_id" binding:"required"`
	Tags       []string           `json:"tags"`
	Type       model.PostType     `json:"type"`
	RefType    model.PostRefType  `json:"ref_type"`
	RefId      primitive.ObjectID `json:"ref_id"`
	Visibility model.PostVisibleT `json:"visibility"`
}

type PostDelReq struct {
	ID string `json:"id" binding:"required"`
}

type PostLockReq struct {
	ID string `json:"id" binding:"required"`
}

type PostStickReq struct {
	ID string `json:"id" binding:"required"`
}

type PostVisibilityReq struct {
	ID         string             `json:"id" binding:"required"`
	Visibility model.PostVisibleT `json:"visibility"`
}

type PostStarReq struct {
	ID string `json:"id" binding:"required"`
}

type PostCollectionReq struct {
	ID string `json:"id" binding:"required"`
}

type PostContentItem struct {
	Content string             `json:"content"  binding:"required"`
	Type    model.PostContentT `json:"type"  binding:"required"`
	Sort    int64              `json:"sort"  binding:"required"`
}

// Check 检查PostContentItem属性
func (p *PostContentItem) Check() error {
	// 检查链接是否合法
	if p.Type == model.CONTENT_TYPE_LINK {
		if strings.Index(p.Content, "http://") != 0 && strings.Index(p.Content, "https://") != 0 {
			return fmt.Errorf("链接不合法")
		}
	}

	return nil
}

func tagsFrom(originTags []string) []string {
	tags := make([]string, 0, len(originTags))
	for _, tag := range originTags {
		// TODO: 优化tag有效性检测
		if tag = strings.TrimSpace(tag); len(tag) > 0 {
			tags = append(tags, tag)
		}
	}
	return tags
}

// CreatePost 创建文章
// TODO: 推文+推文内容需要在一个事务中添加，后续优化
func CreatePost(c *gin.Context, address string, param PostCreationReq) (_ *model.PostFormatted, err error) {
	var (
		post          *model.Post
		mediaContents []string
	)

	defer func() {
		if err != nil {
			deleteOssObjects(mediaContents)
		}
	}()

	if mediaContents, err = persistMediaContents(param.Contents); err != nil {
		return
	}

	switch param.Type {
	case model.Retweet, model.RetweetComment:
		var tags string

		// check orig content exists
		switch param.RefType {
		case model.RefPost:
			// retweet post MUST exist
			origPost, err := ds.GetPostByID(param.RefId)
			if err != nil {
				return nil, fmt.Errorf("need find post to retweet: %v", err)
			}

			// if re-post type is retweet, we'll find original post id
			if origPost.Type == model.Retweet {
				param.RefId = origPost.RefId
			}

			tags = origPost.Tags
		case model.RefComment:
			_, err = ds.GetCommentByID(param.RefId)
		case model.RefCommentReply:
			_, err = ds.GetCommentReplyByID(param.RefId)
		}
		if err != nil {
			return nil, fmt.Errorf("")
		}

		// create post ref
		post, err = ds.CreatePost(&model.Post{
			Address:    address,
			DaoId:      param.DaoId,
			Tags:       tags,
			Visibility: param.Visibility,
			Type:       param.Type,
			RefId:      param.RefId,
			RefType:    param.RefType,
		})
		if err != nil {
			return nil, err
		}

		if param.Type == model.RetweetComment {
			if len(param.Contents) < 1 {
				return nil, fmt.Errorf("empty post content in RetweetComment")
			}
			for _, item := range param.Contents {
				if err := item.Check(); err != nil {
					// 属性非法
					logrus.Infof("contents check err: %v", err)
					continue
				}

				postContent := &model.PostContent{
					PostID:  post.ID,
					Address: address,
					Content: item.Content,
					Type:    item.Type,
					Sort:    item.Sort,
				}
				if _, err = ds.CreatePostContent(postContent); err != nil {
					return nil, err
				}
			}
		}
	default:
		tags := tagsFrom(param.Tags)
		post, err = ds.CreatePost(&model.Post{
			Address:    address,
			DaoId:      param.DaoId,
			Tags:       strings.Join(tags, ","),
			Visibility: param.Visibility,
			Type:       param.Type,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range param.Contents {
			if err := item.Check(); err != nil {
				// 属性非法
				logrus.Infof("contents check err: %v", err)
				continue
			}

			postContent := &model.PostContent{
				PostID:  post.ID,
				Address: address,
				Content: item.Content,
				Type:    item.Type,
				Sort:    item.Sort,
			}
			if _, err = ds.CreatePostContent(postContent); err != nil {
				return nil, err
			}
		}

		if post.Visibility == model.PostVisitPublic {
			for _, t := range tags {
				tag := &model.Tag{
					Address: address,
					Tag:     t,
				}
				ds.CreateTag(tag)
			}

		}
	}

	PushPostToSearch(post)

	formattedPosts, err := ds.RevampPosts([]*model.PostFormatted{post.Format()})
	if err != nil {
		return nil, err
	}
	return formattedPosts[0], nil
}

func DeletePost(user *model.User, id primitive.ObjectID) *errcode.Error {
	if user == nil {
		return errcode.NoPermission
	}

	post, err := ds.GetPostByID(id)
	if err != nil {
		return errcode.GetPostFailed
	}
	if post.Address != user.Address {
		return errcode.NoPermission
	}

	mediaContents, err := ds.DeletePost(post)
	if err != nil {
		logrus.Errorf("service.DeletePost delete post failed: %s", err)
		return errcode.DeletePostFailed
	}

	// 删除推文的媒体内容
	deleteOssObjects(mediaContents)

	// 删除索引
	DeleteSearchPost(post)

	return nil
}

func deleteOssObjects(mediaContents []string) {
	mediaContentsSize := len(mediaContents)
	if mediaContentsSize > 1 {
		// todo
	} else if mediaContentsSize == 1 {
		// todo
	}
}

func StickPost(id primitive.ObjectID) error {
	post, _ := ds.GetPostByID(id)

	err := ds.StickPost(post)

	if err != nil {
		return err
	}

	return nil
}

func VisiblePost(user *model.User, postId primitive.ObjectID, visibility model.PostVisibleT) *errcode.Error {

	post, err := ds.GetPostByID(postId)
	if err != nil {
		return errcode.GetPostFailed
	}

	if err = ds.VisiblePost(post, visibility); err != nil {
		logrus.Warnf("update post failure: %v", err)
		return errcode.VisblePostFailed
	}

	// 推送Search
	post.Visibility = visibility
	PushPostToSearch(post)

	return nil
}

func GetPostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	return ds.GetUserPostStar(postID, address)
}

func CreatePostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	// 加载Post
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	// 私密post不可操作
	if post.Visibility == model.PostVisitPrivate {
		return nil, errors.New("no permision")
	}

	star, err := ds.CreatePostStar(postID, address)
	if err != nil {
		return nil, err
	}

	// 更新Post点赞数
	post.UpvoteCount++
	ds.UpdatePost(post)

	// 更新索引
	PushPostToSearch(post)

	return star, nil
}

func DeletePostStar(star *model.PostStar) error {
	err := ds.DeletePostStar(star)
	if err != nil {
		return err
	}
	// 加载Post
	post, err := ds.GetPostByID(star.PostID)
	if err != nil {
		return err
	}

	// 私密post不可操作
	if post.Visibility == model.PostVisitPrivate {
		return errors.New("no permision")
	}

	// 更新Post点赞数
	post.UpvoteCount--
	ds.UpdatePost(post)

	// 更新索引
	PushPostToSearch(post)

	return nil
}

func CreatePostView(postID primitive.ObjectID) error {
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return err
	}

	// 更新Post观看数
	post.ViewCount++
	ds.UpdatePost(post)

	// 更新索引
	PushPostToSearch(post)

	return nil
}

func GetPostView(postID primitive.ObjectID) (int64, error) {
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return 0, err
	}
	return post.ViewCount, err
}

func GetPostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error) {
	return ds.GetUserPostCollection(postID, address)
}

func CreatePostCollection(postID primitive.ObjectID, address string) (*model.PostCollection, error) {
	// 加载Post
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	// 私密post不可操作
	if post.Visibility == model.PostVisitPrivate {
		return nil, errors.New("no permision")
	}

	collection, err := ds.CreatePostCollection(postID, address)
	if err != nil {
		return nil, err
	}

	// 更新Post点赞数
	post.CollectionCount++
	ds.UpdatePost(post)

	// 更新索引
	PushPostToSearch(post)

	return collection, nil
}

func DeletePostCollection(collection *model.PostCollection) error {
	err := ds.DeletePostCollection(collection)
	if err != nil {
		return err
	}
	// 加载Post
	post, err := ds.GetPostByID(collection.PostID)
	if err != nil {
		return err
	}

	// 私密post不可操作
	if post.Visibility == model.PostVisitPrivate {
		return errors.New("no permision")
	}

	// 更新Post点赞数
	post.CollectionCount--
	ds.UpdatePost(post)

	// 更新索引
	PushPostToSearch(post)

	return nil
}

func GetPost(id primitive.ObjectID) (*model.PostFormatted, error) {
	post, err := ds.GetPostByID(id)

	if err != nil {
		return nil, err
	}

	postContents, err := ds.GetPostContentsByIDs([]primitive.ObjectID{post.ID})
	if err != nil {
		return nil, err
	}

	users, err := ds.GetUsersByAddresses([]string{post.Address})
	if err != nil {
		return nil, err
	}
	dao, err := ds.GetDao(&model.Dao{ID: post.DaoId})
	if err != nil {
		return nil, err
	}

	postFormatted := post.Format()
	for _, user := range users {
		postFormatted.User = user.Format()
	}
	postFormatted.Dao = dao.Format()

	for _, content := range postContents {
		if content.PostID == post.ID {
			postFormatted.Contents = append(postFormatted.Contents, content.Format())
		}
	}
	return postFormatted, nil
}

func GetPostContentByID(id primitive.ObjectID) ([]*model.PostContent, error) {
	return ds.GetPostContentByID(id)
}

func GetIndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error) {
	return ds.IndexPosts(user, offset, limit)
}

func GetPostList(req *PostListReq) ([]*model.PostFormatted, error) {
	posts, err := ds.GetPosts(req.Conditions, req.Offset, req.Limit)

	if err != nil {
		return nil, err
	}

	return ds.MergePosts(posts)
}

func GetPostCount(conditions *model.ConditionsT) (int64, error) {
	return ds.GetPostCount(conditions)
}

func GetPostListFromSearch(user *model.User, q *core.QueryReq, offset, limit int) ([]*model.PostFormatted, int64, error) {
	resp, err := ts.Search(user, q, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	posts, err := ds.RevampPosts(resp.Items)
	if err != nil {
		return nil, 0, err
	}
	return posts, resp.Total, nil
}

func GetPostListFromSearchByQuery(user *model.User, query string, offset, limit int) ([]*model.PostFormatted, int64, error) {
	q := &core.QueryReq{
		Query: query,
		Type:  "search",
	}
	return GetPostListFromSearch(user, q, offset, limit)
}

func PushPostToSearch(post *model.Post) {
	postFormatted := post.Format()
	postFormatted.User = &model.UserFormatted{
		ID: post.Address,
	}
	contents, _ := ds.GetPostContentsByIDs([]primitive.ObjectID{post.ID})
	for _, content := range contents {
		postFormatted.Contents = append(postFormatted.Contents, content.Format())
	}

	contentFormatted := ""

	for _, content := range postFormatted.Contents {
		if content.Type == model.CONTENT_TYPE_TEXT || content.Type == model.CONTENT_TYPE_TITLE {
			contentFormatted = contentFormatted + content.Content + "\n"
		}
	}

	tagMaps := map[string]int8{}
	for _, tag := range strings.Split(post.Tags, ",") {
		tagMaps[tag] = 1
	}

	data := core.DocItems{{
		"id":                post.ID,
		"address":           post.Address,
		"dao_id":            post.DaoId.Hex(),
		"view_count":        post.ViewCount,
		"collection_count":  post.CollectionCount,
		"upvote_count":      post.UpvoteCount,
		"comment_count":     post.CommentCount,
		"member":            post.Member,
		"visibility":        post.Visibility,
		"is_top":            post.IsTop,
		"is_essence":        post.IsEssence,
		"content":           contentFormatted,
		"tags":              tagMaps,
		"type":              post.Type,
		"created_on":        post.CreatedOn,
		"modified_on":       post.ModifiedOn,
		"latest_replied_on": post.LatestRepliedOn,
	}}

	ts.AddDocuments(data, post.ID.Hex())
}

func DeleteSearchPost(post *model.Post) error {
	return ts.DeleteDocuments([]string{post.ID.Hex()})
}

func PushPostsToSearch() {
	splitNum := 1000
	totalRows, _ := GetPostCount(&model.ConditionsT{
		"query": bson.M{"visibility": model.PostVisitPublic},
	})

	pages := math.Ceil(float64(totalRows) / float64(splitNum))
	nums := int(pages)

	for i := 0; i < nums; i++ {
		posts, _ := GetPostList(&PostListReq{
			Conditions: &model.ConditionsT{},
			Offset:     i * splitNum,
			Limit:      splitNum,
		})

		for _, post := range posts {
			contentFormatted := ""

			for _, content := range post.Contents {
				if content.Type == model.CONTENT_TYPE_TEXT || content.Type == model.CONTENT_TYPE_TITLE {
					contentFormatted = contentFormatted + content.Content + "\n"
				}
			}

			docs := core.DocItems{{
				"id":                post.ID,
				"address":           post.Address,
				"dao_id":            post.DaoId.Hex(),
				"view_count":        post.ViewCount,
				"collection_count":  post.CollectionCount,
				"upvote_count":      post.UpvoteCount,
				"comment_count":     post.CommentCount,
				"member":            post.Member,
				"visibility":        post.Visibility,
				"is_top":            post.IsTop,
				"is_essence":        post.IsEssence,
				"content":           contentFormatted,
				"tags":              post.Tags,
				"type":              post.Type,
				"created_on":        post.CreatedOn,
				"modified_on":       post.ModifiedOn,
				"latest_replied_on": post.LatestRepliedOn,
			}}
			_, err := ts.AddDocuments(docs, post.ID.Hex())
			if err != nil {
				log.Printf("add document err: %s\n", err)
				continue
			}
			log.Printf("add document success, post_id: %s\n", post.ID.Hex())
		}
	}
}

func GetPostTags(param *PostTagsReq) ([]*model.TagFormatted, error) {
	num := param.Num
	if num > conf.AppSetting.MaxPageSize {
		num = conf.AppSetting.MaxPageSize
	}

	conditions := &model.ConditionsT{}
	if param.Type == TagTypeHot {
		// 热门标签
		conditions = &model.ConditionsT{
			"ORDER": bson.M{"quote_num": -1},
		}
	}
	if param.Type == TagTypeNew {
		// 热门标签
		conditions = &model.ConditionsT{
			"ORDER": bson.M{"id": -1},
		}
	}

	tags, err := ds.GetTags(conditions, 0, num)
	if err != nil {
		return nil, err
	}

	// 获取创建者User IDs
	userIds := []string{}
	for _, tag := range tags {
		userIds = append(userIds, tag.Address)
	}

	users, _ := ds.GetUsersByAddresses(userIds)

	var tagsFormated []*model.TagFormatted
	for _, tag := range tags {
		tagFormated := tag.Format()
		for _, user := range users {
			if user.Address == tagFormated.Address {
				tagFormated.User = user.Format()
			}
		}
		tagsFormated = append(tagsFormated, tagFormated)
	}

	return tagsFormated, nil
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/notify"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	Member     model.PostMemberT  `json:"member"`
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

func (p *PostContentItem) Check() error {
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
		if tag = strings.TrimSpace(tag); len(tag) > 0 {
			tags = append(tags, tag)
		}
	}
	return tags
}

func CreatePost(user *model.User, param PostCreationReq) (_ *model.PostFormatted, err error) {
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

	var contents []*model.PostContent
	for _, item := range param.Contents {
		if err := item.Check(); err != nil {
			logrus.Infof("contents check err: %v", err)
			continue
		}

		contents = append(contents, &model.PostContent{
			Address: user.Address,
			Content: item.Content,
			Type:    item.Type,
			Sort:    item.Sort,
		})
	}

	// HACKING!
	if param.Type > 1 {
		param.Type = 0
	}

	// check address has alwaysTop feature
	alwaysTop := 0
	for _, addr := range conf.ExternalAppSetting.AlwaysTopAddresses {
		if user.Address == addr {
			alwaysTop = 1
		}
	}

	// put unpin job
	if alwaysTop == 1 {
		defer func() {
			if err == nil {
				task := NewPostUnpinTask(post.ID)
				_, taskErr := queue.Enqueue(task, asynq.ProcessIn(time.Hour*24), asynq.Queue(PostQueue))
				if taskErr != nil {
					logrus.Errorf("unpin post %s enqueue failed: %v", post.ID, taskErr)
				}
			}
		}()
	}

	if !param.RefId.IsZero() {
		// create post ref
		post, err = ds.CreatePost(&model.Post{
			Address:    user.Address,
			DaoId:      param.DaoId,
			Visibility: param.Visibility,
			Type:       param.Type,
			RefId:      param.RefId,
			RefType:    param.RefType,
			Member:     param.Member,
			IsTop:      alwaysTop,
		}, contents)
		if err != nil {
			return nil, err
		}
	} else {
		tags := tagsFrom(param.Tags)
		post, err = ds.CreatePost(&model.Post{
			Address:    user.Address,
			DaoId:      param.DaoId,
			Tags:       strings.Join(tags, ","),
			Visibility: param.Visibility,
			Type:       param.Type,
			OrigType:   param.Type,
			Member:     param.Member,
			IsTop:      alwaysTop,
		}, contents)
		if err != nil {
			return nil, err
		}

		if post.Visibility == model.PostVisitPublic {
			for _, t := range tags {
				tag := &model.Tag{
					Address: user.Address,
					Tag:     t,
				}
				ds.CreateTag(tag)
			}

		}
	}

	PushPostToSearch(post)

	formattedPosts, err := ds.RevampPosts(user.Address, []*model.PostFormatted{post.Format()})
	if err != nil {
		return nil, err
	}
	linkMap := make(map[string]any)
	if post.Type == model.SMS {
		linkMap["route"] = "PostDetail"
	} else {
		linkMap["route"] = "VideoPlay"
	}
	dao := &model.Dao{ID: param.DaoId}
	d, _ := ds.GetDao(dao)

	content := fmt.Sprintf("The %s Dao you subscribe to has posted new content", d.Name)
	linkMap["id"] = post.ID
	links, _ := json.Marshal(&linkMap)
	nr := notify.PushNotifyRequest{
		IsSave:    true,
		NetWorkId: conf.ExternalAppSetting.NetworkID,
		Region:    conf.ExternalAppSetting.Region,
		Title:     "",
		Content:   content,
		Links:     string(links),
		From:      param.DaoId.Hex(),
		FromType:  model.DAO_TYPE,
		To:        param.DaoId.Hex(),
	}
	err = notifyGateway.NotifyDao(context.TODO(), nr)
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

	mediaContents, refPosts, err := ds.DeletePost(post)
	if err != nil {
		logrus.Errorf("service.DeletePost delete post failed: %s", err)
		return errcode.DeletePostFailed
	}

	deleteOssObjects(mediaContents)

	DeleteSearchPost(post, refPosts...)

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
	e := CheckIsMyDAO(user.Address, post.DaoId)
	if e != nil {
		return e
	}
	if err = ds.VisiblePost(post, visibility); err != nil {
		logrus.Warnf("update post failure: %v", err)
		return errcode.VisiblePostFailed
	}

	post.Visibility = visibility
	PushPostToSearch(post)

	return nil
}

func GetPostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	return ds.GetUserPostStar(postID, address)
}

func CreatePostStar(postID primitive.ObjectID, address string) (*model.PostStar, error) {
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	star, err := ds.CreatePostStar(postID, address)
	if err != nil {
		return nil, err
	}

	post.UpvoteCount++
	ds.UpdatePost(post)

	PushPostToSearch(post)

	return star, nil
}

func DeletePostStar(star *model.PostStar) error {
	err := ds.DeletePostStar(star)
	if err != nil {
		return err
	}
	post, err := ds.GetPostByID(star.PostID)
	if err != nil {
		return err
	}

	post.UpvoteCount--
	ds.UpdatePost(post)

	PushPostToSearch(post)

	return nil
}

func CreatePostView(postID primitive.ObjectID) error {
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return err
	}

	post.ViewCount++
	ds.UpdatePost(post)

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
	post, err := ds.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	if post.Visibility == model.PostVisitPrivate {
		return nil, errors.New("no permision")
	}

	collection, err := ds.CreatePostCollection(postID, address)
	if err != nil {
		return nil, err
	}

	post.CollectionCount++
	ds.UpdatePost(post)

	PushPostToSearch(post)

	return collection, nil
}

func DeletePostCollection(collection *model.PostCollection) error {
	err := ds.DeletePostCollection(collection)
	if err != nil {
		return err
	}
	post, err := ds.GetPostByID(collection.PostID)
	if err != nil {
		return err
	}

	if post.Visibility == model.PostVisitPrivate {
		return errors.New("no permision")
	}

	post.CollectionCount--
	ds.UpdatePost(post)

	PushPostToSearch(post)

	return nil
}

func GetPost(user string, id primitive.ObjectID) (*model.PostFormatted, error) {
	post, err := ds.GetPostByID(id)
	if err != nil {
		return nil, err
	}

	postFormatted := post.Format()

	postContents, err := ds.GetPostContentsByIDs([]primitive.ObjectID{post.ID})
	if err != nil {
		return nil, err
	}
	for _, content := range postContents {
		if content.PostID == post.ID {
			postFormatted.Contents = append(postFormatted.Contents, content.Format())
		}
	}

	if !post.RefId.IsZero() {
		switch postFormatted.RefType {
		case model.RefPost:
			refContents, err := ds.GetPostContentByID(post.RefId)
			if err != nil {
				return nil, err
			}
			refContentsFormatted := make([]*model.PostContentFormatted, len(refContents))
			for i := range refContentsFormatted {
				refContentsFormatted[i] = refContents[i].Format()
			}
			postFormatted.OrigContents = append(postFormatted.OrigContents, refContentsFormatted...)
		case model.RefComment:
			refComments, err := ds.GetCommentContentsByIDs([]primitive.ObjectID{post.RefId})
			if err != nil {
				return nil, err
			}
			refCommentsFormatted := make([]*model.PostContentFormatted, len(refComments))
			for i := range refCommentsFormatted {
				refCommentsFormatted[i] = refComments[i].PostFormat()
			}
			postFormatted.OrigContents = append(postFormatted.OrigContents, refCommentsFormatted...)
		case model.RefCommentReply:
			refReplies, err := ds.GetCommentReplyByID(post.RefId)
			if err != nil {
				return nil, err
			}
			postFormatted.OrigContents = append(postFormatted.OrigContents, refReplies.PostFormat())
		}
	}

	users, err := ds.GetUsersByAddresses([]string{post.Address, post.AuthorId})
	if err != nil {
		return nil, err
	}
	daoList, err := ds.GetDaoList(model.ConditionsT{
		"query": bson.M{"_id": bson.M{"$in": []primitive.ObjectID{post.DaoId, post.AuthorDaoId}}},
	}, 0, 0)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]*model.User)
	for _, user := range users {
		userMap[user.Address] = user
	}

	daoMap := make(map[primitive.ObjectID]*model.Dao)
	for _, dao := range daoList {
		daoMap[dao.ID] = dao
	}

	postFormatted.User = userMap[post.Address].Format()
	postFormatted.Dao = daoMap[post.DaoId].Format()
	if postFormatted.Dao.Address == user {
		postFormatted.Dao.IsJoined = true
		postFormatted.Dao.IsSubscribed = true
	} else {
		postFormatted.Dao.IsJoined = CheckJoinedDAO(user, post.DaoId)
		postFormatted.Dao.IsSubscribed = CheckSubscribeDAO(user, post.DaoId)
	}

	if postFormatted.AuthorId != "" {
		postFormatted.Author = userMap[post.AuthorId].Format()
	}

	if !postFormatted.AuthorDaoId.IsZero() {
		postFormatted.AuthorDao = daoMap[post.AuthorDaoId].Format()
		if postFormatted.AuthorDao.Address == user {
			postFormatted.AuthorDao.IsJoined = true
			postFormatted.AuthorDao.IsSubscribed = true
		} else {
			postFormatted.AuthorDao.IsJoined = CheckJoinedDAO(user, post.AuthorDaoId)
			postFormatted.AuthorDao.IsSubscribed = CheckSubscribeDAO(user, post.AuthorDaoId)
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

func GetPostList(user string, req *PostListReq) ([]*model.PostFormatted, error) {
	posts, err := ds.GetPosts(req.Conditions, req.Offset, req.Limit)

	if err != nil {
		return nil, err
	}

	return ds.MergePosts(user, posts)
}

func GetPostCount(conditions *model.ConditionsT) (int64, error) {
	return ds.GetPostCount(conditions)
}

func GetPostListFromSearch(user *model.User, q *core.QueryReq, offset, limit int) ([]*model.PostFormatted, int64, error) {
	if len(q.Type) == 1 && q.Type[0] == model.DAO {
		if q.Query != "" {
			q.Visibility = []core.PostVisibleT{core.PostVisitPublic, core.PostVisitPrivate}
			q.BlockDaoIDs = []string{}
		} else {
			q.Visibility = []core.PostVisibleT{core.PostVisitPublic}
		}
	}
	resp, err := ts.Search(q, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if user == nil {
		user = &model.User{}
	}
	posts, err := ds.RevampPosts(user.Address, resp.Items)
	if err != nil {
		return nil, 0, err
	}
	return posts, resp.Total, nil
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
		"dao_follow_count":  0,
		"view_count":        post.ViewCount,
		"collection_count":  post.CollectionCount,
		"upvote_count":      post.UpvoteCount,
		"comment_count":     post.CommentCount,
		"ref_count":         post.RefCount,
		"member":            post.Member,
		"visibility":        post.Visibility,
		"is_top":            post.IsTop,
		"is_essence":        post.IsEssence,
		"content":           contentFormatted,
		"tags":              tagMaps,
		"type":              post.Type,
		"orig_type":         post.OrigType,
		"origCreatedAt":     post.OrigCreatedAt,
		"author_id":         post.AuthorId,
		"author_dao_id":     post.AuthorDaoId,
		"ref_id":            post.RefId,
		"ref_type":          post.RefType,
		"created_on":        post.CreatedOn,
		"modified_on":       post.ModifiedOn,
		"latest_replied_on": post.LatestRepliedOn,
	}}

	ts.AddDocuments(data, post.ID.Hex())
}

func DeleteSearchPost(post *model.Post, refPostIds ...primitive.ObjectID) error {
	ids := []string{post.ID.Hex()}

	for _, refPostId := range refPostIds {
		ids = append(ids, refPostId.Hex())
	}

	return ts.DeleteDocuments(ids)
}

func PushPostsToSearch() {
	splitNum := 1000
	totalRows, _ := GetPostCount(&model.ConditionsT{
		"query": bson.M{"visibility": model.PostVisitPublic},
	})

	pages := math.Ceil(float64(totalRows) / float64(splitNum))
	nums := int(pages)

	for i := 0; i < nums; i++ {
		posts, _ := GetPostList("", &PostListReq{
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
				"dao_follow_count":  0,
				"view_count":        post.ViewCount,
				"collection_count":  post.CollectionCount,
				"upvote_count":      post.UpvoteCount,
				"comment_count":     post.CommentCount,
				"ref_count":         post.RefCount,
				"member":            post.Member,
				"visibility":        post.Visibility,
				"is_top":            post.IsTop,
				"is_essence":        post.IsEssence,
				"content":           contentFormatted,
				"tags":              post.Tags,
				"type":              post.Type,
				"orig_type":         post.OrigType,
				"origCreatedAt":     post.OrigCreatedAt,
				"author_id":         post.AuthorId,
				"author_dao_id":     post.AuthorDaoId,
				"ref_id":            post.RefId,
				"ref_type":          post.RefType,
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
		conditions = &model.ConditionsT{
			"ORDER": bson.M{"quote_num": -1},
		}
	}
	if param.Type == TagTypeNew {
		conditions = &model.ConditionsT{
			"ORDER": bson.M{"id": -1},
		}
	}

	tags, err := ds.GetTags(conditions, 0, num)
	if err != nil {
		return nil, err
	}

	userIds := []string{}
	for _, tag := range tags {
		userIds = append(userIds, tag.Address)
	}

	users, _ := ds.GetUsersByAddresses(userIds)

	var tagsFormatted []*model.TagFormatted
	for _, tag := range tags {
		tagFormatted := tag.Format()
		for _, user := range users {
			if user.Address == tagFormatted.Address {
				tagFormatted.User = user.Format()
			}
		}
		tagsFormatted = append(tagsFormatted, tagFormatted)
	}

	return tagsFormatted, nil
}

func FilterMemberContent(user *model.User, post *model.PostFormatted) *model.PostFormatted {
	// Warning, Other related places tweetHelpServant.filterMemberContent
	if post.Member == model.PostMemberNothing {
		return post
	}
	if post.Type == model.VIDEO {
		if user == nil || (user.Address != post.Address && !CheckSubscribeDAO(user.Address, post.DaoId)) {
			for k, v := range post.Contents {
				if v.Type == model.CONTENT_TYPE_VIDEO {
					post.Contents[k].Content = ""
				}
			}
		}
	} else if post.OrigType == model.VIDEO {
		if user == nil || (user.Address != post.AuthorId && !CheckSubscribeDAO(user.Address, post.AuthorDaoId)) {
			for k, v := range post.OrigContents {
				if v.Type == model.CONTENT_TYPE_VIDEO {
					post.OrigContents[k].Content = ""
				}
			}
		}
	}
	return post
}

type PostComplaintReq struct {
	PostID string `json:"post_id"  binding:"required"`
	Reason string `json:"reason"   binding:"required"`
}

func ComplaintPost(user *model.User, req PostComplaintReq) error {
	id, err := primitive.ObjectIDFromHex(req.PostID)
	if err != nil {
		return err
	}
	_, err = ds.GetPostByID(id)
	if err != nil {
		return err
	}
	md := model.PostComplaint{
		Address: user.Address,
		PostID:  id,
		Reason:  req.Reason,
	}
	return md.Create(context.Background(), conf.MustMongoDB())
}

func BlockPost(user *model.User, id primitive.ObjectID) error {
	_, err := ds.GetPostByID(id)
	if err != nil {
		return err
	}
	md := model.PostBlock{
		Address: user.Address,
		BlockId: id,
		Model:   model.BlockModelPost,
	}
	return md.Create(context.Background(), conf.MustMongoDB())
}

func GetBlockPostIDs(user *model.User) []string {
	md := model.PostBlock{}
	ops := &options.FindOptions{}
	ops.SetLimit(300)
	ops.SetSort(bson.M{"created_on": -1})
	return md.FindIDs(context.TODO(), conf.MustMongoDB(), bson.M{
		"address": user.Address,
		"model":   model.BlockModelPost,
	}, ops)
}

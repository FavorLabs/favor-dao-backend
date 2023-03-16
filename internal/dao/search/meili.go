package search

import (
	"fmt"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/json"
	"github.com/Masterminds/semver/v3"
	"github.com/meilisearch/meilisearch-go"
	"github.com/sirupsen/logrus"
)

var (
	_ core.TweetSearchService = (*meiliTweetSearchServant)(nil)
	_ core.VersionInfo        = (*meiliTweetSearchServant)(nil)
)

type meiliTweetSearchServant struct {
	tweetSearchFilter

	client        *meilisearch.Client
	index         *meilisearch.Index
	publicFilter  string
	privateFilter string
	friendFilter  string
}

func (s *meiliTweetSearchServant) Name() string {
	return "Meili"
}

func (s *meiliTweetSearchServant) Version() *semver.Version {
	return semver.MustParse("v0.2.0")
}

func (s *meiliTweetSearchServant) IndexName() string {
	return s.index.UID
}

func (s *meiliTweetSearchServant) AddDocuments(data core.DocItems, primaryKey ...string) (bool, error) {
	if len(data) == 0 {
		return true, nil
	}
	if _, err := s.index.AddDocuments(data, primaryKey...); err != nil {
		logrus.Errorf("meiliTweetSearchServant.AddDocuments error: %s", err)
		return false, err
	}
	return true, nil
}

func (s *meiliTweetSearchServant) DeleteDocuments(identifiers []string) error {
	task, err := s.index.DeleteDocuments(identifiers)
	if err != nil {
		logrus.Errorf("meiliTweetSearchServant.DeleteDocuments error: %s", err)
		return err
	}
	logrus.Debugf("meiliTweetSearchServant.DeleteDocuments task: (taskUID:%d, indexUID:%s, status:%s)", task.TaskUID, task.IndexUID, task.Status)
	return nil
}

func (s *meiliTweetSearchServant) Search(user *model.User, q *core.QueryReq, offset, limit int) (resp *core.QueryResp, err error) {
	if q.Search == core.SearchTypeDefault && q.Query != "" {
		resp, err = s.queryByContent(user, q, offset, limit)
	} else if q.Search == core.SearchTypeTag && q.Query != "" {
		resp, err = s.queryByTag(user, q, offset, limit)
	} else if q.Search == core.SearchTypeAddress && q.Query != "" {
		resp, err = s.queryByAddress(user, q, offset, limit)
	} else {
		resp, err = s.queryAny(user, offset, limit)
	}
	if err != nil {
		logrus.Errorf("meiliTweetSearchServant.search searchType:%s query:%s error:%v", q.Type, q.Query, err)
		return
	}

	logrus.Debugf("meiliTweetSearchServant.Search type:%s query:%s resp Hits:%d NbHits:%d offset: %d limit:%d ", q.Type, q.Query, len(resp.Items), resp.Total, offset, limit)
	s.filterResp(user, resp, q)
	return
}

func (s *meiliTweetSearchServant) queryByContent(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	request := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
		Sort:   []string{"is_top:desc", "modified_on:desc"},
	}

	filter := s.filterList(user)
	if len(filter) > 0 {
		request.Filter = filter
	}

	// logrus.Debugf("meiliTweetSearchServant.queryByContent query:%s request%+v", q.Query, request)
	resp, err := s.index.Search(q.Query, request)
	if err != nil {
		return nil, err
	}

	return s.postsFrom(resp)
}

func (s *meiliTweetSearchServant) queryByTag(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	request := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
		Sort:   []string{"is_top:desc", "modified_on:desc"},
	}

	filter := s.filterList(user)
	tagFilter := []string{"tags." + q.Query + "=1"}
	if len(filter) > 0 {
		request.Filter = [][]string{tagFilter, {filter}}
	} else {
		request.Filter = tagFilter
	}

	// logrus.Debugf("meiliTweetSearchServant.queryByTag query:%s request%+v", q.Query, request)
	resp, err := s.index.Search("#"+q.Query, request)
	if err != nil {
		return nil, err
	}

	return s.postsFrom(resp)
}

func (s *meiliTweetSearchServant) queryByAddress(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	request := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
		Sort:   []string{"is_top:desc", "modified_on:desc"},
	}

	filter := s.filterList(user)
	request.Filter = filter
	request.AttributesToRetrieve = []string{"address"}

	// logrus.Debugf("meiliTweetSearchServant.queryByContent query:%s request%+v", q.Query, request)
	resp, err := s.index.Search(q.Query, request)
	if err != nil {
		return nil, err
	}

	return s.postsFrom(resp)
}

func (s *meiliTweetSearchServant) queryAny(user *model.User, offset, limit int) (*core.QueryResp, error) {
	request := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
		Sort:   []string{"is_top:desc", "modified_on:desc"},
	}

	filter := s.filterList(user)
	if len(filter) > 0 {
		request.Filter = filter
	}

	resp, err := s.index.Search("", request)
	if err != nil {
		return nil, err
	}

	return s.postsFrom(resp)
}

func (s *meiliTweetSearchServant) filterList(user *model.User) string {
	if user == nil {
		return s.publicFilter
	}

	return fmt.Sprintf("%s OR (%s%s)", s.publicFilter, s.privateFilter, user.Address)
}

func (s *meiliTweetSearchServant) postsFrom(resp *meilisearch.SearchResponse) (*core.QueryResp, error) {
	posts := make([]*model.PostFormatted, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		item := &model.PostFormatted{}
		raw, err := json.Marshal(hit)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, item); err != nil {
			return nil, err
		}
		posts = append(posts, item)
	}

	return &core.QueryResp{
		Items: posts,
		Total: resp.TotalHits,
	}, nil
}

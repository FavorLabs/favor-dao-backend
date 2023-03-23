package search

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/json"
	"favor-dao-backend/pkg/types"
	"favor-dao-backend/pkg/zinc"
	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

var (
	_ core.TweetSearchService = (*zincTweetSearchServant)(nil)
	_ core.VersionInfo        = (*zincTweetSearchServant)(nil)
)

type zincTweetSearchServant struct {
	tweetSearchFilter

	indexName     string
	client        *zinc.ZincClient
	publicFilter  string
	privateFilter string
	friendFilter  string
}

func (s *zincTweetSearchServant) Name() string {
	return "Zinc"
}

func (s *zincTweetSearchServant) Version() *semver.Version {
	return semver.MustParse("v0.2.0")
}

func (s *zincTweetSearchServant) IndexName() string {
	return s.indexName
}

func (s *zincTweetSearchServant) AddDocuments(data core.DocItems, primaryKey ...string) (bool, error) {
	if len(data) == 0 {
		return true, nil
	}
	buf := make(core.DocItems, 0, len(data)+1)
	if len(primaryKey) > 0 {
		buf = append(buf, map[string]types.Any{
			"index": map[string]types.Any{
				"_index": s.indexName,
				"_id":    primaryKey[0],
			},
		})
	} else {
		buf = append(buf, map[string]types.Any{
			"index": map[string]types.Any{
				"_index": s.indexName,
			},
		})
	}
	buf = append(buf, data...)
	return s.client.BulkPushDoc(buf)
}

func (s *zincTweetSearchServant) DeleteDocuments(identifiers []string) error {
	for _, id := range identifiers {
		if err := s.client.DelDoc(s.indexName, id); err != nil {
			logrus.Errorf("deleteDocuments %s error: %s", id, err)
			return err
		}
	}
	return nil
}

func (s *zincTweetSearchServant) Search(q *core.QueryReq, offset, limit int) (resp *core.QueryResp, err error) {
	resp, err = s.queryAny(q, offset, limit)
	if err != nil {
		logrus.Errorf("zincTweetSearchServant.search query:%v error:%v", q, err)
		return
	}

	logrus.Debugf("zincTweetSearchServant.Search query:%v resp Hits:%d NbHits:%d offset: %d limit:%d ", q, len(resp.Items), resp.Total, offset, limit)
	return
}

func (s *zincTweetSearchServant) queryAny(q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	must := types.AnySlice{}
	if len(q.Type) > 0 {
		must = append(must, map[string]types.Any{
			"terms": map[string]types.Any{
				"type": q.Type,
			},
		})
	}
	if len(q.DaoIDs) > 0 {
		must = append(must, map[string]types.Any{
			"terms": map[string]types.Any{
				"dao_id": q.DaoIDs,
			},
		})
	}
	if len(q.Addresses) > 0 {
		must = append(must, map[string]types.Any{
			"terms": map[string]types.Any{
				"address": q.Addresses,
			},
		})
	}
	if len(q.Visibility) == 0 {
		must = append(must, map[string]types.Any{
			"terms": map[string]types.Any{
				"visibility": types.AnySlice{core.PostVisitPublic}, // default public
			},
		})
	} else {
		must = append(must, map[string]types.Any{
			"terms": map[string]types.Any{
				"visibility": q.Visibility,
			},
		})
	}
	if q.Tag != "" {
		must = append(must, map[string]types.Any{
			"term": map[string]types.Any{
				"tags." + q.Tag: 1,
			},
		})
	}
	if q.Query != "" {
		must = append(must, map[string]types.Any{
			"match_phrase": map[string]types.Any{
				"content": q.Query,
			},
		})
	}
	query := map[string]types.Any{}
	if len(must) == 0 {
		query["match_all"] = map[string]string{}
	} else {
		query["bool"] = map[string]types.Any{
			"must": must,
		}
	}
	sort := types.AnySlice{}
	sort = append(sort, map[string]types.Any{
		"created_on": "desc",
	})
	queryMap := map[string]types.Any{
		"query": query,
		"sort":  sort,
		"from":  offset,
		"size":  limit,
	}
	resp, err := s.client.EsQuery(s.indexName, queryMap)
	if err != nil {
		return nil, err
	}
	return s.postsFrom(resp)
}

func (s *zincTweetSearchServant) postsFrom(resp *zinc.QueryResultT) (*core.QueryResp, error) {
	posts := make([]*model.PostFormatted, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		item := &model.PostFormatted{}
		raw, err := json.Marshal(hit.Source)
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
		Total: resp.Hits.Total.Value,
	}, nil
}

func (s *zincTweetSearchServant) createIndex() {
	// Create index if it does not exist
	s.client.CreateIndex(s.indexName, &zinc.ZincIndexProperty{
		"id": &zinc.ZincIndexPropertyT{
			Type:     "text",
			Index:    true,
			Store:    true,
			Sortable: true,
		},
		"address": &zinc.ZincIndexPropertyT{
			Type:  "text",
			Index: true,
			Store: true,
		},
		"dao_id": &zinc.ZincIndexPropertyT{
			Type:  "text",
			Index: true,
			Store: true,
		},
		"view_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"collection_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"upvote_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"member": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"visibility": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"is_top": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"is_essence": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"content": &zinc.ZincIndexPropertyT{
			Type:           "text",
			Index:          true,
			Store:          true,
			Aggregatable:   true,
			Highlightable:  true,
			Analyzer:       "gse_search",
			SearchAnalyzer: "gse_standard",
		},
		"tags": &zinc.ZincIndexPropertyT{
			Type:  "keyword",
			Index: true,
			Store: true,
		},
		"type": &zinc.ZincIndexPropertyT{
			Type:  "numeric",
			Index: true,
			Store: true,
		},
		"created_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"modified_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"latest_replied_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
	})
}

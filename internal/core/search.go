package core

import (
	"favor-dao-backend/internal/model"
)

const (
	SearchTypeDefault SearchType = "search"
	SearchTypeTag     SearchType = "tag"
	SearchTypeAddress SearchType = "address"
)

type (
	SearchType string

	QueryReq struct {
		Query      string
		Visibility []PostVisibleT
		Type       string
		Search     SearchType
	}

	QueryResp struct {
		Items []*PostFormatted
		Total int64
	}
)

type DocItems []map[string]interface{}

// TweetSearchService tweet search service interface
type TweetSearchService interface {
	IndexName() string
	AddDocuments(documents DocItems, primaryKey ...string) (bool, error)
	DeleteDocuments(identifiers []string) error
	Search(user *model.User, q *QueryReq, offset, limit int) (*QueryResp, error)
}

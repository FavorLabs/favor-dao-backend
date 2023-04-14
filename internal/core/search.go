package core

import "favor-dao-backend/pkg/types"

type (
	SearchType string

	QueryReq struct {
		Query      string
		Visibility []PostVisibleT
		Type       []PostType
		DaoIDs     []string
		Addresses  []string
		Tag        string
		Sort       types.AnySlice
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
	Search(q *QueryReq, offset, limit int) (*QueryResp, error)
}

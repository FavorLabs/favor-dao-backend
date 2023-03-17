package search

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
)

type tweetSearchFilter struct {
	ams core.AuthorizationManageService
}

func (s *tweetSearchFilter) filterResp(user *model.User, resp *core.QueryResp, q *core.QueryReq) {
	var item *model.PostFormatted
	items := resp.Items
	latestIndex := len(items) - 1
	if user == nil {
		for i := 0; i <= latestIndex; i++ {
			item = items[i]
			if item.Visibility != model.PostVisitPublic {
				items[i] = items[latestIndex]
				items = items[:latestIndex]
				resp.Total--
				latestIndex--
				i--
			}
		}
	} else {
		var cutPrivate bool
		for i := 0; i <= latestIndex; i++ {
			item = items[i]
			cutPrivate = item.Visibility == model.PostVisitPrivate && user.Address != item.Address
			if cutPrivate {
				items[i] = items[latestIndex]
				items = items[:latestIndex]
				resp.Total--
				latestIndex--
				i--
			}
		}
	}

	resp.Items = items
}

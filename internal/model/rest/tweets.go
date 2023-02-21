package rest

import "favor-dao-backend/internal/model"

type IndexTweetsResp struct {
	Tweets []*model.PostFormatted
	Total  int64
}

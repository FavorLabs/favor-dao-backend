package rest

import "favor-dao-backend/internal/model"

type IndexTweetsResp struct {
	Tweets []*model.PostFormated
	Total  int64
}

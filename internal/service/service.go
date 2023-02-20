package service

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao"
	"favor-dao-backend/internal/model"
)

var (
	ds core.DataService
	ts core.TweetSearchService
)

func Initialize() {
	ds = dao.DataService()
	ts = dao.TweetSearchService()
}

func persistMediaContents(contents []*PostContentItem) (items []string, err error) {
	items = make([]string, 0, len(contents))
	for _, item := range contents {
		switch item.Type {
		case model.CONTENT_TYPE_IMAGE,
			model.CONTENT_TYPE_VIDEO,
			model.CONTENT_TYPE_AUDIO:
			items = append(items, item.Content)
			if err != nil {
				continue
			}
		}
	}
	return
}

package cache

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/model/rest"
	"github.com/Masterminds/semver/v3"
)

var (
	_ core.CacheIndexService = (*noneCacheIndexServant)(nil)
	_ core.VersionInfo       = (*noneCacheIndexServant)(nil)
)

type noneCacheIndexServant struct {
	ips core.IndexPostsService
}

func (s *noneCacheIndexServant) IndexPosts(user *model.User, offset int, limit int) (*rest.IndexTweetsResp, error) {
	return s.ips.IndexPosts(user, offset, limit)
}

func (s *noneCacheIndexServant) SendAction(_act core.IdxAct, _post *model.Post) {
	// empty
}

func (s *noneCacheIndexServant) Name() string {
	return "NoneCacheIndex"
}

func (s *noneCacheIndexServant) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

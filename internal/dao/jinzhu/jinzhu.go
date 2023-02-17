// Core service implement base gorm+mysql/postgresql/sqlite3.
// Jinzhu is the primary developer of gorm so use his name as
// pakcage name as a saluter.

package jinzhu

import (
	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao/cache"
	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.VersionInfo = (*dataServant)(nil)
)

type dataServant struct{}

func NewDataService() (core.CacheIndexService, *mongo.Database) {
	// initialize CacheIndex if needed
	var (
		c core.CacheIndexService
		v core.VersionInfo
	)
	db := conf.MustGormDB()

	i := newIndexPostsService(db)
	if conf.CfgIf("SimpleCacheIndex") {
		i = newSimpleIndexPostsService(db)
		c, v = cache.NewSimpleCacheIndexService(i)
	} else if conf.CfgIf("BigCacheIndex") {
		c, v = cache.NewBigCacheIndexService(i)
	} else {
		c, v = cache.NewNoneCacheIndexService(i)
	}
	logrus.Infof("use %s as cache index service by version: %s", v.Name(), v.Version())

	return c, db
}

func NewAuthorizationManageService() core.AuthorizationManageService {
	return &authorizationManageServant{
		db: conf.MustGormDB(),
	}
}

func (s *dataServant) Name() string {
	return "Mongo"
}

func (s *dataServant) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}

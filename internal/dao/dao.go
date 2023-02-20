package dao

import (
	"sync"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao/jinzhu"
	"favor-dao-backend/internal/dao/search"
	"github.com/sirupsen/logrus"
)

var (
	ts core.TweetSearchService
	ds core.DataService

	onceTs, onceDs sync.Once
)

func DataService() core.DataService {
	onceDs.Do(func() {
		var v core.VersionInfo
		ds, v = jinzhu.NewDataService()
		logrus.Infof("use %s as data service with version %s", v.Name(), v.Version())
	})
	return ds
}

func TweetSearchService() core.TweetSearchService {
	onceTs.Do(func() {
		var v core.VersionInfo
		ams := newAuthorizationManageService()
		if conf.CfgIf("Zinc") {
			ts, v = search.NewZincTweetSearchService(ams)
		} else if conf.CfgIf("Meili") {
			ts, v = search.NewMeiliTweetSearchService(ams)
		} else {
			// default use Zinc as tweet search service
			ts, v = search.NewZincTweetSearchService(ams)
		}
		logrus.Infof("use %s as tweet search serice by version %s", v.Name(), v.Version())

		ts = search.NewBridgeTweetSearchService(ts)
	})
	return ts
}

func newAuthorizationManageService() (s core.AuthorizationManageService) {
	s = jinzhu.NewAuthorizationManageService()
	return
}

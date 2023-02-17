package dao

import (
	"sync"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao/jinzhu"
	"favor-dao-backend/internal/dao/search"
	"favor-dao-backend/internal/dao/storage"
	"github.com/sirupsen/logrus"
)

var (
	ts  core.TweetSearchService
	ds  core.DataService
	oss core.ObjectStorageService

	onceTs, onceDs, onceOss sync.Once
)

func DataService() core.DataService {
	onceDs.Do(func() {
		var v core.VersionInfo
		ds, v = jinzhu.NewDataService()
		logrus.Infof("use %s as data service with version %s", v.Name(), v.Version())
	})
	return ds
}

func ObjectStorageService() core.ObjectStorageService {
	onceOss.Do(func() {
		var v core.VersionInfo
		if conf.CfgIf("AliOSS") {
			oss, v = storage.MustAliossService()
		} else if conf.CfgIf("COS") {
			oss, v = storage.NewCosService()
		} else if conf.CfgIf("HuaweiOBS") {
			oss, v = storage.MustHuaweiobsService()
		} else if conf.CfgIf("MinIO") {
			oss, v = storage.MustMinioService()
		} else if conf.CfgIf("S3") {
			oss, v = storage.MustS3Service()
			logrus.Infof("use S3 as object storage by version %s", v.Version())
			return
		} else if conf.CfgIf("LocalOSS") {
			oss, v = storage.MustLocalossService()
		} else {
			// default use AliOSS as object storage service
			oss, v = storage.MustAliossService()
			logrus.Infof("use default AliOSS as object storage by version %s", v.Version())
			return
		}
		logrus.Infof("use %s as object storage by version %s", v.Name(), v.Version())
	})
	return oss
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

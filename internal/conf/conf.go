package conf

import (
	"log"
	"time"
)

var (
	loggerSetting      *LoggerSettingS
	loggerFileSetting  *LoggerFileSettingS
	loggerZincSetting  *LoggerZincSettingS
	loggerMeiliSetting *LoggerMeiliSettingS
	redisSetting       *RedisSettingS
	features           *FeaturesSettingS

	DatabaseSetting         *DatabaseSettingS
	MongoDBSetting          *MongoDBSettingS
	ServerSetting           *ServerSettingS
	AppSetting              *AppSettingS
	CacheIndexSetting       *CacheIndexSettingS
	SimpleCacheIndexSetting *SimpleCacheIndexSettingS
	BigCacheIndexSetting    *BigCacheIndexSettingS
	TweetSearchSetting      *TweetSearchS
	ZincSetting             *ZincSettingS
	MeiliSetting            *MeiliSettingS
	ObjectStorage           *ObjectStorageS
	AliOSSSetting           *AliOSSSettingS
	COSSetting              *COSSettingS
	HuaweiOBSSetting        *HuaweiOBSSettingS
	MinIOSetting            *MinIOSettingS
	S3Setting               *S3SettingS
	LocalOSSSetting         *LocalOSSSettingS
)

func setupSetting(suite []string, noDefault bool) error {
	setting, err := NewSetting()
	if err != nil {
		return err
	}

	features = setting.FeaturesFrom("Features")
	if len(suite) > 0 {
		if err = features.Use(suite, noDefault); err != nil {
			return err
		}
	}

	objects := map[string]interface{}{
		"App":              &AppSetting,
		"Server":           &ServerSetting,
		"CacheIndex":       &CacheIndexSetting,
		"SimpleCacheIndex": &SimpleCacheIndexSetting,
		"BigCacheIndex":    &BigCacheIndexSetting,
		"Logger":           &loggerSetting,
		"LoggerFile":       &loggerFileSetting,
		"LoggerZinc":       &loggerZincSetting,
		"LoggerMeili":      &loggerMeiliSetting,
		"Database":         &DatabaseSetting,
		"Mongo":            &MongoDBSetting,
		"TweetSearch":      &TweetSearchSetting,
		"Zinc":             &ZincSetting,
		"Meili":            &MeiliSetting,
		"Redis":            &redisSetting,
		"ObjectStorage":    &ObjectStorage,
		"AliOSS":           &AliOSSSetting,
		"COS":              &COSSetting,
		"HuaweiOBS":        &HuaweiOBSSetting,
		"MinIO":            &MinIOSetting,
		"LocalOSS":         &LocalOSSSetting,
		"S3":               &S3Setting,
	}
	if err = setting.Unmarshal(objects); err != nil {
		return err
	}

	ServerSetting.ReadTimeout *= time.Second
	ServerSetting.WriteTimeout *= time.Second
	SimpleCacheIndexSetting.CheckTickDuration *= time.Second
	SimpleCacheIndexSetting.ExpireTickDuration *= time.Second
	BigCacheIndexSetting.ExpireInSecond *= time.Second

	return nil
}

func Initialize(suite []string, noDefault bool) {
	err := setupSetting(suite, noDefault)
	if err != nil {
		log.Fatalf("init.setupSetting err: %v", err)
	}

	setupLogger()
	setupDBEngine()
}

// Cfg get value by key if exist
func Cfg(key string) (string, bool) {
	return features.Cfg(key)
}

// CfgIf check expression is true. if expression just have a string like
// `Sms` is mean `Sms` whether define in suite feature settings. expression like
// `Sms = SmsJuhe` is mean whether `Sms` define in suite feature settings and value
// is `SmsJuhe``
func CfgIf(expression string) bool {
	return features.CfgIf(expression)
}

func GetOssDomain() string {
	uri := "https://"
	if CfgIf("AliOSS") {
		return uri + AliOSSSetting.Domain + "/"
	} else if CfgIf("COS") {
		return uri + COSSetting.Domain + "/"
	} else if CfgIf("HuaweiOBS") {
		return uri + HuaweiOBSSetting.Domain + "/"
	} else if CfgIf("MinIO") {
		if !MinIOSetting.Secure {
			uri = "http://"
		}
		return uri + MinIOSetting.Domain + "/" + MinIOSetting.Bucket + "/"
	} else if CfgIf("S3") {
		if !S3Setting.Secure {
			uri = "http://"
		}
		// TODO: will not work well need test in real world
		return uri + S3Setting.Domain + "/" + S3Setting.Bucket + "/"
	} else if CfgIf("LocalOSS") {
		if !LocalOSSSetting.Secure {
			uri = "http://"
		}
		return uri + LocalOSSSetting.Domain + "/oss/" + LocalOSSSetting.Bucket + "/"
	}
	return uri + AliOSSSetting.Domain + "/"
}

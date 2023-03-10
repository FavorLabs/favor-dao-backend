package conf

import (
	"log"
	"os"
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
	ExternalAppSetting      *ExternalAppSettingS
	CacheIndexSetting       *CacheIndexSettingS
	SimpleCacheIndexSetting *SimpleCacheIndexSettingS
	BigCacheIndexSetting    *BigCacheIndexSettingS
	TweetSearchSetting      *TweetSearchS
	ZincSetting             *ZincSettingS
	MeiliSetting            *MeiliSettingS
	EthSetting              *EthSettingS
	ChatSetting             *ChatSettingS
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
		"External":         &ExternalAppSetting,
		"Server":           &ServerSetting,
		"CacheIndex":       &CacheIndexSetting,
		"SimpleCacheIndex": &SimpleCacheIndexSetting,
		"BigCacheIndex":    &BigCacheIndexSetting,
		"Logger":           &loggerSetting,
		"LoggerFile":       &loggerFileSetting,
		"LoggerZinc":       &loggerZincSetting,
		"LoggerMeili":      &loggerMeiliSetting,
		"Database":         &DatabaseSetting,
		"MongoDB":          &MongoDBSetting,
		"TweetSearch":      &TweetSearchSetting,
		"Zinc":             &ZincSetting,
		"Meili":            &MeiliSetting,
		"Redis":            &redisSetting,
		"Eth":              &EthSetting,
		"Chat":             &ChatSetting,
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

	if ChatSetting == nil || ChatSetting.Api == "" {
		panic("MUST connect with revolt api")
	}

	// set default timezone
	_ = os.Setenv("TZ", "UTC")

	setupLogger()
	setupDBEngine()
}

// Cfg get value by key if exist
func Cfg(key string) (string, bool) {
	return features.Cfg(key)
}

// CfgIf check expression is true. if expression just have a string like
func CfgIf(expression string) bool {
	return features.CfgIf(expression)
}

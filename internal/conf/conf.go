package conf

import (
	"log"
	"os"
	"reflect"
	"strings"
	"time"
)

var (
	loggerSetting      *LoggerSettingS
	loggerFileSetting  *LoggerFileSettingS
	loggerZincSetting  *LoggerZincSettingS
	loggerMeiliSetting *LoggerMeiliSettingS
	features           *FeaturesSettingS

	RedisSetting            *RedisSettingS
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
	PointSetting            *PointSettingS
	NotifySetting           *NotifySettingS
)

func setupSetting(suite []string, noDefault bool, configPath ...string) error {
	setting, err := NewSetting(configPath...)
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
		"Redis":            &RedisSetting,
		"Eth":              &EthSetting,
		"Chat":             &ChatSetting,
		"Point":            &PointSetting,
		"Notify":           &NotifySetting,
	}
	if err = setting.Unmarshal(objects); err != nil {
		return err
	}

	ServerSetting.ReadTimeout *= time.Second
	ServerSetting.WriteTimeout *= time.Second
	ServerSetting.CancellationTimeInterval *= time.Minute
	SimpleCacheIndexSetting.CheckTickDuration *= time.Second
	SimpleCacheIndexSetting.ExpireTickDuration *= time.Second
	BigCacheIndexSetting.ExpireInSecond *= time.Second
	ExternalAppSetting.RedPacketTimeout *= time.Second

	return nil
}

func Initialize(suite []string, noDefault bool, configPath ...string) {
	err := setupSetting(suite, noDefault, configPath...)
	if err != nil {
		log.Fatalf("init.setupSetting err: %v", err)
	}

	CheckSetting(PointSetting, "gateway", "callback")
	CheckSetting(NotifySetting, "gateway")
	CheckSetting(ChatSetting, "appid", "region", "apikey")
	CheckSetting(EthSetting, "endpoint")

	// set default timezone
	_ = os.Setenv("TZ", "UTC")

	setupLogger()
	setupDBEngine()
}

func CheckSetting(i interface{}, keys ...string) {
	rv := reflect.ValueOf(i)

	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	for _, key := range keys {
		f := rv.FieldByNameFunc(func(s string) bool {
			return strings.ToLower(s) == key
		})
		if f.IsZero() {
			log.Fatalf("%s.%s must be filled", rv.Type().Name(), key)
		}
	}
}

// Cfg get value by key if exist
func Cfg(key string) (string, bool) {
	return features.Cfg(key)
}

// CfgIf check expression is true. if expression just have a string like
func CfgIf(expression string) bool {
	return features.CfgIf(expression)
}

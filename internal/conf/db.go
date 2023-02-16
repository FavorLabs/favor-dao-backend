package conf

import (
	"context"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var (
	db    *mongo.Database
	Redis *redis.Client
	once  sync.Once
)

func MustGormDB() *mongo.Database {
	once.Do(func() {
		var err error
		if db, err = newDBEngine(); err != nil {
			logrus.Fatalf("new mongo db failed: %s", err)
		}
	})
	return db
}

func newDBEngine() (*mongo.Database, error) {
	logrus.Debugln("use Mongo as db")
	option := options.Client().
		ApplyURI(MongoDBSetting.Dsn()).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetReadConcern(readconcern.Majority()).
		SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

	client, err := mongo.NewClient(option)
	if err != nil {
		return nil, err
	}
	if err = client.Connect(context.Background()); err != nil {
		return nil, err
	}
	db = client.Database(MongoDBSetting.DBName)
	return db, nil
}

func setupDBEngine() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     redisSetting.Host,
		Password: redisSetting.Password,
		DB:       redisSetting.DB,
	})
}

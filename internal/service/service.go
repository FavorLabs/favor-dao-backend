package service

import (
	"context"
	"fmt"
	"strings"

	"favor-dao-backend/pkg/notify"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/comet"
	"favor-dao-backend/pkg/pointSystem"
	"favor-dao-backend/pkg/psub"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis_rate/v10"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var (
	ds            core.DataService
	ts            core.TweetSearchService
	eth           *ethclient.Client
	chat          *comet.ChatGateway
	point         *pointSystem.Gateway
	pubsub        *psub.Service
	queue         *asynq.Client
	limiter       *redis_rate.Limiter
	notifyGateway *notify.Gateway
)

func Initialize() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.RedisSetting.Host,
		DB:       conf.RedisSetting.DB,
		Password: conf.RedisSetting.Password,
	})
	limiter = redis_rate.NewLimiter(rdb)

	setupJobServer()
	ds = dao.DataService()
	ts = dao.TweetSearchService()

	pubsub = psub.New()
	// MUST connect!
	client, err := ethclient.Dial(conf.EthSetting.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("dial eth: %s", err))
	}
	eth = client
	notifyGateway = notify.New(conf.NotifySetting.Gateway)
	if err != nil {
		panic(err)
	}
	chat = comet.New(conf.ChatSetting.AppId, conf.ChatSetting.Region, conf.ChatSetting.ApiKey)
	point = pointSystem.New(conf.PointSetting.Gateway)
	conf.PointSetting.Callback = strings.TrimRight(conf.PointSetting.Callback, "/")
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

const PostQueue = "post"

func setupJobServer() {
	resiConfig := asynq.RedisClientOpt{
		DB:       conf.RedisSetting.DB,
		Addr:     conf.RedisSetting.Host,
		Password: conf.RedisSetting.Password,
	}
	server := asynq.NewServer(
		resiConfig,
		asynq.Config{
			Logger: logrus.StandardLogger(),
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logrus.WithField("payload", string(task.Payload())).
					WithField("type", task.Type()).
					WithError(err).
					Errorf("task occur error")
			}),
			Concurrency:    1,
			StrictPriority: true,
			Queues: map[string]int{
				PostQueue:      10,
				QueueRedpacket: 10,
			},
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc(PostUnpin, HandlePostUnpinTask)
	mux.HandleFunc(TypeRedpacketDone, HandleRedpacketDoneTask)
	mux.HandleFunc(TypeRedpacketClaim, HandleRedpacketClaimTask)

	go func() {
		if err := server.Run(mux); err != nil {
			panic(err)
		}
	}()

	queue = asynq.NewClient(resiConfig)
}

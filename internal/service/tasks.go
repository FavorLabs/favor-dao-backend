package service

import (
	"context"
	"errors"
	"fmt"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/json"
	"favor-dao-backend/pkg/pointSystem"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RedpacketDonePayload struct {
	Id string
}

func NewRedpacketDoneTask(redpacketID string) *asynq.Task {
	payload, _ := json.Marshal(RedpacketDonePayload{Id: redpacketID})

	return asynq.NewTask(TypeRedpacketDone, payload)
}

func HandleRedpacketDoneTask(ctx context.Context, t *asynq.Task) (err error) {
	var p RedpacketDonePayload
	if err = json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v", err)
	}
	logrus.Debugf("Redpacket Done: id=%s\n", p.Id)

	m := model.Redpacket{}
	m.ID, err = primitive.ObjectIDFromHex(p.Id)
	if err != nil {
		return fmt.Errorf("objectID failed: %v", err)
	}
	// del redis key
	key := PrefixRedisKeyRedpacket + p.Id
	conf.Redis.Del(ctx, key)

	err = m.FindAndUpdate(ctx, conf.MustMongoDB(), nil, bson.M{"$set": bson.M{"is_timeout": true}})
	if err != nil {
		return err
	}
	_, err = model.UseTransaction(ctx, conf.MustMongoDB(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		err = m.First(sessCtx, conf.MustMongoDB())
		if err != nil {
			return nil, err
		}
		// check need refund
		if m.Total-m.ClaimCount > 0 && m.RefundTxID == "" {
			// pay
			m.RefundTxID, err = point.Pay(sessCtx, pointSystem.PayRequest{
				FromObject: conf.ExternalAppSetting.RedpacketAddress,
				ToSubject:  m.Address,
				Amount:     m.Balance,
				Comment:    "",
				Channel:    "refund_redpacket",
				ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=refund_redpacket&order_id=" + p.Id,
				BindOrder:  p.Id,
			})
		}
		if err != nil {
			return nil, err
		}
		return m, nil
	})
	return err
}

func eventRefundRedpacket(notify PayCallbackParam) error {
	ctx := context.Background()
	m := &model.Redpacket{}
	m.ID, _ = primitive.ObjectIDFromHex(notify.OrderId)
	err := m.First(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("refund_redpacket on notify: redpacket.First _id:%s err:%s", notify.OrderId, err)
		return err
	}
	m.RefundTxID = notify.TxID
	upFun := func() error {
		err = m.Update(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("refund_redpacket on notify: redpacket.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
			return err
		}
		return nil
	}
	switch notify.TxStatus {
	case TxCompleted:
		// success
		m.RefundStatus = model.PaySuccess
	case TxRollback, TxCancelled:
		// failed
		m.RefundStatus = model.PayFailed
	default:
		return nil
	}
	return upFun()
}

type PostUnpinPayload struct {
	Id primitive.ObjectID
}

const PostUnpin = "post:unpin"

func NewPostUnpinTask(postId primitive.ObjectID) *asynq.Task {
	payload, _ := json.Marshal(PostUnpinPayload{Id: postId})

	return asynq.NewTask(PostUnpin, payload)
}

func HandlePostUnpinTask(ctx context.Context, t *asynq.Task) (err error) {
	var p PostUnpinPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v", err)
	}
	logrus.Debugf("Unpin post: id=%s\n", p.Id)

	defer func() {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logrus.Warnf("post %s not found to unpin", p.Id)
			err = nil
		}
	}()

	post, err := ds.GetPostByID(p.Id)
	if err != nil {
		return err
	}

	if post.IsTop == 0 {
		return nil
	}

	err = ds.StickPost(post)
	if err != nil {
		return err
	}

	if err := DeleteSearchPost(post); err != nil {
		logrus.Warnf("post cannot remove from search engine: %v", err)
	}
	PushPostToSearch(post)

	return nil
}

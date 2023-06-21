package service

import (
	"context"
	"errors"
	"fmt"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
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

type RedpacketClaimPayload struct {
	Id      primitive.ObjectID
	Address string
}

func NewRedpacketClaimTask(redpacketID primitive.ObjectID, address string) *asynq.Task {
	payload, _ := json.Marshal(RedpacketClaimPayload{Id: redpacketID, Address: address})

	return asynq.NewTask(TypeRedpacketClaim, payload)
}

func HandleRedpacketClaimTask(ctx context.Context, t *asynq.Task) (err error) {
	var info RedpacketClaimPayload
	_ = json.Unmarshal(t.Payload(), &info)
	logrus.Debugf("Redpacket Claim: user=%s id=%s\n", info.Address, info.Id)
	key := PrefixRedisKeyRedpacket + info.Id.Hex()
	rr := &model.RedpacketClaim{}
	var e *errcode.Error
	defer func() {
		claimKey := fmt.Sprintf("%s%s:%s", PrefixRedisKeyRedpacketClaim, info.Id.Hex(), info.Address)
		pubsub.Notify(claimKey, &ClaimChResponse{
			Info: rr,
			err:  e,
		})
		if e == nil {
			if conf.Redis.Decr(ctx, key).Val() <= 0 {
				conf.Redis.Del(ctx, key)
			}
		}
	}()
	err = rr.FindOne(ctx, conf.MustMongoDB(), bson.M{
		"address":      info.Address,
		"redpacket_id": info.Id,
	})
	if err == nil {
		e = errcode.RedpacketAlreadyClaim
		return
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		e = errcode.ServerError.WithDetails(err.Error())
		return nil
	}
	err = model.UseTransaction(ctx, conf.MustMongoDB(), func(ctx context.Context) error {
		// Calculation amount
		redpacket := &model.Redpacket{}
		redpacket.ID = info.Id
		err = redpacket.First(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("ClaimRedpacket redpacket.First err:%s", err)
			return err
		}
		var price, balance string
		// ---
		if redpacket.Type == model.RedpacketTypeAverage {
			price = redpacket.AvgAmount
			b := convert.StrTo(redpacket.Balance).MustBigInt()
			b.Sub(b, convert.StrTo(price).MustBigInt())
			balance = b.String()
		} else {
			price, balance = redpacketRand(redpacket.Balance, redpacket.Total-redpacket.ClaimCount)
		}
		// records
		rr.RedpacketId = info.Id
		rr.Address = info.Address
		rr.Amount = price
		err = rr.Create(ctx, conf.MustMongoDB())
		if err != nil {
			return err
		}
		err = redpacket.FindAndUpdate(ctx, conf.MustMongoDB(), bson.M{
			"$set": bson.M{"balance": balance, "claim_count": redpacket.ClaimCount + 1},
		})
		if err != nil {
			return err
		}
		// pay
		redpacket.TxID, err = point.Pay(ctx, pointSystem.PayRequest{
			FromObject: conf.ExternalAppSetting.RedpacketAddress,
			ToSubject:  info.Address,
			Amount:     rr.Amount,
			Comment:    "",
			Channel:    "claim_redpacket",
			ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=claim_redpacket&order_id=" + rr.ID.Hex(),
			BindOrder:  rr.ID.Hex(),
		})
		return err
	})
	if err != nil {
		e = errcode.ServerError.WithDetails(err.Error())
	}
	return nil
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

	err = m.FindAndUpdate(ctx, conf.MustMongoDB(), bson.M{"$set": bson.M{"is_timeout": true}})
	if err != nil {
		return err
	}
	err = model.UseTransaction(ctx, conf.MustMongoDB(), func(ctx context.Context) error {
		err = m.First(ctx, conf.MustMongoDB())
		if err != nil {
			return err
		}
		// check need refund
		if m.Total-m.ClaimCount > 0 && m.RefundTxID == "" {
			// pay
			m.RefundTxID, err = point.Pay(ctx, pointSystem.PayRequest{
				FromObject: conf.ExternalAppSetting.RedpacketAddress,
				ToSubject:  m.Address,
				Amount:     m.Balance,
				Comment:    "",
				Channel:    "refund_redpacket",
				ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=refund_redpacket&order_id=" + p.Id,
				BindOrder:  p.Id,
			})
		}
		return err
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

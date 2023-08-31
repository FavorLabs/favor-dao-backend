package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/pointSystem"
	"favor-dao-backend/pkg/psub"

	"github.com/go-redis/redis_rate/v10"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	PrefixRedisKeyRedpacket      = "redpacket_send:"
	PrefixRedisKeyRedpacketClaim = "redpacket_claim:"
	TypeRedpacketDone            = "redpacket:done"
	TypeRedpacketClaim           = "redpacket:claim"
	QueueRedpacket               = "redpacket"
)

type RedpacketRequestAuth struct {
	Auth             AuthByWalletRequest `json:"auth"     binding:"required"`
	RedpacketRequest `json:",inline"`
}

type RedpacketRequest struct {
	Type   model.RedpacketType `json:"type"`
	Title  string              `json:"title"    binding:"required"`
	Amount string              `json:"amount"   binding:"required"`
	Total  int64               `json:"total"    binding:"required"`
}

type ClaimChRequest struct {
	Id      primitive.ObjectID
	Address string
	Count   int64
}

type ClaimChResponse struct {
	Info *model.RedpacketClaim
	err  *errcode.Error
}

func CreateRedpacket(address string, parm RedpacketRequest) (id string, err error) {
	if parm.Total > conf.ExternalAppSetting.RedPacketMaxCount || parm.Total < 1 {
		err = errcode.RedpacketNumberErr
		return
	}
	amount, err := convert.StrTo(parm.Amount).BigInt()
	if err != nil {
		return
	}
	if parm.Type == model.RedpacketTypeLucked && amount.Div(amount, big.NewInt(parm.Total)).Cmp(big.NewInt(0)) < 1 {
		err = errcode.RedpacketAmountErr
		return
	}
	var (
		notify *psub.Notify
	)
	defer func() {
		if notify != nil {
			notify.Cancel()
		}
	}()

	ctx := context.TODO()
	// create
	redpacket := &model.Redpacket{
		Address: address,
		Title:   parm.Title,
		Amount:  parm.Amount,
		Type:    parm.Type,
		Total:   parm.Total,
		Balance: parm.Amount,
	}
	if parm.Type == model.RedpacketTypeAverage {
		redpacket.AvgAmount = parm.Amount
		redpacket.Amount = amount.Mul(amount, new(big.Int).SetInt64(parm.Total)).String()
		redpacket.Balance = redpacket.Amount
	}
	_, err = model.UseTransaction(ctx, conf.MustMongoDB(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		err = redpacket.Create(sessCtx, conf.MustMongoDB())
		if err != nil {
			return nil, err
		}
		id = redpacket.ID.Hex()

		// sub order
		notify, err = pubsub.NewSubscribe(id)
		if err != nil {
			return nil, err
		}
		// pay
		redpacket.TxID, err = point.Pay(sessCtx, pointSystem.PayRequest{
			FromObject: address,
			ToSubject:  conf.ExternalAppSetting.RedpacketAddress,
			Amount:     redpacket.Amount,
			Comment:    "",
			Channel:    "send_redpacket",
			ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=send_redpacket&order_id=" + id,
			BindOrder:  id,
		})
		if err != nil {
			return nil, err
		}
		return redpacket, nil
	})
	if err != nil {
		return
	}
	e := redpacket.Update(context.Background(), conf.MustMongoDB())
	if e != nil {
		logrus.Errorf("redpacket.Update order_id:%s tx_id:%s err:%s", id, redpacket.TxID, e)
		// When an error occurs, wait for the callback to fix the txID again
	}
	// wait pay notify
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case val := <-notify.Ch:
		if !val.(bool) {
			err = errors.New("create redpacket failed")
		}
	}
	return
}

func eventSendRedpacket(notify PayCallbackParam) error {
	ctx := context.Background()
	m := &model.Redpacket{}
	m.ID, _ = primitive.ObjectIDFromHex(notify.OrderId)
	err := m.First(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("send_redpacket on notify: redpacket.First _id:%s err:%s", notify.OrderId, err)
		return err
	}
	m.TxID = notify.TxID
	switch notify.TxStatus {
	case TxCompleted:
		// success
		m.PayStatus = model.PaySuccess
		err = m.Update(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("send_redpacket on notify: redpacket.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
			return err
		}
		key := PrefixRedisKeyRedpacket + notify.OrderId
		err = conf.Redis.Set(ctx, key, m.Total, 0).Err()
		if err != nil {
			logrus.Errorf("send_redpacket on notify: redis set redpacket_%s err:%s", notify.OrderId, err)
			return err
		}
		task := NewRedpacketDoneTask(notify.OrderId)
		_, er := queue.Enqueue(task, asynq.ProcessIn(conf.ExternalAppSetting.RedPacketTimeout), asynq.Queue(QueueRedpacket))
		if er != nil {
			logrus.Errorf("enqueue RedpacketDoneTask %s", er)
		}
		pubsub.Notify(notify.OrderId, true)
	case TxRollback, TxCancelled:
		// failed
		m.PayStatus = model.PayFailed
		err = m.Update(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("send_redpacket on notify: redpacket.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
			return err
		}
		pubsub.Notify(notify.OrderId, false)
	}
	return nil
}

func redpacketLucked(totalAmount string, numbers int64) []string {
	restPrice := convert.StrTo(totalAmount).MustBigInt()
	restNumbers := new(big.Int).SetInt64(numbers)
	dub := new(big.Int).SetInt64(2)

	packets := make([]string, numbers)
	for i := int64(0); i < numbers-1; i++ {
		max := new(big.Int)
		max.Div(restPrice, restNumbers).Mul(max, dub).Sub(max, new(big.Int).SetInt64(1))

		value, _ := rand.Int(rand.Reader, max)
		value.Add(value, new(big.Int).SetInt64(1))

		packets[i] = value.String()

		restPrice.Sub(restPrice, value)
		restNumbers.Sub(restNumbers, new(big.Int).SetInt64(1))
	}

	packets[numbers-1] = restPrice.String()

	return packets
}

func redpacketRand(balance string, count int64) (amount, nowBalance string) {
	// if count <= 1 {
	//	// 	return balance, "0"
	//	// }
	//	// restPrice := convert.StrTo(balance).MustBigInt()
	//	// restNumbers := new(big.Int).SetInt64(count)
	//	// max := new(big.Int)
	//	// dub := new(big.Int).SetInt64(2)
	//	//
	//	// max.Div(restPrice, restNumbers).Mul(max, dub).Sub(max, new(big.Int).SetInt64(1))
	//	//
	//	// value, _ := rand.Int(rand.Reader, max)
	//	// value.Add(value, new(big.Int).SetInt64(1))
	//	//
	//	// restPrice.Sub(restPrice, value)
	//	//
	//	// return value.String(), restPrice.String()

	if count == 1 {
		return balance, "0"
	}

	max := new(big.Int)
	min := new(big.Int).SetInt64(1)
	dub := new(big.Int).SetInt64(2)

	restPrice := convert.StrTo(balance).MustBigInt()
	restNumbers := new(big.Int).SetInt64(count)

	max.Div(restPrice, restNumbers).Mul(max, dub)
	if max.Cmp(min) != 1 {
		max.Set(min)
	}

	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return
	}

	restNumbers.Sub(restNumbers, min)
	restPrice.Sub(restPrice, value)

	return value.String(), restPrice.String()
}

func redpacketAverage(price string, numbers int64) (packets []string) {
	packets = make([]string, numbers)
	for k := range packets {
		packets[k] = price
	}
	return
}

func ClaimRedpacket(ctx context.Context, address string, redpacketID primitive.ObjectID) (rr *model.RedpacketClaim, e *errcode.Error) {
	claimKey := fmt.Sprintf("%s%s:%s", PrefixRedisKeyRedpacketClaim, redpacketID.Hex(), address)
	res, err := limiter.Allow(ctx, claimKey, redis_rate.PerMinute(3))
	if err != nil || res.Allowed == 0 {
		return nil, errcode.TooManyRequests
	}

	key := PrefixRedisKeyRedpacket + redpacketID.Hex()
	count, err := conf.Redis.Get(ctx, key).Int64()
	if err != nil || count <= 0 {
		return nil, errcode.RedpacketHasBeenCollectedCompletely
	}

	err = rr.FindOne(ctx, conf.MustMongoDB(), bson.M{
		"address":      address,
		"redpacket_id": redpacketID,
	})
	if err == nil {
		return nil, errcode.RedpacketAlreadyClaim
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errcode.ServerError.WithDetails(err.Error())
	}

	_, err = model.UseTransaction(ctx, conf.MustMongoDB(), func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Calculation amount
		redpacket := &model.Redpacket{}
		redpacket.ID = redpacketID
		err = redpacket.First(sessCtx, conf.MustMongoDB())
		if err != nil {
			return nil, err
		}
		if redpacket.IsTimeout {
			return nil, errcode.RedpacketTimeout
		}
		if redpacket.ClaimCount == redpacket.Total {
			return nil, errcode.RedpacketHasBeenCollectedCompletely
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
		if price == "" {
			return nil, errcode.ServerError
		}
		// records
		rr = &model.RedpacketClaim{}
		rr.RedpacketId = redpacketID
		rr.Address = address
		rr.Amount = price
		err = rr.Create(sessCtx, conf.MustMongoDB())
		if err != nil {
			return nil, err
		}
		if rr.ID.IsZero() {
			return nil, errcode.RedpacketAlreadyClaim
		}
		err = redpacket.FindAndUpdate(sessCtx, conf.MustMongoDB(), bson.M{
			"$set": bson.M{"balance": balance, "claim_count": redpacket.ClaimCount + 1},
		})
		if err != nil {
			return nil, err
		}
		// pay
		redpacket.TxID, err = point.Pay(sessCtx, pointSystem.PayRequest{
			FromObject: conf.ExternalAppSetting.RedpacketAddress,
			ToSubject:  address,
			Amount:     rr.Amount,
			Comment:    "",
			Channel:    "claim_redpacket",
			ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=claim_redpacket&order_id=" + rr.ID.Hex(),
			BindOrder:  rr.ID.Hex(),
		})
		if err != nil {
			return nil, err
		}
		return redpacket, nil
	})
	if err != nil {
		var ee *errcode.Error
		if errors.As(err, &ee) {
			return nil, ee
		}

		logrus.Errorf("claim redpacket: %v", err)
		return nil, errcode.ServerError.WithDetails(err.Error())
	}

	conf.Redis.Decr(ctx, key)

	return rr, nil
}

func eventClaimRedpacket(notify PayCallbackParam) error {
	ctx := context.Background()
	m := &model.RedpacketClaim{}
	m.ID, _ = primitive.ObjectIDFromHex(notify.OrderId)
	err := m.First(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("claim_redpacket on notify: RedpacketRecords.First _id:%s err:%s", notify.OrderId, err)
		return err
	}
	m.TxID = notify.TxID
	switch notify.TxStatus {
	case TxCompleted:
		// success
		m.PayStatus = model.PaySuccess
	case TxRollback, TxCancelled:
		// failed
		m.PayStatus = model.PayFailed
	default:
		return nil
	}
	err = m.Update(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("claim_redpacket on notify: RedpacketRecords.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
		return err
	}
	return nil
}

func RedpacketInfo(ctx context.Context, redpacketID primitive.ObjectID, address string) (info *model.RedpacketViewFormatted, err error) {
	info = &model.RedpacketViewFormatted{}
	rd := model.Redpacket{}
	rd.ID = redpacketID
	err = rd.First(ctx, conf.MustMongoDB())
	if err != nil {
		return
	}
	info.Redpacket = rd
	user, err := ds.GetUserByAddress(rd.Address)
	if err != nil {
		return
	}
	info.UserAvatar = user.Avatar
	info.UserNickname = user.Nickname
	record := model.RedpacketClaim{}
	_ = record.FindOne(ctx, conf.MustMongoDB(), bson.M{"address": address, "redpacket_id": redpacketID})
	info.ClaimAmount = record.Amount
	return info, nil
}

type RedpacketStatsResponse struct {
	TotalAmount string `json:"total_amount"`
}

func RedpacketSendStats(ctx context.Context, req RedpacketQueryParam) RedpacketStatsResponse {
	filter := bson.M{"address": req.Address}
	filter["created_on"] = bson.M{
		"$gte": req.StartTime,
		"$lt":  req.EndTime,
	}
	rrd := model.Redpacket{}
	amount := rrd.CountAmount(ctx, conf.MustMongoDB(), filter)
	return RedpacketStatsResponse{TotalAmount: amount}
}

func RedpacketClaimStats(ctx context.Context, req RedpacketQueryParam) RedpacketStatsResponse {
	filter := bson.M{"address": req.Address}
	filter["created_on"] = bson.M{
		"$gte": req.StartTime,
		"$lt":  req.EndTime,
	}
	rrd := model.RedpacketClaim{}
	amount := rrd.CountAmount(ctx, conf.MustMongoDB(), filter)
	return RedpacketStatsResponse{TotalAmount: amount}
}

type RedpacketQueryParam struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Address   string `json:"address"`
}

func RedpacketSendList(ctx context.Context, req RedpacketQueryParam, limit, offset int) (total int64, out []*model.RedpacketSendFormatted) {
	filter := bson.M{"address": req.Address}
	filter["created_on"] = bson.M{
		"$gte": req.StartTime,
		"$lt":  req.EndTime,
	}
	rrd := model.Redpacket{}
	total = rrd.Count(ctx, conf.MustMongoDB(), filter)
	out = rrd.FindList(ctx, conf.MustMongoDB(), filter, limit, offset)
	return
}

func RedpacketClaimList(ctx context.Context, rid primitive.ObjectID, limit, offset int) (total int64, out []*model.RedpacketClaimFormatted) {
	filter := bson.M{}
	filter["redpacket_id"] = rid
	rrd := model.RedpacketClaim{}
	total = rrd.Count(ctx, conf.MustMongoDB(), filter)
	out = rrd.FindList(ctx, conf.MustMongoDB(), filter, limit, offset)
	return
}

func RedpacketClaimListForMy(ctx context.Context, req RedpacketQueryParam, limit, offset int) (total int64, out []*model.RedpacketClaimFormatted) {
	filter := bson.M{}
	filter["created_on"] = bson.M{
		"$gte": req.StartTime,
		"$lt":  req.EndTime,
	}
	filter["address"] = req.Address
	rrd := model.RedpacketClaim{}
	total = rrd.Count(ctx, conf.MustMongoDB(), filter)
	out = rrd.FindListForMy(ctx, conf.MustMongoDB(), filter, limit, offset)
	return
}

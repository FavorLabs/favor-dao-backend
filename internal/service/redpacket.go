package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/pointSystem"
	"favor-dao-backend/pkg/psub"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RedpacketRequest struct {
	Auth   AuthByWalletRequest `json:"auth"     binding:"required"`
	Type   model.RedpacketType `json:"type"     binding:"required"`
	Title  string              `json:"title"    binding:"required"`
	Amount string              `json:"amount"   binding:"required"`
	Total  int64               `json:"total"    binding:"required"`
}

func CreateRedpacket(address string, parm RedpacketRequest) (id string, err error) {
	amount, err := convert.StrTo(parm.Amount).BigInt()
	if err != nil {
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
	err = model.UseTransaction(ctx, conf.MustMongoDB(), func(ctx context.Context) error {
		err = redpacket.Create(ctx, conf.MustMongoDB())
		if err != nil {
			return err
		}
		id = redpacket.ID.Hex()

		// sub order
		notify, err = pubsub.NewSubscribe(id)
		if err != nil {
			return err
		}
		// pay
		redpacket.TxID, err = point.Pay(ctx, pointSystem.PayRequest{
			FromObject: address,
			ToSubject:  conf.ExternalAppSetting.RedpacketAddress,
			Amount:     redpacket.Amount,
			Comment:    "",
			Channel:    "send_redpacket",
			ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=send_redpacket&order_id=" + id,
			BindOrder:  id,
		})
		return err
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
	err := m.First(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("send_redpacket on notify: redpacket.First _id:%s err:%s", notify.OrderId, err)
		return err
	}
	m.TxID = notify.TxID
	upFun := func() error {
		err = m.Update(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("send_redpacket on notify: redpacket.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
			return err
		}
		return nil
	}
	switch notify.TxStatus {
	case TxCompleted:
		// success
		m.PayStatus = model.PaySuccess
		err = upFun()
		if err != nil {
			return err
		}
		key := "redpacket_" + notify.OrderId
		// var packets []string
		// switch m.Type {
		// case model.Lucked:
		// 	packets = redpacketLucked(m.TotalAmount, m.Total)
		// case model.Average:
		// 	packets = redpacketAverage(m.Amount, m.Total)
		// }
		// err = conf.Redis.RPush(ctx, key, packets).Err()
		err = conf.Redis.Set(ctx, key, m.Total, 0).Err()
		if err != nil {
			logrus.Errorf("send_redpacket on notify: redis set redpacket_%s err:%s", notify.OrderId, err)
			return err
		}
		pubsub.Notify(notify.OrderId, true)
	case TxRollback, TxCancelled:
		// failed
		m.PayStatus = model.PayFailed
		err = upFun()
		if err != nil {
			return err
		}
		pubsub.Notify(notify.OrderId, false)
	default:
		return nil
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
	if count == 1 {
		return balance, "0"
	}
	restPrice := convert.StrTo(balance).MustBigInt()
	restNumbers := new(big.Int).SetInt64(count)
	max := new(big.Int)
	dub := new(big.Int).SetInt64(2)

	max.Div(restPrice, restNumbers).Mul(max, dub).Sub(max, new(big.Int).SetInt64(1))

	value, _ := rand.Int(rand.Reader, max)
	value.Add(value, new(big.Int).SetInt64(1))

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
	key := "redpacket_" + redpacketID.Hex()
	// price, err := conf.Redis.RPop(ctx, key).Result()
	// if err != nil {
	// 	return nil, errcode.RedpacketHasBeenCollectedCompletely
	// }
	// defer func() {
	// 	if e != nil {
	// 		conf.Redis.RPush(ctx, key, price)
	// 	}
	// }()
	err := conf.Redis.Exists(ctx, key).Err()
	if err != nil {
		return nil, errcode.RedpacketHasBeenCollectedCompletely
	}
	count := conf.Redis.Decr(ctx, key).Val()
	if count < 0 {
		return nil, errcode.RedpacketHasBeenCollectedCompletely
	}
	defer func() {
		if e != nil {
			// todo Ensuring Success
			conf.Redis.Incr(ctx, key)
		}
	}()

	err = model.UseTransaction(ctx, conf.MustMongoDB(), func(ctx context.Context) error {
		// Calculation amount
		redpacket := &model.Redpacket{}
		redpacket.ID = redpacketID
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
			price, balance = redpacketRand(redpacket.Balance, count+1)
		}
		// records
		rr = &model.RedpacketClaim{}
		rr.RedpacketId = redpacketID
		rr.Address = address
		rr.Amount = price
		err = rr.Create(ctx, conf.MustMongoDB())
		if err != nil {
			return err
		}
		err = redpacket.FindAndUpdate(ctx, conf.MustMongoDB(), bson.M{
			"$set": bson.M{"balance": balance, "claim_count": redpacket.ClaimCount - 1},
		})
		if err != nil {
			return err
		}
		// pay
		redpacket.TxID, err = point.Pay(ctx, pointSystem.PayRequest{
			FromObject: conf.ExternalAppSetting.RedpacketAddress,
			ToSubject:  address,
			Amount:     rr.Amount,
			Comment:    "",
			Channel:    "claim_redpacket",
			ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=claim_redpacket&order_id=" + rr.ID.Hex(),
			BindOrder:  rr.ID.Hex(),
		})
		return err
	})
	if err != nil {
		return rr, errcode.ServerError.WithDetails(err.Error())
	}
	return rr, nil
}

func eventClaimRedpacket(notify PayCallbackParam) error {
	ctx := context.Background()
	m := &model.RedpacketClaim{}
	err := m.First(ctx, conf.MustMongoDB())
	if err != nil {
		logrus.Errorf("claim_redpacket on notify: RedpacketRecords.First _id:%s err:%s", notify.OrderId, err)
		return err
	}
	m.TxID = notify.TxID
	upFun := func() error {
		err = m.Update(ctx, conf.MustMongoDB())
		if err != nil {
			logrus.Errorf("claim_redpacket on notify: RedpacketRecords.Update tx_status:%s tx_id:%s _id:%s err:%s", notify.TxStatus, notify.TxID, notify.OrderId, err)
			return err
		}
		return nil
	}
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
	return upFun()
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
	record := model.RedpacketClaim{}
	_ = record.FindOne(ctx, conf.MustMongoDB(), bson.M{"address": address, "redpacket_id": redpacketID.Hex()})
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
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	RedpacketID string `json:"redpacket_id"`
	Address     string `json:"address"`
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

func RedpacketClaimList(ctx context.Context, req RedpacketQueryParam, limit, offset int) (total int64, out []*model.RedpacketClaimFormatted) {
	filter := bson.M{}
	if req.RedpacketID != "" {
		filter["redpacket_id"] = req.RedpacketID
	} else {
		filter["created_on"] = bson.M{
			"$gte": req.StartTime,
			"$lt":  req.EndTime,
		}
		filter["address"] = req.Address
	}
	rrd := model.RedpacketClaim{}
	total = rrd.Count(ctx, conf.MustMongoDB(), filter)
	out = rrd.FindList(ctx, conf.MustMongoDB(), filter, limit, offset)
	return
}

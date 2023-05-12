package service

import (
	"context"
	"errors"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/pkg/pointSystem"
	"github.com/sirupsen/logrus"
)

type PayCallbackParam struct {
	OrderId     string `form:"order_id"`
	Method      string `form:"method"`
	TxID        string `form:"tx_id"`
	TxStatus    string `form:"tx_status"`
	TxTimestamp string `form:"tx_timestamp"`
}

const (
	TxPending    = "pending"
	TxInProgress = "in_progress"
	TxCompleted  = "completed"
	TxFailed     = "failed"
	TxCancelled  = "cancelled"
	TxRollback   = "rollback"
	TxCrashed    = "crashed"
)

func PayNotify(notify PayCallbackParam) (err error) {
	logrus.Infof("PayNotify method:%s order_id:%s tx_id:%s tx_status:%s", notify.Method, notify.OrderId, notify.TxID, notify.TxStatus)
	switch notify.Method {
	case "sub_dao":
		return eventSubDAO(notify)
	case "send_redpacket":
		return eventSendRedpacket(notify)
	case "claim_redpacket":
		return eventClaimRedpacket(notify)
	default:
		return errors.New("unknown method")
	}

}

func eventSubDAO(notify PayCallbackParam) error {
	var subStatus core.DaoSubscribeT
	switch notify.TxStatus {
	case TxCompleted:
		// success
		subStatus = core.DaoSubscribeSuccess
	case TxRollback, TxCancelled:
		// failed
		subStatus = core.DaoSubscribeFailed
	default:
		return nil
	}
	err := UpdateSubscribeDAO(notify.OrderId, notify.TxID, subStatus)
	if err != nil {
		return err
	}
	pubsub.Notify(notify.OrderId, subStatus)
	return nil
}

func FindAccounts(ctx context.Context, uid string) (accounts []pointSystem.Account, err error) {
	return point.FindAccounts(ctx, uid)
}

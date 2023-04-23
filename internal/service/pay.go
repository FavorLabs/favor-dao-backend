package service

import (
	"errors"

	"favor-dao-backend/internal/core"
	"github.com/sirupsen/logrus"
)

type PayCallbackParam struct {
	OrderId     string `json:"order_id"`
	Method      string `json:"method"`
	TxID        string `json:"tx_id"`
	TxStatus    string `json:"tx_status"`
	TxTimestamp string `json:"tx_timestamp"`
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

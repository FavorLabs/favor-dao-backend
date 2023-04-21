package service

import "sync"

type PayStatus int

const (
	PAYING PayStatus = iota
)

type Pay struct {
	// order_id: chan
	subPub sync.Map
}

type Notify struct {
	// todo
	orderId string
	status  PayStatus
}

func New() *Pay {
	return &Pay{}
}

func PayNotify(notify Notify) {
	pay.Pub(notify)
}

func (p *Pay) Pub(notify Notify) {
	c, ok := p.subPub.Load(notify.orderId)
	if !ok {
		return
	}
	ch := c.(chan Notify)
	ch <- notify
}

func (p *Pay) Sub(orderId string) *Notify {
	ch := make(chan Notify, 1)
	p.subPub.Store(orderId, ch)
	defer p.subPub.Delete(orderId)
	defer close(ch)
	for {
		n := <-ch
		return &n
	}
}

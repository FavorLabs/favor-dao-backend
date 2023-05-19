package psub

import (
	"errors"
	"sync"
)

type Service struct {
	subPub *sync.Map
}

type Notify struct {
	Key    string
	subPub *sync.Map
	Ch     <-chan interface{}
}

func New() *Service {
	return &Service{
		subPub: &sync.Map{},
	}
}

func (p *Service) Notify(key string, notify interface{}) {
	c, ok := p.subPub.Load(key)
	if !ok {
		return
	}
	ch := c.(chan interface{})
	select {
	case ch <- notify:
	default:
	}
}

func (p *Service) NewSubscribe(key string) (*Notify, error) {
	notify, ok := p.subPub.Load(key)
	if ok {
		return notify.(*Notify), errors.New("key already exists")
	}
	ch := make(chan interface{}, 1)
	p.subPub.Store(key, ch)
	return &Notify{
		Key:    key,
		subPub: p.subPub,
		Ch:     ch,
	}, nil
}

func (s *Notify) Cancel() {
	val, ok := s.subPub.LoadAndDelete(s.Key)
	if ok {
		close(val.(chan interface{}))
	}
}

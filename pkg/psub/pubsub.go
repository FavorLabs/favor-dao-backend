package psub

import (
	"errors"
	"sync"
)

var ErrKeyAlreadyExists = errors.New("key already exists")

type Service struct {
	subPub *sync.Map
}

type Notify struct {
	Key    string
	subPub *sync.Map
	Ch     chan interface{}
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
	obj := c.(*Notify)
	select {
	case obj.Ch <- notify:
	default:
	}
}

func (p *Service) NewSubscribe(key string) (*Notify, error) {
	v, ok := p.subPub.Load(key)
	if ok {
		return v.(*Notify), ErrKeyAlreadyExists
	}
	notify := &Notify{
		Key:    key,
		subPub: p.subPub,
		Ch:     make(chan interface{}, 1),
	}
	p.subPub.Store(key, notify)
	return notify, nil
}

func (s *Notify) Cancel() {
	val, ok := s.subPub.LoadAndDelete(s.Key)
	if ok {
		close(val.(*Notify).Ch)
	}
}

package marketdata

import (
	"context"
	"sync"
)

type Subscription struct {
	ch     <-chan Data
	cancel context.CancelFunc
	once   sync.Once
	done   chan struct{}
}

func NewSubscription(ch <-chan Data, cancel context.CancelFunc) *Subscription {
	return &Subscription{
		ch:     ch,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func (s *Subscription) Channel() <-chan Data {
	return s.ch
}

func (s *Subscription) Close() {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		close(s.done)
	})
}

func (s *Subscription) Done() <-chan struct{} {
	return s.done
}

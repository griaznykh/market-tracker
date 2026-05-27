package broker

import (
	"market-service/internal/marketdata"
	"sync"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan marketdata.Data
	closed      bool
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan marketdata.Data),
	}
}

func (sb *Broker) Subscribe(topic string) <-chan marketdata.Data {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.closed {
		return nil
	}

	ch := make(chan marketdata.Data, 10)
	sb.subscribers[topic] = append(sb.subscribers[topic], ch)

	return ch
}

func (sb *Broker) Unsubscribe(topic string, ch <-chan marketdata.Data) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	subscribers, exists := sb.subscribers[topic]
	if !exists {
		return
	}

	for i, sub := range subscribers {
		if sub == ch {
			sb.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
			close(sub)
			break
		}
	}

	if len(sb.subscribers[topic]) == 0 {
		delete(sb.subscribers, topic)
	}
}

func (ps *Broker) Publish(topic string, msg marketdata.Data) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.closed {
		return
	}

	for _, ch := range ps.subscribers[topic] {
		go func(c chan marketdata.Data) {
			c <- msg
		}(ch)
	}
}

func (ps *Broker) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return
	}
	ps.closed = true

	for topic, subs := range ps.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		delete(ps.subscribers, topic)
	}
}

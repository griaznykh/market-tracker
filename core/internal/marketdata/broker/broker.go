package broker

import (
	"fmt"
	"market-service/internal/marketdata"
	"sync"
)

type (
	EventType int

	SubscriptionEvent struct {
		Event    EventType
		Provider string
		Ticker   string
	}

	Broker struct {
		mu          sync.RWMutex
		subscribers map[string][]chan marketdata.Data
		event       chan SubscriptionEvent
		closed      bool
	}
)

const (
	Unsubscribe EventType = 0
	Subscribe   EventType = 1
)

func NewBroker() *Broker {
	return &Broker{
		event:       make(chan SubscriptionEvent, 10),
		subscribers: make(map[string][]chan marketdata.Data),
	}
}

func (sb *Broker) Subscribe(provider string, ticker string) <-chan marketdata.Data {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.closed {
		return nil
	}

	topic := fmt.Sprintf("%s:%s", provider, ticker)
	ch := make(chan marketdata.Data, 10)
	sb.subscribers[topic] = append(sb.subscribers[topic], ch)

	if len(sb.subscribers[topic]) == 1 {
		sb.event <- SubscriptionEvent{
			Subscribe,
			provider,
			ticker,
		}
	}

	return ch
}

func (sb *Broker) Unsubscribe(provider string, ticker string, ch <-chan marketdata.Data) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	topic := fmt.Sprintf("%s:%s", provider, ticker)
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

		sb.event <- SubscriptionEvent{
			Unsubscribe,
			provider,
			ticker,
		}
	}
}

func (sb *Broker) Publish(msg marketdata.Data) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if sb.closed {
		return
	}

	topic := fmt.Sprintf("%s:%s", msg.Provider, msg.Ticker)
	for _, ch := range sb.subscribers[topic] {
		go func(c chan marketdata.Data) {
			c <- msg
		}(ch)
	}
}

func (sb *Broker) Event() <-chan SubscriptionEvent {
	return sb.event
}

func (sb *Broker) Close() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.closed {
		return
	}
	sb.closed = true

	for topic, subs := range sb.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		delete(sb.subscribers, topic)
	}

	close(sb.event)
}

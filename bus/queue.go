package bus

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type MessageBus struct {
	inbound  chan *InboundMessage
	outbound chan *OutboundMessage
	sync.RWMutex
	outboundSubscribers map[string][]func(*OutboundMessage) error
	running             bool
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:             make(chan *InboundMessage, 100),
		outbound:            make(chan *OutboundMessage, 100),
		outboundSubscribers: make(map[string][]func(*OutboundMessage) error),
		running:             false,
	}
}

func (mb *MessageBus) PublishInbound(msg *InboundMessage) {
	mb.inbound <- msg
}

func (mb *MessageBus) ConsumeInbound(ctx context.Context) (*InboundMessage, error) {
	select {
	case msg := <-mb.inbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (mb *MessageBus) PublishOutbound(msg *OutboundMessage) {
	mb.outbound <- msg
}

func (mb *MessageBus) ConsumeOutbound(ctx context.Context) (*OutboundMessage, error) {
	select {
	case msg := <-mb.outbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (mb *MessageBus) SubscribeOutbound(channel string, callback func(*OutboundMessage) error) {
	mb.Lock()
	defer mb.Unlock()

	mb.outboundSubscribers[channel] = append(mb.outboundSubscribers[channel], callback)
}

func (mb *MessageBus) DispatchOutbound(ctx context.Context) error {
	mb.Lock()
	mb.running = true
	mb.Unlock()

	logger := logrus.WithField("component", "message_bus")
	logger.Info("Outbound dispatcher started")

	for mb.running {
		select {
		case msg := <-mb.outbound:
			mb.RLock()
			subscribers := mb.outboundSubscribers[msg.Channel]
			mb.RUnlock()

			for _, callback := range subscribers {
				if err := callback(msg); err != nil {
					logger.WithError(err).Errorf("Error dispatching to %s", msg.Channel)
				}
			}
		case <-time.After(1 * time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (mb *MessageBus) Stop() {
	mb.Lock()
	defer mb.Unlock()
	mb.running = false
}

func (mb *MessageBus) InboundSize() int {
	return len(mb.inbound)
}

func (mb *MessageBus) OutboundSize() int {
	return len(mb.outbound)
}

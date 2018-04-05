package messages

import (
	"context"
	"log"

	nats "github.com/nats-io/go-nats"
)

type OnMessage func(msg []byte) error

type Dispatcher interface {
	Publish(topic string, message []byte) error
	Subscribe(ctx context.Context, topic string, callback OnMessage) error
}

type Nats struct {
	conn *nats.Conn
}

func NewNatsDispatcher(c *nats.Conn) Dispatcher {
	return &Nats{c}
}

func (d Nats) Publish(topic string, message []byte) error {
	return d.conn.Publish(topic, message)
}

func (d Nats) Subscribe(ctx context.Context, topic string, callback OnMessage) error {
	sub, err := d.conn.Subscribe(topic, func(msg *nats.Msg) {
		err := callback(msg.Data)
		if err != nil {
			log.Println("[NatsDispatcher] callback error:", err)
		}
	})
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()
	return err
}

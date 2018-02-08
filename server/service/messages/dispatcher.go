package messages

import nats "github.com/nats-io/go-nats"

type OnMessage func(msg []byte) error

type Dispatcher interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string, callback OnMessage) error
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

func (d Nats) Subscribe(topic string, callback OnMessage) error {
	_, err := d.conn.Subscribe(topic, func(msg *nats.Msg) {
		callback(msg.Data)
	})
	return err
}

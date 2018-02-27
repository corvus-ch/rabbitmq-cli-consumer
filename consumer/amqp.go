package consumer

import (
	"github.com/streadway/amqp"
	"io"
)

// Connection describes the part of amqp.Connection required by this code base.
type Connection interface {
	io.Closer
	Channel() (*amqp.Channel, error)
}

// Channel describes the part of amqp.Channel required by this code base.
type Channel interface {
	io.Closer
	Cancel(consumer string, noWait bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
}

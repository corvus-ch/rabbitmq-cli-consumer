package consumer_test

import (
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
)

type TestConnection struct {
	consumer.Connection
	mock.Mock
}

func (t *TestConnection) Close() error {
	argsT := t.Called()

	return argsT.Error(0)
}

func (t *TestConnection) Channel() (*amqp.Channel, error) {
	argsT := t.Called()

	return argsT.Get(0).(*amqp.Channel), argsT.Error(1)
}

type TestChannel struct {
	consumer.Channel
	mock.Mock
	notifyClose chan *amqp.Error
}

func (t *TestChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	argsT := t.Called(name, kind, durable, autoDelete, internal, noWait, args)

	return argsT.Error(0)
}

func (t *TestChannel) NotifyClose(c chan *amqp.Error) chan *amqp.Error {
	t.notifyClose = c
	return c
}

func (t *TestChannel) TriggerNotifyClose(reason string) bool {
	if t.notifyClose != nil {
		t.notifyClose <- &amqp.Error{
			Reason: reason,
			Code:   320,
		}
		return true
	}
	return false
}

func (t *TestChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	argsT := t.Called(name, durable, autoDelete, exclusive, noWait, args)

	return argsT.Get(0).(amqp.Queue), argsT.Error(1)
}

func (t *TestChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	argsT := t.Called(prefetchCount, prefetchSize, global)

	return argsT.Error(0)
}

func (t *TestChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	argsT := t.Called(name, key, exchange, noWait, args)

	return argsT.Error(0)
}

func (t *TestChannel) Close() error {
	argsT := t.Called()

	return argsT.Error(0)
}

func (t *TestChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	argsT := t.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)

	d, ok := argsT.Get(0).(chan amqp.Delivery)
	if !ok {
		d = nil
	}

	return d, argsT.Error(1)
}

func (t *TestChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	argsT := t.Called(exchange, key, mandatory, immediate, msg)

	return argsT.Error(0)
}
func (t *TestChannel) Cancel(consumer string, noWait bool) error {
	argsT := t.Called(consumer, noWait)

	return argsT.Error(0)
}

type TestAmqpAcknowledger struct {
	amqp.Acknowledger
	mock.Mock
}

func (a *TestAmqpAcknowledger) Ack(tag uint64, multiple bool) error {
	argsT := a.Called(tag, multiple)

	return argsT.Error(0)
}

func (a *TestAmqpAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	argsT := a.Called(tag, multiple, requeue)

	return argsT.Error(0)
}

func (a *TestAmqpAcknowledger) Reject(tag uint64, requeue bool) error {
	argsT := a.Called(tag, requeue)

	return argsT.Error(0)
}

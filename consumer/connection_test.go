package consumer

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const defaultConfig = `[rabbitmq]
  host=localhost
  username=ricbra
  password=t3st
  vhost=staging
  port=123
  queue=worker

  [prefetch]
  count=3
  global=On

  [queuesettings]
  routingkey=foo

  [exchange]
  name=worker
  autodelete=Off
  type=test
  durable=On

  [logs]
  error=a
  info=b`

const ttlConfig = `[rabbitmq]
  host=localhost
  username=ricbra
  password=t3st
  vhost=staging
  port=123
  queue=worker

  [prefetch]
  count=3
  global=On

  [queuesettings]
  routingkey=foo
  messagettl=1200

  [exchange]
  name=worker
  autodelete=Off
  type=test
  durable=On

  [logs]
  error=a
  info=b`

var amqpTable amqp.Table

var queueTests = []struct {
	config string
	setup  func(*TestChannel)
	err    error
}{
	// The happy path.
	{
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "worker", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "worker", "test", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "worker", "foo", "worker", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with TTL.
	{
		ttlConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "worker", true, false, false, false, amqp.Table{"x-message-ttl": int32(1200)}).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "worker", "test", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "worker", "foo", "worker", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Set QoS fails.
	{
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, true).Return(fmt.Errorf("QoS error")).Once()
		},
		fmt.Errorf("failed to set QoS: QoS error"),
	},
	// Declare queue fails.
	{
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "worker", true, false, false, false, amqpTable).Return(amqp.Queue{}, fmt.Errorf("queue error")).Once()
		},
		fmt.Errorf("failed to declare queue: queue error"),
	},
	// Bind queue fails.
	{
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "worker", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "worker", "test", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "worker", "foo", "worker", false, amqpTable).Return(fmt.Errorf("queue bind error")).Once()
		},
		fmt.Errorf("failed to bind queue to exchange: queue bind error"),
	},
}

func TestQueueSettings(t *testing.T) {
	for _, test := range queueTests {
		cfg, err := config.CreateFromString(test.config)
		if err != nil {
			t.Errorf("failed to create config: %v", err)
		}

		var b bytes.Buffer
		infLogger := log.New(&b, "", 0)
		errLogger := log.New(&b, "", 0)

		ch := new(TestChannel)

		test.setup(ch)

		conn := &rabbitMqConnection{
			cfg: cfg,
			ch:  ch,
			outLog: infLogger,
			errLog: errLogger,
		}

		assert.Equal(t, test.err, conn.Setup())
		ch.AssertExpectations(t)
	}
}

type TestChannel struct {
	mock.Mock
}

func (t *TestChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	argsT := t.Called(name, kind, durable, autoDelete, internal, noWait, args)

	return argsT.Error(0)
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

	return argsT.Get(0).(<-chan amqp.Delivery), argsT.Error(0)
}

func (t *TestChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	argsT := t.Called(exchange, key, mandatory, immediate, msg)

	return argsT.Error(0)
}

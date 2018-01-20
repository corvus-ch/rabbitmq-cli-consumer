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
  queue=defaultQueue

  [queuesettings]
  routingkey=defaultRouting

  [exchange]
  name=defaultExchange
  autodelete=Off
  type=test
  durable=On
`

const qosConfig = `[rabbitmq]
  queue = qosQueue

  [prefetch]
  count = 42
  global = On
`

const ttlConfig = `[rabbitmq]
  queue = ttlQueue

  [queuesettings]
  messagettl=1200
`

const priorityConfig = `[rabbitmq]
  queue = priorityWorker

  [queuesettings]
  priority=42

  [exchange]
  name = priorityExchange
  type = priorityType
`

const multipleRoutingKeysConfig = `[rabbitmq]
  queue = multiRoutingQueue

  [queuesettings]
  routingkey=foo
  routingkey=bar

  [exchange]
  name = multiRoutingExchange
  type = multiRoutingType
`

const oneEmptyRoutingKeyConfig = `[rabbitmq]
  queue = emptyRoutingQueue

  [queuesettings]
  routingkey="<empty>"

  [exchange]
  name = emptyRoutingExchange
  type = emptyRoutingType
`

const noRoutingKeyConfig = `[rabbitmq]
  queue = noRoutingQueue

  [exchange]
  name = noRoutingExchange
  type = noRoutingType
`

var amqpTable amqp.Table

var queueTests = []struct {
	name   string
	config string
	setup  func(*TestChannel)
	err    error
}{
	// The happy path.
	{
		"happyPath",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "defaultExchange", "test", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "defaultQueue", "defaultRouting", "defaultExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with TTL.
	{
		"queueWithTTL",
		ttlConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "ttlQueue", true, false, false, false, amqp.Table{"x-message-ttl": int32(1200)}).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Define queue with Priority.
	{
		"queueWithPriority",
		priorityConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "priorityWorker", true, false, false, false, amqp.Table{"x-max-priority": int32(42)}).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "priorityExchange", "priorityType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "priorityWorker", "", "priorityExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with multiple routing keys.
	{
		"queueWithMultipleRoutingKeys",
		multipleRoutingKeysConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "multiRoutingQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "multiRoutingExchange", "multiRoutingType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "foo", "multiRoutingExchange", false, amqpTable).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "bar", "multiRoutingExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with one emtpy routing key.
	{
		"queueWithOneEmptyRoutingKey",
		oneEmptyRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "emptyRoutingQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "emptyRoutingExchange", "emptyRoutingType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "emptyRoutingQueue", "", "emptyRoutingExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue without routing key.
	{
		"queueWithoutRoutingKey",
		noRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "noRoutingQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "noRoutingExchange", "noRoutingType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "noRoutingQueue", "", "noRoutingExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Set QoS.
	{
		"setQos",
		qosConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 42, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "qosQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Set QoS fails.
	{
		"setQosFail",
		qosConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 42, 0, true).Return(fmt.Errorf("QoS error")).Once()
		},
		fmt.Errorf("failed to set QoS: QoS error"),
	},
	// Declare queue fails.
	{
		"declareQueueFail",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, fmt.Errorf("queue error")).Once()
		},
		fmt.Errorf("failed to declare queue: queue error"),
	},
	// Declare exchange fails.
	{
		"declareExchangeFail",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "defaultExchange", "test", true, false, false, false, amqp.Table{}).Return(fmt.Errorf("declare exchagne error")).Once()
		},
		fmt.Errorf("failed to declare exchange: declare exchagne error"),
	},
	// Bind queue fails.
	{
		"bindQueueFail",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "defaultExchange", "test", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "defaultQueue", "defaultRouting", "defaultExchange", false, amqpTable).Return(fmt.Errorf("queue bind error")).Once()
		},
		fmt.Errorf("failed to bind queue to exchange: queue bind error"),
	},
}

func TestQueueSettings(t *testing.T) {
	for _, test := range queueTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.CreateFromString(test.config)

			var b bytes.Buffer
			infLogger := log.New(&b, "", 0)
			errLogger := log.New(&b, "", 0)

			ch := new(TestChannel)

			test.setup(ch)

			conn := &rabbitMqConnection{
				cfg:    cfg,
				ch:     ch,
				outLog: infLogger,
				errLog: errLogger,
			}

			assert.Equal(t, test.err, conn.Setup())
			ch.AssertExpectations(t)
		})
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

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

const (
	autodeleteExchangeConfig  = "autodelete"
	defaultConfig             = "default"
	durableExchangeConfig     = "durable"
	multipleRoutingKeysConfig = "multiple_routing"
	noRoutingKeyConfig        = "no_routing"
	oneEmptyRoutingKeyConfig  = "empty_routing"
	priorityConfig            = "priority"
	qosConfig                 = "qos"
	routingConfig             = "routing"
	simpleExchangeConfig      = "exchange"
	ttlConfig                 = "ttl"
)

var amqpTable amqp.Table

var queueTests = []struct {
	name   string
	config string
	setup  func(*TestChannel)
	err    error
}{
	// Simple queue.
	{
		"simpleQueue",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
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
	// Declare exchange.
	{
		"declareExchange",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare durable exchange.
	{
		"declareDurableExchange",
		durableExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", true, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare auto delete exchange.
	{
		"declareAutoDeleteExchange",
		autodeleteExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, true, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare exchange fails.
	{
		"declareExchangeFail",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, amqp.Table{}).Return(fmt.Errorf("declare exchagne error")).Once()
		},
		fmt.Errorf("failed to declare exchange: declare exchagne error"),
	},
	// Bind queue.
	{
		"bindQueue",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, amqpTable).Return(nil).Once()
		},
		nil,
	},
	// Bind queue fails.
	{
		"bindQueueFail",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, amqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, amqp.Table{}).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, amqpTable).Return(fmt.Errorf("queue bind error")).Once()
		},
		fmt.Errorf("failed to bind queue to exchange: queue bind error"),
	},
}

func TestQueueSettings(t *testing.T) {
	for _, test := range queueTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.LoadAndParse(fmt.Sprintf("fixtures/%s.conf", test.config))

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

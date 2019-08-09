package consumer_test

import (
	"fmt"
	"testing"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
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
	autoDeleteQueue           = "autodelete_queue"
	durableQueue              = "durable_queue"
	nonDurableQueue           = "non_durable_queue"
	defaultQueueDurability    = "default_queue_durability"
	exclusiveQueue            = "exclusive_queue"
	noWaitQueue               = "nowait_queue"
)

var nilAmqpTable amqp.Table
var emptyAmqpTable = amqp.Table{}

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
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
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
			ch.On("ExchangeDeclare", "priorityExchange", "priorityType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "priorityWorker", "", "priorityExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with multiple routing keys.
	{
		"queueWithMultipleRoutingKeys",
		multipleRoutingKeysConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "multiRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "multiRoutingExchange", "multiRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "foo", "multiRoutingExchange", false, nilAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "bar", "multiRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with one emtpy routing key.
	{
		"queueWithOneEmptyRoutingKey",
		oneEmptyRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "emptyRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "emptyRoutingExchange", "emptyRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "emptyRoutingQueue", "", "emptyRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue without routing key.
	{
		"queueWithoutRoutingKey",
		noRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "noRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "noRoutingExchange", "noRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "noRoutingQueue", "", "noRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Set QoS.
	{
		"setQos",
		qosConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 42, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "qosQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
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
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, fmt.Errorf("queue error")).Once()
		},
		fmt.Errorf("failed to declare queue: queue error"),
	},
	// Declare exchange.
	{
		"declareExchange",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare durable exchange.
	{
		"declareDurableExchange",
		durableExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", true, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare auto delete exchange.
	{
		"declareAutoDeleteExchange",
		autodeleteExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, true, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare exchange fails.
	{
		"declareExchangeFail",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, emptyAmqpTable).Return(fmt.Errorf("declare exchagne error")).Once()
		},
		fmt.Errorf("failed to declare exchange: declare exchagne error"),
	},
	// Bind queue.
	{
		"bindQueue",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Bind queue fails.
	{
		"bindQueueFail",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, nilAmqpTable).Return(fmt.Errorf("queue bind error")).Once()
		},
		fmt.Errorf("failed to bind queue to exchange: queue bind error"),
	},
	// Durable queue
	{
		"durableQueue",
		durableQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "durableQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Non durable queue
	{
		"nonDurableQueue",
		nonDurableQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "nonDurableQueue", false, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Default queue durability
	{
		"defaultQueueDurability",
		defaultQueueDurability,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueueDurability", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// AutoDelete queue
	{
		"autoDeleteQueue",
		autoDeleteQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "autoDeleteQueue", true, true, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Exclusive queue
	{
		"exclusiveQueue",
		exclusiveQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "exclusiveQueue", true, false, true, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Nowait queue
	{
		"noWaitQueue",
		noWaitQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "noWaitQueue", true, false, false, true, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
}

func TestQueueSettings(t *testing.T) {
	for _, test := range queueTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.LoadAndParse(fmt.Sprintf("fixtures/%s.conf", test.config))
			ch := new(TestChannel)
			test.setup(ch)
			cl := &consumer.ChannelList{}
			cl.AddChannel(ch)
			assert.Equal(t, test.err, consumer.Setup(cfg, cl, log.New(0)))
			ch.AssertExpectations(t)
		})
	}
}

package consumer

import (
	"fmt"
	"io"
	"log"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/streadway/amqp"
)

// Connection describes the interface used by teh consumer to interact with the message queue.
type Connection interface {
	io.Closer

	// Setup channel according to config.
	Setup() error

	// Consume immediately starts delivering queued messages.
	Consume() (<-chan amqp.Delivery, error)

	// NotifyClose registers a listener for close events.
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

type rabbitMqConnection struct {
	Connection
	conn   *amqp.Connection
	ch     Channel
	cfg    *config.Config
	outLog *log.Logger
	errLog *log.Logger
}

// NewConnection creates a new implementing instance of the connection interface.
func NewConnection(cfg *config.Config, outLog, errLog *log.Logger) (Connection, error) {
	outLog.Println("Connecting RabbitMQ...")
	conn, err := amqp.Dial(cfg.AmqpUrl())
	if nil != err {
		return nil, fmt.Errorf("failed connecting RabbitMQ: %v", err)
	}
	outLog.Println("Connected.")

	outLog.Println("Opening channel...")
	ch, err := conn.Channel()
	if nil != err {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	outLog.Println("Done.")

	return &rabbitMqConnection{
		conn:   conn,
		ch:     ch,
		cfg:    cfg,
		outLog: outLog,
		errLog: errLog,
	}, nil
}

func (c rabbitMqConnection) Setup() error {
	if err := c.setupQoS(); err != nil {
		return err
	}

	if err := c.declareQueue(); err != nil {
		return err
	}
	// Empty Exchange name means default, no need to declare
	if c.cfg.HasExchange() {
		if err := c.declareExchange(); err != nil {
			return err
		}
	}

	return nil
}

// Consume calls consume on the amqp channel.
func (c *rabbitMqConnection) Consume() (<-chan amqp.Delivery, error) {
	return c.ch.Consume(c.cfg.RabbitMq.Queue, "", false, false, false, false, nil)
}

// NotifyClose calls NotifyClose on the amqp connection.
func (c *rabbitMqConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	return c.conn.NotifyClose(receiver)
}

func (c *rabbitMqConnection) Close() error {
	if err := c.ch.Close(); err != nil {
		return err
	}

	return c.conn.Close()
}

func (c *rabbitMqConnection) setupQoS() error {
	c.outLog.Println("Setting QoS... ")
	if err := c.ch.Qos(c.cfg.PrefetchCount(), 0, c.cfg.Prefetch.Global); err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	c.outLog.Println("Succeeded setting QoS.")
	return nil
}

func (c *rabbitMqConnection) declareQueue() error {
	c.outLog.Printf("Declaring queue \"%s\"...", c.cfg.RabbitMq.Queue)
	_, err := c.ch.QueueDeclare(c.cfg.RabbitMq.Queue, true, false, false, false, c.queueArgs())
	if nil != err {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	return nil
}

func (c *rabbitMqConnection) declareExchange() error {
	c.outLog.Printf("Declaring exchange \"%s\"...", c.cfg.Exchange.Name)
	if err := c.ch.ExchangeDeclare(
		c.cfg.Exchange.Name,
		c.cfg.ExchangeType(),
		c.cfg.Exchange.Durable,
		c.cfg.Exchange.Autodelete,
		false,
		false,
		amqp.Table{},
	); nil != err {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Bind queue
	c.outLog.Printf("Binding queue \"%s\" to exchange \"%s\"...", c.cfg.RabbitMq.Queue, c.cfg.Exchange.Name)
	for _, routingKey := range c.cfg.RoutingKeys() {
		if err := c.ch.QueueBind(
			c.cfg.RabbitMq.Queue,
			routingKey,
			c.cfg.ExchangeName(),
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to exchange: %v", err)
		}
	}

	return nil
}

func (c *rabbitMqConnection) queueArgs() amqp.Table {

	args := make(amqp.Table)

	if c.cfg.HasMessageTTL() {
		args["x-message-ttl"] = c.cfg.MessageTTL()
	}

	if c.cfg.HasDeadLetterExchange() {
		args["x-dead-letter-exchange"] = c.cfg.DeadLetterExchange()

		if c.cfg.HasDeadLetterRouting() {
			args["x-dead-letter-routing-key"] = c.cfg.DeadLetterRoutingKey()
		}
	}

	if c.cfg.HasPriority() {
		args["x-max-priority"] = c.cfg.Priority()
	}

	if len(args) > 0 {
		return args
	}

	return nil
}

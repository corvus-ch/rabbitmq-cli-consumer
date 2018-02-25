package consumer

import (
	"fmt"
	"io"
	"os"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/streadway/amqp"
	"github.com/thockin/logr"
)

type Consumer struct {
	Connection   Connection
	Builder      command.Builder
	Acknowledger Acknowledger
	Log          logr.Logger
}

// ConnectionCloseHandler calls os.Exit after the connection to RabbitMQ got closed.
func ConnectionCloseHandler(closeErr chan *amqp.Error, c *Consumer) {
	err := <-closeErr
	c.Log.Error("Connection closed: %v", err)
	os.Exit(10)
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume() error {
	c.Log.Info("Registering consumer... ")
	msgs, err := c.Connection.Consume()
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %s", err)
	}

	c.Log.Info("Succeeded registering consumer.")

	defer c.Connection.Close()

	closeErr := make(chan *amqp.Error)
	closeErr = c.Connection.NotifyClose(closeErr)

	go ConnectionCloseHandler(closeErr, c)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			c.ProcessMessage(delivery.New(d))
		}
	}()

	c.Log.Info("Waiting for messages...")
	<-forever

	return nil
}

// ProcessMessage processes a single message by running the executable.
func (c *Consumer) ProcessMessage(d delivery.Delivery) {
	cmd, err := c.Builder.GetCommand(d.Properties(), d.Info(), d.Body())
	if err != nil {
		c.Log.Errorf("failed to create command: %v", err)
		d.Nack(true, true)
		return
	}

	exitCode := cmd.Run()

	if err := c.Acknowledger.Ack(d, exitCode); err != nil {
		c.Log.Errorf("Message acknowledgement error: %v", err)
		os.Exit(11)
	}
}

// New returns a initialized consumer based on config
func New(cfg *config.Config, builder command.Builder, ack Acknowledger, l logr.Logger) (*Consumer, error) {
	conn, err := NewConnection(cfg, l)
	if err != nil {
		return nil, err
	}

	if err := conn.Setup(); err != nil {
		return nil, err
	}

	return &Consumer{
		Connection:   conn,
		Builder:      builder,
		Acknowledger: ack,
		Log:          l,
	}, nil
}

type Channel interface {
	io.Closer
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

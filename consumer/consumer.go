package consumer

import (
	"io"
	"log"
	"os"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/streadway/amqp"
)

type Consumer struct {
	Connection   Connection
	Builder      command.Builder
	Acknowledger Acknowledger
	ErrLogger    *log.Logger
	InfLogger    *log.Logger
}

// ConnectionCloseHandler calls os.Exit after the connection to RabbitMQ got closed.
func ConnectionCloseHandler(closeErr chan *amqp.Error, c *Consumer) {
	err := <-closeErr
	c.ErrLogger.Fatalf("Connection closed: %v", err)
	os.Exit(10)
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume() {
	c.InfLogger.Println("Registering consumer... ")
	msgs, err := c.Connection.Consume()
	if err != nil {
		c.ErrLogger.Fatalf("Failed to register a consumer: %s", err)
	}

	c.InfLogger.Println("Succeeded registering consumer.")

	defer c.Connection.Close()

	closeErr := make(chan *amqp.Error)
	closeErr = c.Connection.NotifyClose(closeErr)

	go ConnectionCloseHandler(closeErr, c)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			c.ProcessMessage(NewRabbitMqDelivery(d), metadata.NewProperties(d), metadata.NewDeliveryInfo(d))
		}
	}()

	c.InfLogger.Println("Waiting for messages...")
	<-forever
}

// ProcessMessage processes a single message by running the executable.
func (c *Consumer) ProcessMessage(d Delivery, p metadata.Properties, m metadata.DeliveryInfo) {
	cmd, err := c.Builder.GetCommand(p, m, d.Body())
	if err != nil {
		c.ErrLogger.Printf("failed to create command: %v", err)
		d.Nack(true, true)
		return
	}

	exitCode := cmd.Run()

	if err := c.Acknowledger.Ack(d, exitCode); err != nil {
		c.ErrLogger.Printf("Message acknowledgement error: %v", err)
		os.Exit(11)
	}
}

// New returns a initialized consumer based on config
func New(cfg *config.Config, builder command.Builder, ack Acknowledger, errLogger, infLogger *log.Logger) (*Consumer, error) {
	conn, err := NewConnection(cfg, infLogger, errLogger)
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
		ErrLogger:    errLogger,
		InfLogger:    infLogger,
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

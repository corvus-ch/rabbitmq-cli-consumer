package consumer

import (
	"fmt"
	"io"
	"os"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/corvus-ch/rabbitmq-cli-consumer/processor"
	"github.com/streadway/amqp"
	"github.com/thockin/logr"
)

type Consumer struct {
	Connection Connection
	Log        logr.Logger
}

// ConnectionCloseHandler calls os.Exit after the connection to RabbitMQ got closed.
func ConnectionCloseHandler(closeErr chan *amqp.Error, c *Consumer) {
	err := <-closeErr
	c.Log.Error("Connection closed: %v", err)
	os.Exit(10)
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume(p processor.Processor) error {
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
			err := p.Process(delivery.New(d))
			if err == nil {
				continue
			}

			switch err.(type) {
			case *processor.CreateCommandError:
				c.Log.Error(err)

			case *processor.AcknowledgmentError:
				c.Log.Error(err)
				c.Connection.Close()
				os.Exit(11)
			}
		}
	}()

	c.Log.Info("Waiting for messages...")
	<-forever

	return nil
}

// New returns a initialized consumer based on config
func New(cfg *config.Config, l logr.Logger) (*Consumer, error) {
	conn, err := NewConnection(cfg, l)
	if err != nil {
		return nil, err
	}

	if err := conn.Setup(); err != nil {
		return nil, err
	}

	return &Consumer{
		Connection: conn,
		Log:        l,
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

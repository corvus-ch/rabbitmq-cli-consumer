package consumer

import (
	"fmt"
	"os"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/corvus-ch/rabbitmq-cli-consumer/processor"
	"github.com/streadway/amqp"
	"github.com/thockin/logr"
)

type Consumer struct {
	Connection Connection
	Channel    Channel
	Queue      string
	Log        logr.Logger
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.
func New(conn Connection, ch Channel, queue string, l logr.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Queue:      queue,
		Log:        l,
	}
}

// NewFromConfig creates a new consumer instance. The setup of the amqp connection and channel is done according to the
// configuration.
func NewFromConfig(cfg *config.Config, l logr.Logger) (*Consumer, error) {
	l.Info("Connecting RabbitMQ...")
	conn, err := amqp.Dial(cfg.AmqpUrl())
	if nil != err {
		return nil, fmt.Errorf("failed connecting RabbitMQ: %v", err)
	}
	l.Info("Connected.")

	l.Info("Opening channel...")
	ch, err := conn.Channel()
	if nil != err {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	l.Info("Done.")

	Setup(cfg, ch, l)

	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Queue:      cfg.RabbitMq.Queue,
		Log:        l,
	}, nil
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
	msgs, err := c.Channel.Consume(c.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %s", err)
	}

	c.Log.Info("Succeeded registering consumer.")

	defer c.Connection.Close()

	go ConnectionCloseHandler(c.Channel.NotifyClose(make(chan *amqp.Error)), c)

	c.Log.Info("Waiting for messages...")

	for d := range msgs {
		if err := p.Process(delivery.New(d)); err != nil {
			switch err.(type) {
			case *processor.CreateCommandError:
				c.Log.Error(err)

			default:
				return err
			}
		}
	}

	return nil
}

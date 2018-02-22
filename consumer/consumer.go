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
	Tag        string
	Log        logr.Logger
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.
func New(conn Connection, ch Channel, queue, tag string, l logr.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Queue:      queue,
		Tag:        tag,
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
		Tag:        cfg.ConsumerTag(),
		Log:        l,
	}, nil
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume(p processor.Processor) error {
	c.Log.Info("Registering consumer... ")
	msgs, err := c.Channel.Consume(c.Queue, c.Tag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %s", err)
	}

	c.Log.Info("Succeeded registering consumer.")
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

// Cancel stops new messages from being consumed.
//
// All messages already received will still be processed.
func (c *Consumer) Cancel() error {
	return c.Channel.Cancel(c.Tag, false)
}

// Close tears the connection down, taking the channel with it.
func (c *Consumer) Close() error {
	if c.Connection == nil {
		return nil
	}
	return c.Connection.Close()
}

// NotifyClose registers a listener for when the connection gets closed by the server.
//
// The chan provided will be closed when the Channel is closed and on a Graceful close, no error will be sent.
func (c *Consumer) NotifyClose(receiver chan error) chan error {
	if c.Channel != nil {
		realChan := make(chan *amqp.Error)
		c.Channel.NotifyClose(realChan)

		go func() {
			for {
				err, ok := <-realChan
				if !ok {
					close(realChan)
					return
				}
				receiver <- err
			}
		}()
	}

	return receiver
}

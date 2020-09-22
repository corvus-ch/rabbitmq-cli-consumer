package consumer

import (
	"context"
	"fmt"
	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/corvus-ch/rabbitmq-cli-consumer/processor"
	"github.com/streadway/amqp"
)

type Consumer struct {
	Connection Connection
	Channel    Channel
	Queue      string
	Tag        string
	Processor  processor.Processor
	Log        logr.Logger
	canceled   bool
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.
func New(conn Connection, ch Channel, p processor.Processor, l logr.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Processor:  p,
		Log:        l,
	}
}

// NewFromConfig creates a new consumer instance. The setup of the amqp connection and channel is done according to the
// configuration.
func NewFromConfig(cfg Config, p processor.Processor, l logr.Logger) (*Consumer, error) {
	l.Info("Connecting RabbitMQ...")

	var conn Connection
	var err error
	if cfg.TLSConfig() != nil {
		conn, err = amqp.DialTLS(cfg.AmqpUrl(), cfg.TLSConfig())
	} else {
		conn, err = amqp.Dial(cfg.AmqpUrl())
	}
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

	if err := Setup(cfg, ch, l); err != nil {
		return nil, err
	}

	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Queue:      cfg.QueueName(),
		Tag:        cfg.ConsumerTag(),
		Processor:  p,
		Log:        l,
	}, nil
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume(ctx context.Context) error {
	c.Log.Info("Registering consumer... ")
	msgs, err := c.Channel.Consume(c.Queue, c.Tag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %s", err)
	}

	c.Log.Info("Succeeded registering consumer.")
	c.Log.Info("Waiting for messages...")

	remoteClose := make(chan *amqp.Error)
	c.Channel.NotifyClose(remoteClose)

	done := make(chan error)
	go c.consume(msgs, done)

	select {
	case err := <-remoteClose:
		return err

	case <-ctx.Done():
		c.canceled = true
		err := c.Channel.Cancel(c.Tag, false)
		if err == nil {
			err = <-done
		}
		return err

	case err := <-done:
		return err
	}
}

func (c *Consumer) consume(msgs <-chan amqp.Delivery, done chan error) {
	for m := range msgs {
		d := delivery.New(m)
		if c.canceled {
			d.Nack(true)
			continue
		}
		if err := c.checkError(c.Processor.Process(d)); err != nil {
			done <- err
			return
		}
	}
	done <- nil
}

func (c *Consumer) checkError(err error) error {
	switch err.(type) {
	case *processor.CreateCommandError:
		c.Log.Error(err)
		return nil

	default:
		return err
	}
}

// Close tears the connection down, taking the channel with it.
func (c *Consumer) Close() error {
	if c.Connection == nil {
		return nil
	}
	return c.Connection.Close()
}

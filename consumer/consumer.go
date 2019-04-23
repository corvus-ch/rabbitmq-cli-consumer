package consumer

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/streadway/amqp"
)

type Consumer struct {
	Connection Connection
	Channel    Channel
	Queue      string
	Tag        string
	Processor  worker.Process
	Log        logr.Logger
	canceled   bool
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.
func New(conn Connection, ch Channel, p worker.Process, l logr.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Processor:  p,
		Log:        l,
	}
}

// NewFromConfig creates a new consumer instance. The setup of the amqp connection and channel is done according to the
// configuration.
func NewFromConfig(cfg Config, p worker.Process, l logr.Logger) (*Consumer, error) {
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
	defer func() {
		if r := recover(); r != nil {
			done <- fmt.Errorf("fatal error in worker: %v", r)
		}
	}()

	for m := range msgs {
		if c.canceled {
			m.Reject(true)
			continue
		}

		attr, err := newAttributes(m)
		if err != nil {
			m.Reject(true)
			continue
		}

		ack, err := c.Processor(attr, bytes.NewBuffer(m.Body), c.Log)
		acknowledge(m, ack, c.Log)
	}

	done <- nil
}

func acknowledge(d amqp.Delivery, ack worker.Acknowledgment, log logr.Logger) {
	level := 1

	switch ack {
	case worker.Reject:
		log.Info("Reject message")
		d.Reject(false)
		break

	case worker.Requeue:
		d.Reject(true)
		break

	default:
		log = log.WithField("reason", "unknown acknowledgement")
		level = 0
		// Intentionally fall through to next case.

	case worker.Ack:
		log.V(level).Info("Acknowledge message")
		d.Ack(false)
		break
	}
}

// Close tears the connection down, taking the channel with it.
func (c *Consumer) Close() error {
	if c.Connection == nil {
		return nil
	}
	c.Log.V(1).Info("Closing AMQP connection.")
	return c.Connection.Close()
}

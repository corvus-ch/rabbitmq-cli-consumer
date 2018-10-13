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

    defer c.Connection.Close()
    go ConnectionCloseHandler(c.Channel.NotifyClose(make(chan *amqp.Error)), c)

	c.Log.Info("Waiting for messages...")

	done := make(chan error)
	go c.consume(msgs, done)

	select {
	case <-ctx.Done():
		c.canceled = true
		err := c.Channel.Cancel(c.Tag, false)
		if err != nil {
			return err
		}
		return <-done

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
					return
				}
				receiver <- err
			}
		}()
	}

	return receiver
}

// ConnectionCloseHandler calls os.Exit after the connection to RabbitMQ got closed.
func ConnectionCloseHandler(closeErr chan *amqp.Error, c *Consumer) {
	err := <-closeErr
	c.Log.Error("Connection closed: %v", err)
	os.Exit(10)
}

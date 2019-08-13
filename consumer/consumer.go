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
	Channels   ChannelMultiplexer
	Queue      string
	Tag        string
	Processor  processor.Processor
	Log        logr.Logger
	canceled   bool
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.
func New(conn Connection, cm ChannelMultiplexer, p processor.Processor, l logr.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channels:   cm,
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

	cl, err := NewChannelList(conn, cfg.NumChannels(), l)
	if nil != err {
		return nil, fmt.Errorf("failed creating channel(s): %v", err)
	}

	l.Info("Setting QoS... ")
	if err := cl.Qos(cfg.PrefetchCount(), 0, cfg.PrefetchIsGlobal()); err != nil {
		return nil, err
	}
	l.Info("Succeeded setting QoS.")

	ch, err := cl.FirstChannel()
	if nil != err {
		return nil, err
	}

	if err := Setup(cfg, ch, l); err != nil {
		return nil, err
	}

	return &Consumer{
		Connection: conn,
		Channels:   cl,
		Queue:      cfg.QueueName(),
		Tag:        cfg.ConsumerTag(),
		Processor:  p,
		Log:        l,
	}, nil
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume(ctx context.Context) error {
	remoteClose := make(chan *amqp.Error)
	done := make(chan error)

	c.Log.Info("Registering channels... ")
	for i, ch := range c.Channels.Channels() {
		msgs, err := ch.Consume(c.Queue, c.Tag, false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to register a channel: %s", err)
		}

		c.Log.Infof("Succeeded registering channel %d.", i)

		ch.NotifyClose(remoteClose)

		go c.consume(i, msgs, done)
	}

	c.Log.Info("Waiting for messages...")

	select {
	case err := <-remoteClose:
		return err

	case <-ctx.Done():
		return c.Cancel(done)

	case err := <-done:
		return err
	}
}

// Cancel marks the Consumer as cancelled, and then cancels all the Channels.
func (c *Consumer) Cancel(done chan error) error {
	c.canceled = true
	var firstError error
	for i, ch := range c.Channels.Channels() {
		c.Log.Infof("closing channel %d...", i)
		err := ch.Cancel(c.Tag, false)
		if err == nil {
			err = <-done
		}
		if nil != err && nil == firstError {
			firstError = err
		}
	}

	return firstError
}

func (c *Consumer) consume(channel int, msgs <-chan amqp.Delivery, done chan error) {
	for m := range msgs {
		d := delivery.New(m)
		if c.canceled {
			d.Nack(true)
			continue
		}
		if err := c.checkError(c.Processor.Process(channel, d)); err != nil {
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

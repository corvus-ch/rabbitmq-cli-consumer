package consumer

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/streadway/amqp"
)

const (
	EXIT_ACK            = 0
	EXIT_REJECT         = 3
	EXIT_REJECT_REQUEUE = 4
	EXIT_NACK           = 5
	EXIT_NACK_REQUEUE   = 6
	EMPTY_STRING        = "<empty>"
)

type Consumer struct {
	Channel         *amqp.Channel
	Connection      *amqp.Connection
	Queue           string
	Builder         command.Builder
	ErrLogger       *log.Logger
	InfLogger       *log.Logger
	Compression     bool
	IncludeMetadata bool
	StrictExitCode  bool
	OnFailure       int
}

func ConnectionCloseHandler(closeErr chan *amqp.Error, c *Consumer) {
	err := <-closeErr
	c.ErrLogger.Fatalf("Connection closed: %v", err)
	os.Exit(10)
}

func (c *Consumer) Consume(output bool) {
	c.InfLogger.Println("Registering consumer... ")
	msgs, err := c.Channel.Consume(c.Queue, "", false, false, false, false, nil)
	if err != nil {
		c.ErrLogger.Fatalf("Failed to register a consumer: %s", err)
	}

	c.InfLogger.Println("Succeeded registering consumer.")

	defer c.Connection.Close()
	defer c.Channel.Close()

	closeErr := make(chan *amqp.Error)
	closeErr = c.Connection.NotifyClose(closeErr)

	go ConnectionCloseHandler(closeErr, c)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			props := metadata.Properties{
				Headers:         d.Headers,
				ContentType:     d.ContentType,
				ContentEncoding: d.ContentEncoding,
				DeliveryMode:    d.DeliveryMode,
				Priority:        d.Priority,
				CorrelationId:   d.CorrelationId,
				ReplyTo:         d.ReplyTo,
				Expiration:      d.Expiration,
				MessageId:       d.MessageId,
				Timestamp:       d.Timestamp,
				Type:            d.Type,
				AppId:           d.AppId,
				UserId:          d.UserId,
			}

			delivery := metadata.DeliveryInfo{
				ConsumerTag:  d.ConsumerTag,
				MessageCount: d.MessageCount,
				DeliveryTag:  d.DeliveryTag,
				Redelivered:  d.Redelivered,
				Exchange:     d.Exchange,
				RoutingKey:   d.RoutingKey,
			}

			cmd, err := c.Builder.GetCommand(props, delivery, d.Body, output)
			if err != nil {
				c.ErrLogger.Printf("failed to create command: %v", err)
				d.Nack(true, true)
			}

			exitCode := cmd.Run()

			if err := c.ack(d, exitCode); err != nil {
				c.ErrLogger.Fatalf("Message acknowledgement error: %v", err)
				os.Exit(11)
			}
		}
	}()

	c.InfLogger.Println("Waiting for messages...")
	<-forever
}

func (c *Consumer) ack(d amqp.Delivery, exitCode int) error {
	if c.StrictExitCode == false {
		if exitCode == EXIT_ACK {
			d.Ack(true)
			return nil
		}
		switch c.OnFailure {
		case EXIT_REJECT:
			d.Reject(false)
		case EXIT_REJECT_REQUEUE:
			d.Reject(true)
		case EXIT_NACK:
			d.Nack(true, false)
		case EXIT_NACK_REQUEUE:
			d.Nack(true, true)
		default:
			d.Nack(true, true)
		}
		return nil
	}

	switch exitCode {
	case EXIT_ACK:
		d.Ack(true)
	case EXIT_REJECT:
		d.Reject(false)
	case EXIT_REJECT_REQUEUE:
		d.Reject(true)
	case EXIT_NACK:
		d.Nack(true, false)
	case EXIT_NACK_REQUEUE:
		d.Nack(true, true)
	default:
		d.Nack(true, true)
		return errors.New(fmt.Sprintf("Unexpected exit code %v", exitCode))
	}

	return nil
}

// New returns a initialized consumer based on config
func New(cfg *config.Config, builder command.Builder, errLogger, infLogger *log.Logger) (*Consumer, error) {
	infLogger.Println("Connecting RabbitMQ...")
	conn, err := amqp.Dial(cfg.AmqpUrl())
	if nil != err {
		return nil, fmt.Errorf("failed connecting RabbitMQ: %v", err)
	}
	infLogger.Println("Connected.")

	infLogger.Println("Opening channel...")
	ch, err := conn.Channel()
	if nil != err {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	infLogger.Println("Done.")

	if err := Initialize(cfg, ch, errLogger, infLogger); err != nil {
		return nil, err
	}

	return &Consumer{
		Channel:     ch,
		Connection:  conn,
		Queue:       cfg.RabbitMq.Queue,
		Builder:     builder,
		ErrLogger:   errLogger,
		InfLogger:   infLogger,
		Compression: cfg.RabbitMq.Compression,
	}, nil
}

// Initialize channel according to config
func Initialize(cfg *config.Config, ch Channel, errLogger, infLogger *log.Logger) error {
	infLogger.Println("Setting QoS... ")

	// Attempt to preserve BC here
	if cfg.Prefetch.Count == 0 {
		cfg.Prefetch.Count = 3
	}

	if err := ch.Qos(cfg.Prefetch.Count, 0, cfg.Prefetch.Global); err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	infLogger.Println("Succeeded setting QoS.")

	infLogger.Printf("Declaring queue \"%s\"...", cfg.RabbitMq.Queue)
	_, err := ch.QueueDeclare(cfg.RabbitMq.Queue, true, false, false, false, sanitizeQueueArgs(cfg))
	if nil != err {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Check for missing exchange settings to preserve BC
	if "" == cfg.Exchange.Name && "" == cfg.Exchange.Type && !cfg.Exchange.Durable && !cfg.Exchange.Autodelete {
		cfg.Exchange.Type = "direct"
	}

	// Empty Exchange name means default, no need to declare
	if "" != cfg.Exchange.Name {
		infLogger.Printf("Declaring exchange \"%s\"...", cfg.Exchange.Name)
		err = ch.ExchangeDeclare(cfg.Exchange.Name, cfg.Exchange.Type, cfg.Exchange.Durable, cfg.Exchange.Autodelete, false, false, amqp.Table{})

		if nil != err {
			return fmt.Errorf("failed to declare exchange: %v", err)
		}

		// Bind queue
		infLogger.Printf("Binding queue \"%s\" to exchange \"%s\"...", cfg.RabbitMq.Queue, cfg.Exchange.Name)
		err = ch.QueueBind(cfg.RabbitMq.Queue, transformToStringValue(cfg.QueueSettings.Routingkey), transformToStringValue(cfg.Exchange.Name), false, nil)

		if nil != err {
			return fmt.Errorf("failed to bind queue to exchange: %v", err)
		}
	}

	return nil
}

func sanitizeQueueArgs(cfg *config.Config) amqp.Table {

	args := make(amqp.Table)

	if cfg.QueueSettings.MessageTTL > 0 {
		args["x-message-ttl"] = int32(cfg.QueueSettings.MessageTTL)
	}

	if cfg.QueueSettings.DeadLetterExchange != "" {
		args["x-dead-letter-exchange"] = transformToStringValue(cfg.QueueSettings.DeadLetterExchange)

		if cfg.QueueSettings.DeadLetterRoutingKey != "" {
			args["x-dead-letter-routing-key"] = transformToStringValue(cfg.QueueSettings.DeadLetterRoutingKey)
		}
	}

	if len(args) > 0 {
		return args
	}

	return nil
}

func transformToStringValue(val string) string {
	if val == EMPTY_STRING {
		return ""
	}

	return val
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

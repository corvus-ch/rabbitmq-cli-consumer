package consumer

import (
	"fmt"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/streadway/amqp"
	"github.com/thockin/logr"
)

func Setup(cfg *config.Config, ch Channel, l logr.Logger) error {
	if err := setupQoS(cfg, ch, l); err != nil {
		return err
	}

	if err := declareQueue(cfg, ch, l); err != nil {
		return err
	}
	// Empty Exchange name means default, no need to declare
	if cfg.HasExchange() {
		if err := declareExchange(cfg, ch, l); err != nil {
			return err
		}
	}

	return nil
}

func setupQoS(cfg *config.Config, ch Channel, l logr.Logger) error {
	l.Info("Setting QoS... ")
	if err := ch.Qos(cfg.PrefetchCount(), 0, cfg.Prefetch.Global); err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	l.Info("Succeeded setting QoS.")
	return nil
}

func declareQueue(cfg *config.Config, ch Channel, l logr.Logger) error {
	l.Infof("Declaring queue \"%s\"...", cfg.RabbitMq.Queue)
	_, err := ch.QueueDeclare(cfg.RabbitMq.Queue, true, false, false, false, queueArgs(cfg))
	if nil != err {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	return nil
}

func declareExchange(cfg *config.Config, ch Channel, l logr.Logger) error {
	l.Infof("Declaring exchange \"%s\"...", cfg.Exchange.Name)
	if err := ch.ExchangeDeclare(
		cfg.Exchange.Name,
		cfg.ExchangeType(),
		cfg.Exchange.Durable,
		cfg.Exchange.Autodelete,
		false,
		false,
		amqp.Table{},
	); nil != err {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Bind queue
	l.Infof("Binding queue \"%s\" to exchange \"%s\"...", cfg.RabbitMq.Queue, cfg.Exchange.Name)
	for _, routingKey := range cfg.RoutingKeys() {
		if err := ch.QueueBind(
			cfg.RabbitMq.Queue,
			routingKey,
			cfg.ExchangeName(),
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to exchange: %v", err)
		}
	}

	return nil
}

func queueArgs(cfg *config.Config) amqp.Table {

	args := make(amqp.Table)

	if cfg.HasMessageTTL() {
		args["x-message-ttl"] = cfg.MessageTTL()
	}

	if cfg.HasDeadLetterExchange() {
		args["x-dead-letter-exchange"] = cfg.DeadLetterExchange()

		if cfg.HasDeadLetterRouting() {
			args["x-dead-letter-routing-key"] = cfg.DeadLetterRoutingKey()
		}
	}

	if cfg.HasPriority() {
		args["x-max-priority"] = cfg.Priority()
	}

	return args
}

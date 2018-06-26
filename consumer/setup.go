package consumer

import (
	"fmt"

	"github.com/bketelsen/logr"
	"github.com/streadway/amqp"
)

// Setup configures queues, exchanges and bindings in between according to the configuration.
func Setup(cfg Config, ch Channel, l logr.Logger) error {
	if err := setupQoS(cfg, ch, l); err != nil {
		return err
	}

	if cfg.MustDeclareQueue() {
		if err := declareQueue(cfg, ch, l); err != nil {
			return err
		}
	}

	// Empty Exchange name means default, no need to declare
	if cfg.HasExchange() {
		if err := declareExchange(cfg, ch, l); err != nil {
			return err
		}
	}

	return nil
}

func setupQoS(cfg Config, ch Channel, l logr.Logger) error {
	l.Info("Setting QoS... ")
	if err := ch.Qos(cfg.PrefetchCount(), 0, cfg.PrefetchIsGlobal()); err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	l.Info("Succeeded setting QoS.")
	return nil
}

func declareQueue(cfg Config, ch Channel, l logr.Logger) error {
	l.Infof("Declaring queue \"%s\"...", cfg.QueueName())
	_, err := ch.QueueDeclare(cfg.QueueName(), true, false, false, false, queueArgs(cfg))
	if nil != err {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	return nil
}

func declareExchange(cfg Config, ch Channel, l logr.Logger) error {
	l.Infof("Declaring exchange \"%s\"...", cfg.ExchangeName())
	if err := ch.ExchangeDeclare(
		cfg.ExchangeName(),
		cfg.ExchangeType(),
		cfg.ExchangeIsDurable(),
		cfg.ExchangeIsAutoDelete(),
		false,
		false,
		amqp.Table{},
	); nil != err {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Bind queue
	l.Infof("Binding queue \"%s\" to exchange \"%s\"...", cfg.QueueName(), cfg.ExchangeName())
	for _, routingKey := range cfg.RoutingKeys() {
		if err := ch.QueueBind(
			cfg.QueueName(),
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

func queueArgs(cfg Config) amqp.Table {

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

package acknowledger

import (
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

// Mapping of script exit codes and message acknowledgment.
const (
	exitAck           = 0
	exitReject        = 3
	exitRejectRequeue = 4
	exitNack          = 5
	exitNackRequeue   = 6
)

// Acknowledger does message acknowledgment depending on the scripts exit code.
type Acknowledger interface {
	Ack(d delivery.Delivery, code int) error
}

// New creates new Acknowledger using strict or default behaviour.
func New(strict bool, onFailure int) Acknowledger {
	if strict {
		return &Strict{}
	}

	return &Default{
		OnFailure: onFailure,
	}
}

// NewFromConfig creates a new Acknowledger from the given configuration.
func NewFromConfig(cfg *config.Config) Acknowledger {
	if cfg.RabbitMq.Stricfailure {
		return &Strict{}
	}

	return &Default{cfg.RabbitMq.Onfailure}
}

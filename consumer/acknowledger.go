package consumer

import (
	"fmt"
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

// Message acknowledgment depending on the scripts exit code.
type Acknowledger interface {
	Ack(d delivery.Delivery, code int) error
}

// Creates new Acknowledger using strict or default behaviour.
func NewAcknowledger(strict bool, onFailure int) Acknowledger {
	if strict {
		return &StrictAcknowledger{}
	}

	return &DefaultAcknowledger{
		OnFailure: onFailure,
	}
}

// The default Acknowledger.
type DefaultAcknowledger struct {
	Acknowledger
	OnFailure int
}

// Default acknowledgment using a predefined behaviour on script error.
func (a DefaultAcknowledger) Ack(d delivery.Delivery, code int) error {
	if code == exitAck {
		d.Ack()
		return nil
	}
	switch a.OnFailure {
	case exitReject:
		d.Reject(false)
	case exitRejectRequeue:
		d.Reject(true)
	case exitNack:
		d.Nack(false)
	case exitNackRequeue:
		d.Nack(true)
	default:
		d.Nack(true)
	}
	return nil
}

// The strict Acknowledger.
type StrictAcknowledger struct {
	Acknowledger
}

// Strict acknowledgment returning an err if script exits with an unknown exit code.
func (a StrictAcknowledger) Ack(d delivery.Delivery, code int) error {
	switch code {
	case exitAck:
		d.Ack()
	case exitReject:
		d.Reject(false)
	case exitRejectRequeue:
		d.Reject(true)
	case exitNack:
		d.Nack(false)
	case exitNackRequeue:
		d.Nack(true)
	default:
		d.Nack(true)
		return fmt.Errorf("unexpected exit code %v", code)
	}

	return nil
}

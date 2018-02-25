package acknowledger

import "github.com/corvus-ch/rabbitmq-cli-consumer/delivery"

// Default is an Acknowledger implementation using a configurable default behaviour for script errors.
type Default struct {
	OnFailure int
}

// Ack acknowledges the message on success or negatively acknowledges or rejects the message according to the configured
// on error behaviour.
func (a Default) Ack(d delivery.Delivery, code int) error {
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

package acknowledger

import (
	"fmt"

	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

// The strict Acknowledger.
type Strict struct{}

// Strict acknowledgment returning an err if script exits with an unknown exit code.
func (a Strict) Ack(d delivery.Delivery, code int) error {
	switch code {
	case exitAck:
		d.Ack(true)
	case exitReject:
		d.Reject(false)
	case exitRejectRequeue:
		d.Reject(true)
	case exitNack:
		d.Nack(true, false)
	case exitNackRequeue:
		d.Nack(true, true)
	default:
		d.Nack(true, true)
		return fmt.Errorf("unexpected exit code %v", code)
	}

	return nil
}

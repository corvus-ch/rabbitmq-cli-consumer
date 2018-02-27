package acknowledger

import "github.com/corvus-ch/rabbitmq-cli-consumer/delivery"

// The default Acknowledger.
type Default struct {
	OnFailure int
}

// Default acknowledgment using a predefined behaviour on script error.
func (a Default) Ack(d delivery.Delivery, code int) error {
	if code == exitAck {
		d.Ack(true)
		return nil
	}
	switch a.OnFailure {
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
	}
	return nil
}

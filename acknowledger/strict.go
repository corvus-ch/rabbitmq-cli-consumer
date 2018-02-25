package acknowledger

import (
	"fmt"

	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

// Strict is an Acknowledger implementation strictly using the scripts exit code.
type Strict struct{}

// Ack acknowledges the message on success or negatively acknowledges or rejects the message according to the scripts
// exit code. It is an error if the script does not exit with one of the predefined exit codes.
func (a Strict) Ack(d delivery.Delivery, code int) error {
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

package worker

import (
	"io"
	"time"

	"github.com/bketelsen/logr"
)

const (
	// Ack indicates the message has been successfully processed and can be
	// removed from the queue.
	Ack Acknowledgment = iota

	// Reject indicates the message could not be processed and never can and
	// therefore needs to be removed from the queue.
	Reject Acknowledgment = iota

	// Requeue indicates the message could not be processed but might be in
	// future and therefore should be redelivered to another consumer.
	Requeue Acknowledgment = iota
)

// Process takes care of the heavy lifting to be done for each consumed message.
// The implementation must be save to be called concurrently.
//
// If the worker fails to process a message, but it is confident future
// processing can be successful, an error shall be returned. If processing will
// not be possible any more, the worker shall panic.
//
// The logger shall not be used for error handling but for informational
// purposes only. When capturing the output of the actual worker scripts, its
// STDOUT should be logged as error.
type Process func(attr Attributes, payload io.Reader, log logr.Logger) (Acknowledgment, error)

// Acknowledgment represent the status of a message processing and indicates to
// the caller of a worker how the message has t obe acknowledged.
type Acknowledgment uint8

// Attributes is a data transfer object holding the attributes of an AMQP message.
type Attributes interface {
	AppId() string
	ConsumerTag() string
	ContentEncoding() string
	ContentType() string
	CorrelationId() string
	DeliveryMode() uint8
	DeliveryTag() uint64
	Exchange() string
	Expiration() string
	Headers() map[string]interface{}
	MessageId() string
	Priority() uint8
	Redelivered() bool
	ReplyTo() string
	RoutingKey() string
	Timestamp() time.Time
	Type() string
	UserId() string

	// The JSON serialized representation of this DTO.
	JSON() io.Reader
}

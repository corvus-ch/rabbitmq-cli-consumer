package delivery

import (
	"github.com/streadway/amqp"
)

// Delivery interface describes interface for messages
type Delivery interface {
	Ack() error
	Nack(requeue bool) error
	Reject(requeue bool) error
	Body() []byte
	Properties() Properties
	Info() Info
}

// New creates a new delivery instance from the given AMQP delivery.
func New(d amqp.Delivery) Delivery {
	return &delivery{d}
}

type delivery struct {
	d amqp.Delivery
}

// Ack acknowledges the message.
func (r delivery) Ack() error {
	return r.d.Ack(true)
}

// Nack negatively acknowledges the message.
func (r delivery) Nack(requeue bool) error {
	return r.d.Nack(true, requeue)
}

// Reject rejects the message.
func (r delivery) Reject(requeue bool) error {
	return r.d.Reject(requeue)
}

// Body returns the message body.
func (r delivery) Body() []byte {
	return r.d.Body
}

// Properties returns the properties struct for the message.
func (r delivery) Properties() Properties {
	return NewProperties(r.d)
}

// Info returns the delivery info struct for the message.
func (r delivery) Info() Info {
	return NewDeliveryInfo(r.d)
}

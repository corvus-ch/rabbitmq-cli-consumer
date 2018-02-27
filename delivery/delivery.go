package delivery

import (
	"github.com/streadway/amqp"
)

// Delivery interface describes interface for messages
type Delivery interface {
	Ack(multiple bool) error
	Nack(multiple, requeue bool) error
	Reject(requeue bool) error
	Body() []byte
	Properties() Properties
	Info() Info
}

func New(d amqp.Delivery) Delivery {
	return &delivery{d}
}

type delivery struct {
	d amqp.Delivery
}

func (r delivery) Ack(multiple bool) error {
	return r.d.Ack(multiple)
}

func (r delivery) Nack(multiple, requeue bool) error {
	return r.d.Nack(multiple, requeue)
}

func (r delivery) Reject(requeue bool) error {
	return r.d.Reject(requeue)
}

func (r delivery) Body() []byte {
	return r.d.Body
}

func (r delivery) Properties() Properties {
	return NewProperties(r.d)
}

func (r delivery) Info() Info {
	return NewDeliveryInfo(r.d)
}

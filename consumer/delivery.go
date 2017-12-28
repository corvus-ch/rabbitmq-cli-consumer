package consumer

import "github.com/streadway/amqp"

// Delivery interface describes interface for messages
type Delivery interface {
	Ack(multiple bool) error
	Nack(multiple, requeue bool) error
	Reject(requeue bool) error
	Body() []byte
}

func NewRabbitMqDelivery(d amqp.Delivery) Delivery {
	return &rabbitMqDelivery{d}
}

type rabbitMqDelivery struct {
	d amqp.Delivery
}

func (r rabbitMqDelivery) Ack(multiple bool) error {
	return r.d.Ack(multiple)
}

func (r rabbitMqDelivery) Nack(multiple, requeue bool) error {
	return r.d.Nack(multiple, requeue)
}

func (r rabbitMqDelivery) Reject(requeue bool) error {
	return r.d.Reject(requeue)
}

func (r rabbitMqDelivery) Body() []byte {
	return r.d.Body
}

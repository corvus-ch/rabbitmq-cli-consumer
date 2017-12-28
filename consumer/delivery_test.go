package consumer_test

import (
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/stretchr/testify/mock"
)

type TestDelivery struct {
	consumer.Delivery
	mock.Mock
}

func (t *TestDelivery) Ack(multiple bool) error {
	argstT := t.Called(multiple)

	return argstT.Error(0)
}

func (t *TestDelivery) Nack(multiple bool, requeue bool) error {
	argsT := t.Called(multiple, requeue)

	return argsT.Error(0)
}

func (t *TestDelivery) Reject(requeue bool) error {
	argsT := t.Called(requeue)

	return argsT.Error(0)
}

func (t *TestDelivery) Body() []byte {
	argsT := t.Called()

	return argsT.Get(0).([]byte)
}

package acknowledger_test

import (
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/stretchr/testify/mock"
)

type TestDelivery struct {
	mock.Mock
}

func (t *TestDelivery) Ack() error {
	argstT := t.Called()

	return argstT.Error(0)
}

func (t *TestDelivery) Nack(requeue bool) error {
	argsT := t.Called(requeue)

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

func (t *TestDelivery) Properties() delivery.Properties {
	argsT := t.Called()

	return argsT.Get(0).(delivery.Properties)
}

func (t *TestDelivery) Info() delivery.Info {
	argsT := t.Called()

	return argsT.Get(0).(delivery.Info)
}

package consumer_test

import (
	"fmt"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/magiconair/properties/assert"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
)

var ackTests = []struct {
	name   string
	method string
	tag    uint64
	args   []interface{}
	err    error
	call   func(d consumer.Delivery) error
}{
	{
		"ackMultiple",
		"Ack",
		3,
		[]interface{}{true},
		nil,
		func(d consumer.Delivery) error { return d.Ack(true) },
	},
	{
		"ackSingle",
		"Ack",
		5,
		[]interface{}{false},
		nil,
		func(d consumer.Delivery) error { return d.Ack(false) },
	},
	{
		"ackError",
		"Ack",
		7,
		[]interface{}{true},
		fmt.Errorf("ack"),
		func(d consumer.Delivery) error { return d.Ack(true) },
	},
	{
		"nackMultiple",
		"Nack",
		11,
		[]interface{}{true, false},
		nil,
		func(d consumer.Delivery) error { return d.Nack(true, false) },
	},
	{
		"nackMultipleRequeue",
		"Nack",
		17,
		[]interface{}{true, true},
		nil,
		func(d consumer.Delivery) error { return d.Nack(true, true) },
	},
	{
		"nackError",
		"Nack",
		19,
		[]interface{}{true, true},
		fmt.Errorf("nack"),
		func(d consumer.Delivery) error { return d.Nack(true, true) },
	},
	{
		"rejectMultiple",
		"Reject",
		23,
		[]interface{}{true},
		nil,
		func(d consumer.Delivery) error { return d.Reject(true) },
	},
	{
		"rejectSingle",
		"Reject",
		29,
		[]interface{}{false},
		nil,
		func(d consumer.Delivery) error { return d.Reject(false) },
	},
	{
		"rejectError",
		"Reject",
		31,
		[]interface{}{true},
		fmt.Errorf("reject"),
		func(d consumer.Delivery) error { return d.Reject(true) },
	},
}

func TestRabbitMqDelivery(t *testing.T) {
	for _, test := range ackTests {
		t.Run(test.name, func(t *testing.T) {
			a := TestAmqpAcknowledger{}
			d := consumer.NewRabbitMqDelivery(amqp.Delivery{
				Acknowledger: &a,
				DeliveryTag:  test.tag,
				Body:         []byte(test.name),
			})
			a.On(test.method, append([]interface{}{test.tag}, test.args...)...).Return(test.err)
			assert.Equal(t, test.call(d), test.err)
			assert.Equal(t, d.Body(), []byte(test.name))
			a.AssertExpectations(t)
		})
	}
}

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

type TestAmqpAcknowledger struct {
	amqp.Acknowledger
	mock.Mock
}

func (t TestAmqpAcknowledger) Ack(tag uint64, multiple bool) error {
	argstT := t.Called(tag, multiple)

	return argstT.Error(0)
}

func (t TestAmqpAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	argstT := t.Called(tag, multiple, requeue)

	return argstT.Error(0)
}

func (t TestAmqpAcknowledger) Reject(tag uint64, requeue bool) error {
	argstT := t.Called(tag, requeue)

	return argstT.Error(0)
}

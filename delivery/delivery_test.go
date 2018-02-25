package delivery_test

import (
	"fmt"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
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
	call   func(d delivery.Delivery) error
}{
	{
		"ack",
		"Ack",
		3,
		[]interface{}{true},
		nil,
		func(d delivery.Delivery) error { return d.Ack() },
	},
	{
		"ackError",
		"Ack",
		7,
		[]interface{}{true},
		fmt.Errorf("ack"),
		func(d delivery.Delivery) error { return d.Ack() },
	},
	{
		"nack",
		"Nack",
		11,
		[]interface{}{true, false},
		nil,
		func(d delivery.Delivery) error { return d.Nack(false) },
	},
	{
		"nackRequeue",
		"Nack",
		17,
		[]interface{}{true, true},
		nil,
		func(d delivery.Delivery) error { return d.Nack(true) },
	},
	{
		"nackError",
		"Nack",
		19,
		[]interface{}{true, true},
		fmt.Errorf("nack"),
		func(d delivery.Delivery) error { return d.Nack(true) },
	},
	{
		"reject",
		"Reject",
		23,
		[]interface{}{true},
		nil,
		func(d delivery.Delivery) error { return d.Reject(true) },
	},
	{
		"rejectRequeue",
		"Reject",
		29,
		[]interface{}{true},
		nil,
		func(d delivery.Delivery) error { return d.Reject(true) },
	},
	{
		"rejectError",
		"Reject",
		31,
		[]interface{}{true},
		fmt.Errorf("reject"),
		func(d delivery.Delivery) error { return d.Reject(true) },
	},
}

func TestRabbitMqDelivery(t *testing.T) {
	for _, test := range ackTests {
		t.Run(test.name, func(t *testing.T) {
			a := TestAcknowledger{}
			d := delivery.New(amqp.Delivery{
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

type TestAcknowledger struct {
	amqp.Acknowledger
	mock.Mock
}

func (t TestAcknowledger) Ack(tag uint64, multiple bool) error {
	argstT := t.Called(tag, multiple)

	return argstT.Error(0)
}

func (t TestAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	argstT := t.Called(tag, multiple, requeue)

	return argstT.Error(0)
}

func (t TestAcknowledger) Reject(tag uint64, requeue bool) error {
	argstT := t.Called(tag, requeue)

	return argstT.Error(0)
}

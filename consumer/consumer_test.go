package consumer_test

import (
	"fmt"
	"testing"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/corvus-ch/rabbitmq-cli-consumer/processor"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var consumeTests = []struct {
	name  string
	setup func(*testing.T, *TestChannel, *TestProcessor, chan amqp.Delivery, amqp.Delivery) error
	log   string
}{
	{
		"happy path",
		func(t *testing.T, ch *TestChannel, p *TestProcessor, msgs chan amqp.Delivery, d amqp.Delivery) error {
			ch.On("Consume", t.Name(), "", false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
			ch.On("NotifyClose", mock.Anything)
			p.On("Process", delivery.New(d)).Once().Return(nil)
			return nil
		},
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
	},
	{
		"consume error",
		func(t *testing.T, ch *TestChannel, p *TestProcessor, msgs chan amqp.Delivery, d amqp.Delivery) error {
			ch.On("Consume", t.Name(), "", false, false, false, false, nilAmqpTable).
				Once().
				Return(nil, fmt.Errorf("consume error"))
			return fmt.Errorf("failed to register a consumer: consume error")
		},
		"INFO Registering consumer... \n",
	},
	{
		"process error",
		func(t *testing.T, ch *TestChannel, p *TestProcessor, msgs chan amqp.Delivery, d amqp.Delivery) error {
			err := fmt.Errorf("process error")
			ch.On("Consume", t.Name(), "", false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
			ch.On("NotifyClose", mock.Anything)
			p.On("Process", delivery.New(d)).Once().Return(err)
			return err
		},
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
	},
	{
		"create command error",
		func(t *testing.T, ch *TestChannel, p *TestProcessor, msgs chan amqp.Delivery, d amqp.Delivery) error {
			err := processor.NewCreateCommandError(fmt.Errorf("create command error"))
			ch.On("Consume", t.Name(), "", false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
			ch.On("NotifyClose", mock.Anything)
			p.On("Process", delivery.New(d)).Once().Return(err)
			return nil
		},
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\nERROR failed to register a consumer: create command error\n",
	},
	{
		"ack error",
		func(t *testing.T, ch *TestChannel, p *TestProcessor, msgs chan amqp.Delivery, d amqp.Delivery) error {
			err := processor.NewAcknowledgmentError(fmt.Errorf("ack error"))
			ch.On("Consume", t.Name(), "", false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
			ch.On("NotifyClose", mock.Anything)
			p.On("Process", delivery.New(d)).Once().Return(err)
			return err
		},
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
	},
}

func TestConsumer_Consume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.name, func(t *testing.T) {
			done := make(chan error)
			msgs := make(chan amqp.Delivery)
			d := amqp.Delivery{}
			conn := new(TestConnection)
			conn.On("Close").Return(nil)
			ch := new(TestChannel)
			p := new(TestProcessor)
			exp := test.setup(t, ch, p, msgs, d)
			l := log.New(0)
			c := consumer.New(conn, ch, t.Name(), l)
			go func() {
				err := c.Consume(p)
				done <- err
			}()
			go func() {
				msgs <- d
				close(msgs)
			}()
			err := <-done
			assert.Equal(t, exp, err)
			assert.Equal(t, test.log, l.Buf().String())
			ch.AssertExpectations(t)
			p.AssertExpectations(t)
		})
	}
}

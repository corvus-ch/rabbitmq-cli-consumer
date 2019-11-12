package consumer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/corvus-ch/rabbitmq-cli-consumer/processor"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

const intMax = int(^uint(0) >> 1)

type setupFunc func(*testing.T, *consumeTest) error

type consumeTest struct {
	Name   string
	Setup  setupFunc
	Output string
	Tag    string

	sync        chan bool
	done        chan error
	msgs        chan amqp.Delivery
	ch          *TestChannel
	p           *TestProcessor
	a           *TestAmqpAcknowledger
	dd          []amqp.Delivery
	cancelCount int
}

func newSimpleConsumeTest(name, output string, setup setupFunc) *consumeTest {
	return newConsumeTest(name, output, 1, intMax, setup)
}

func newConsumeTest(name, output string, count uint64, cancelCount int, setup setupFunc) *consumeTest {
	a := new(TestAmqpAcknowledger)
	dd := make([]amqp.Delivery, count)
	for i := uint64(0); i < count; i++ {
		dd[i] = amqp.Delivery{Acknowledger: a, DeliveryTag: i}
	}
	return &consumeTest{
		Name:   name,
		Output: output,
		Setup:  setup,
		Tag:    "ctag",

		sync:        make(chan bool),
		done:        make(chan error),
		msgs:        make(chan amqp.Delivery),
		ch:          new(TestChannel),
		p:           new(TestProcessor),
		a:           a,
		dd:          dd,
		cancelCount: cancelCount,
	}
}

func (ct *consumeTest) Run(t *testing.T) {
	exp := ct.Setup(t, ct)
	l := log.New(0)
	cl := &consumer.ChannelList{}
	cl.AddChannel(ct.ch)
	c := consumer.New(nil, cl, ct.p, l)
	c.Queue = t.Name()
	c.Tag = ct.Tag
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ct.done <- c.Consume(ctx)
	}()
	go ct.produce(cancel)
	assert.Equal(t, exp, <-ct.done)
	assert.Equal(t, ct.Output, l.Buf().String())
	ct.ch.AssertExpectations(t)
	ct.p.AssertExpectations(t)
	ct.a.AssertExpectations(t)
}

func (ct *consumeTest) produce(cancel func()) {
	defer close(ct.msgs)
	if len(ct.dd) == 0 && ct.cancelCount == 0 {
		cancel()
		return
	}
	for i, d := range ct.dd {
		go func() {
			if i >= ct.cancelCount {
				<-ct.sync
				cancel()
				time.Sleep(time.Second)
				ct.sync <- true
				return
			}
		}()
		ct.msgs <- d
	}
}

var consumeTests = []*consumeTest{
	newConsumeTest(
		"happy path",
		"INFO Registering channels... \nINFO Succeeded registering channel 0.\nINFO Waiting for messages...\n",
		3,
		intMax,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[0])).Once().Return(nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[1])).Once().Return(nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[2])).Once().Return(nil)
			return nil
		},
	),
	newSimpleConsumeTest(
		"consume error",
		"INFO Registering channels... \n",
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(nil, fmt.Errorf("consume error"))
			return fmt.Errorf("failed to register a channel: consume error")
		},
	),
	newSimpleConsumeTest(
		"process error",
		"INFO Registering channels... \nINFO Succeeded registering channel 0.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := fmt.Errorf("process error")
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
	newSimpleConsumeTest(
		"create command error",
		"INFO Registering channels... \nINFO Succeeded registering channel 0.\nINFO Waiting for messages...\nERROR failed to register a consumer: create command error\n",
		func(t *testing.T, ct *consumeTest) error {
			err := processor.NewCreateCommandError(fmt.Errorf("create command error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[0])).Once().Return(err)
			return nil
		},
	),
	newSimpleConsumeTest(
		"ack error",
		"INFO Registering channels... \nINFO Succeeded registering channel 0.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := processor.NewAcknowledgmentError(fmt.Errorf("ack error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", 0, delivery.New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
}

func TestConsumer_Consume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.Name, test.Run)
	}
}

func TestConsumer_Consume_NotifyClose(t *testing.T) {
	ch := new(TestChannel)
	d := make(chan amqp.Delivery)
	done := make(chan error)
	l := log.New(0)

	ch.On("Consume", "", "", false, false, false, false, nilAmqpTable).Once().Return(d, nil)

	cl := &consumer.ChannelList{}
	cl.AddChannel(ch)
	c := consumer.New(nil, cl, new(TestProcessor), l)

	go func() {
		done <- c.Consume(context.Background())
	}()

	retry := 5
	for !ch.TriggerNotifyClose("server close") && retry > 0 {
		retry--
		if retry == 0 {
			t.Fatal("No notify handler registered.")
		}
		// When called too early, the close handler is not yet registered. Try again later.
		time.Sleep(time.Millisecond)
	}

	assert.Equal(t, &amqp.Error{Reason: "server close", Code: 320}, <-done)
	ch.AssertExpectations(t)
}

package consumer_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/bketelsen/logr"
	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/sebdah/goldie"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

const intMax = int(^uint(0) >> 1)

type setupFunc func(*testing.T, *consumeTest) error

type consumeTest struct {
	Setup setupFunc
	Tag   string

	sync        chan bool
	done        chan error
	msgs        chan amqp.Delivery
	ch          *TestChannel
	p           worker.Process
	a           *TestAmqpAcknowledger
	dd          []amqp.Delivery
	cancelCount int
}

func newSimpleConsumeTest(setup setupFunc) *consumeTest {
	return newConsumeTest(1, intMax, setup)
}

func newConsumeTest(count uint64, cancelCount int, setup setupFunc) *consumeTest {
	a := new(TestAmqpAcknowledger)
	dd := make([]amqp.Delivery, count)
	for i := uint64(0); i < count; i++ {
		dd[i] = amqp.Delivery{Acknowledger: a, DeliveryTag: i}
	}
	t := &consumeTest{
		Setup: setup,
		Tag:   "ctag",

		sync:        make(chan bool),
		done:        make(chan error),
		msgs:        make(chan amqp.Delivery),
		ch:          new(TestChannel),
		a:           a,
		dd:          dd,
		cancelCount: cancelCount,
	}

	return t
}

func (ct *consumeTest) Run(t *testing.T) {
	exp := ct.Setup(t, ct)
	l := log.New(0)
	c := consumer.New(nil, ct.ch, ct.p, l)
	c.Queue = t.Name()
	c.Tag = ct.Tag
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ct.done <- c.Consume(ctx)
	}()
	go ct.produce(cancel)
	assert.Equal(t, exp, <-ct.done)
	goldie.Assert(t, t.Name(), l.Buf().Bytes())
	ct.ch.AssertExpectations(t)
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

var consumeTests = map[string]*consumeTest{
	"happy path": newConsumeTest(
		3,
		intMax,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p = func(_ worker.Attributes, _ io.Reader, _ logr.Logger) (worker.Acknowledgment, error) {
				return worker.Ack, nil
			}
			ct.a.On("Ack", uint64(0), false).Return(nil)
			ct.a.On("Ack", uint64(1), false).Return(nil)
			ct.a.On("Ack", uint64(2), false).Return(nil)
			return nil
		},
	),

	"consume error": newSimpleConsumeTest(
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(nil, fmt.Errorf("consume error"))
			return fmt.Errorf("failed to register a consumer: consume error")
		},
	),

	"process error": newSimpleConsumeTest(
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p = func(_ worker.Attributes, _ io.Reader, _ logr.Logger) (worker.Acknowledgment, error) {
				return worker.Requeue, fmt.Errorf("process error")
			}
			ct.a.On("Reject", uint64(0), true).Return(nil)
			return nil
		},
	),
}

func TestConsumer_Consume(t *testing.T) {
	for name, test := range consumeTests {
		t.Run(name, test.Run)
	}
}

func TestConsumer_Consume_NotifyClose(t *testing.T) {
	ch := new(TestChannel)
	d := make(chan amqp.Delivery)
	done := make(chan error)
	l := log.New(0)

	ch.On("Consume", "", "", false, false, false, false, nilAmqpTable).Once().Return(d, nil)

	c := consumer.New(nil, ch, func(_ worker.Attributes, _ io.Reader, _ logr.Logger) (worker.Acknowledgment, error) {
		return worker.Ack, nil
	}, l)

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

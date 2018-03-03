package consumer_test

import (
	"context"
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
	dd          []amqp.Delivery
	cancelCount int
}

func newSimpleConsumetst(name, output string, setup setupFunc) *consumeTest {
	return newConsumeTest(name, output, 1, intMax, setup)
}

func newConsumeTest(name, output string, count uint64, cancelCount int, setup setupFunc) *consumeTest {
	dd := make([]amqp.Delivery, count)
	for i := uint64(0); i < count; i++ {
		dd[i] = amqp.Delivery{DeliveryTag: i}
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
		dd:          dd,
		cancelCount: cancelCount,
	}
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
	assert.Equal(t, ct.Output, l.Buf().String())
	ct.ch.AssertExpectations(t)
	ct.p.AssertExpectations(t)
}

func (ct *consumeTest) produce(cancel func()) {
	defer close(ct.msgs)
	for i, d := range ct.dd {
		go func() {
			if i >= ct.cancelCount {
				<-ct.sync
				cancel()
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
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		3,
		intMax,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", delivery.New(ct.dd[0])).Once().Return(nil)
			ct.p.On("Process", delivery.New(ct.dd[1])).Once().Return(nil)
			ct.p.On("Process", delivery.New(ct.dd[2])).Once().Return(nil)
			return nil
		},
	),
	newSimpleConsumetst(
		"consume error",
		"INFO Registering consumer... \n",
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(nil, fmt.Errorf("consume error"))
			return fmt.Errorf("failed to register a consumer: consume error")
		},
	),
	newSimpleConsumetst(
		"process error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := fmt.Errorf("process error")
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", delivery.New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
	newSimpleConsumetst(
		"create command error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\nERROR failed to register a consumer: create command error\n",
		func(t *testing.T, ct *consumeTest) error {
			err := processor.NewCreateCommandError(fmt.Errorf("create command error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", delivery.New(ct.dd[0])).Once().Return(err)
			return nil
		},
	),
	newSimpleConsumetst(
		"ack error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := processor.NewAcknowledgmentError(fmt.Errorf("ack error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", delivery.New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
}

func TestConsumer_Consume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.Name, test.Run)
	}
}

func TestConsumer_Close(t *testing.T) {
	t.Run("no connection", func(t *testing.T) {
		c := consumer.New(nil, nil, nil, log.New(0))
		assert.Nil(t, c.Close())
	})
	t.Run("with connection", func(t *testing.T) {
		conn := new(TestConnection)
		conn.On("Close").Once().Return(nil)
		c := consumer.New(conn, nil, nil, log.New(0))
		assert.Nil(t, c.Close())
		conn.AssertExpectations(t)
	})
	t.Run("close error", func(t *testing.T) {
		err := fmt.Errorf("close error")
		conn := new(TestConnection)
		conn.On("Close").Once().Return(err)
		c := consumer.New(conn, nil, nil, log.New(0))
		assert.Equal(t, err, c.Close())
		conn.AssertExpectations(t)
	})
}

func testConsumerCancel(t *testing.T, err error) {
	done := make(chan error)
	ch := new(TestChannel)
	ch.On("Consume", "queue", t.Name(), false, false, false, false, nilAmqpTable).Once().Return(make(chan *amqp.Delivery), nil)
	ch.On("Cancel", t.Name(), false).Once().Return(err)
	ctx, cancel := context.WithCancel(context.Background())
	c := consumer.New(nil, ch, nil, log.New(0))
	c.Queue = "queue"
	c.Tag = t.Name()
	go func() {
		done <- c.Consume(ctx)
	}()
	cancel()
	assert.Equal(t, err, <-done)
	ch.AssertExpectations(t)
}

func TestConsumer_Cancel(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		testConsumerCancel(t, nil)
	})
	t.Run("error", func(t *testing.T) {
		testConsumerCancel(t, fmt.Errorf("cancel error"))
	})
	ct := newConsumeTest(
		"skip remaining",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		3,
		1,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), ct.Tag, false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.ch.On("Cancel", ct.Tag, false).Return(nil)
			ct.p.On("Process", delivery.New(ct.dd[0])).Return(nil).Run(func(_ mock.Arguments) {
				ct.sync <- true
				<-ct.sync
			})
			return nil
		},
	)
	t.Run(ct.Name, ct.Run)
}

func TestConsumer_NotifyClose(t *testing.T) {
	err := amqp.ErrClosed
	done := make(chan error)
	var realChan chan *amqp.Error
	ch := new(TestChannel)
	ch.On("NotifyClose", mock.Anything).Return(done).Run(func(args mock.Arguments) {
		realChan = args.Get(0).(chan *amqp.Error)
	})
	c := consumer.New(nil, ch, nil, log.New(0))
	assert.Equal(t, done, c.NotifyClose(done))
	realChan <- err
	assert.Equal(t, err, <-done)
}

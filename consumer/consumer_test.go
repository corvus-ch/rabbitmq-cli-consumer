package consumer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func testConsumerCancel(t *testing.T, err error, withNotify bool) {
	done := make(chan error)
	ch := new(TestChannel)
	msgs := make(chan amqp.Delivery)
	ch.On("Consume", "queue", t.Name(), false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
	ch.On("Cancel", t.Name(), false).Once().Return(err).Run(func(_ mock.Arguments) {
		close(msgs)
	})
	ctx, cancel := context.WithCancel(context.Background())
	c := consumer.New(nil, ch, nil, log.New(0))
	c.Queue = "queue"
	c.Tag = t.Name()
	if withNotify {
		ch.On("NotifyClose", mock.Anything)
		c.NotifyClose(make(chan error))
	}
	go func() {
		done <- c.Consume(ctx)
	}()
	cancel()
	assert.Equal(t, err, <-done)
	ch.AssertExpectations(t)
}

var cancelTests = []*consumeTest{
	newConsumeTest(
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
			ct.a.On("Nack", uint64(1), true, true).Return(nil)
			ct.a.On("Nack", uint64(2), true, true).Return(nil)
			return nil
		},
	),
	newConsumeTest(
		"no messages",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		0,
		0,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), ct.Tag, false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.ch.On("Cancel", ct.Tag, false).Return(nil)
			return nil
		},
	),
}

func TestConsumer_Cancel(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		testConsumerCancel(t, nil, false)
	})
	t.Run("error", func(t *testing.T) {
		testConsumerCancel(t, fmt.Errorf("cancel error"), false)
	})
	t.Run("notify no block", func(t *testing.T) {
		ch := make(chan bool)
		go func() {
			testConsumerCancel(t, nil, true)
			ch <- true
		}()
		select {
		case <-ch:
			// Intentionally left blank.
		case <-time.After(5 * time.Second):
			t.Error("Timeout because notify handler is blocking cancel")
		}
	})
	for _, test := range cancelTests {
		t.Run(test.Name, test.Run)
	}
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

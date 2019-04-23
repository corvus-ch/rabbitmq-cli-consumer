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

func testConsumerCancel(t *testing.T, err error) {
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
	go func() {
		done <- c.Consume(ctx)
	}()
	cancel()
	assert.Equal(t, err, <-done)
	ch.AssertExpectations(t)
}

var cancelTests = map[string]*consumeTest{
	"skip remaining": newConsumeTest(
		3,
		1,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), ct.Tag, false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.ch.On("Cancel", ct.Tag, false).Return(nil)
			ct.p = func(_ worker.Attributes, _ io.Reader, _ logr.Logger) (worker.Acknowledgment, error) {
				ct.sync <- true
				<-ct.sync
				return worker.Ack, nil
			}
			ct.a.On("Ack", uint64(0), false).Return(nil)
			ct.a.On("Reject", uint64(1), true).Return(nil)
			ct.a.On("Reject", uint64(2), true).Return(nil)
			return nil
		},
	),
	"no messages": newConsumeTest(
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
		testConsumerCancel(t, nil)
	})
	t.Run("error", func(t *testing.T) {
		testConsumerCancel(t, fmt.Errorf("cancel error"))
	})
	t.Run("notify no block", func(t *testing.T) {
		ch := make(chan bool)
		go func() {
			testConsumerCancel(t, nil)
			ch <- true
		}()
		select {
		case <-ch:
			// Intentionally left blank.
		case <-time.After(5 * time.Second):
			t.Error("Timeout because notify handler is blocking cancel")
		}
	})
	for name, test := range cancelTests {
		t.Run(name, test.Run)
	}
}

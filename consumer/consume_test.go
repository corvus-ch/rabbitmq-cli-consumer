package consumer_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/bouk/monkey"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var processingTests = []struct {
	name      string
	ackMethod string
	ackArgs   []interface{}
	exit      int
	strict    bool
	onFailure int
}{
	{"ack", "Ack", []interface{}{true}, 0, false, 0},
	{"reject", "Reject", []interface{}{false}, 1, false, 3},
	{"rejectRequeue", "Reject", []interface{}{true}, 1, false, 4},
	{"nack", "Nack", []interface{}{true, false}, 1, false, 5},
	{"nackRequeue", "Nack", []interface{}{true, true}, 1, false, 6},
	{"fallback", "Nack", []interface{}{true, true}, 1, false, 0},
	{"strictAck", "Ack", []interface{}{true}, 0, true, 0},
	{"strictReject", "Reject", []interface{}{false}, 3, true, 0},
	{"strictRejectRequeue", "Reject", []interface{}{true}, 4, true, 0},
	{"strictNack", "Nack", []interface{}{true, false}, 5, true, 0},
	{"strictNackRequeue", "Nack", []interface{}{true, true}, 6, true, 0},
}

func TestProcessing(t *testing.T) {
	for _, test := range processingTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			b := new(TestBuilder)
			p := metadata.Properties{}
			di := metadata.DeliveryInfo{}
			cmd := new(TestCommand)
			body := []byte(test.name)
			c := consumer.Consumer{
				Builder:      b,
				Acknowledger: consumer.NewAcknowledger(test.strict, test.onFailure),
			}

			b.On("GetCommand", p, di, body).Return(cmd, nil)
			d.On("Body").Return(body)
			d.On(test.ackMethod, test.ackArgs...).Return(nil)
			cmd.On("Run").Return(test.exit)

			c.ProcessMessage(d, p, di)

			d.AssertExpectations(t)
			b.AssertExpectations(t)
			cmd.AssertExpectations(t)
		})
	}
}

func TestCommandFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	d := new(TestDelivery)
	b := new(TestBuilder)
	p := metadata.Properties{}
	di := metadata.DeliveryInfo{}
	body := []byte("cmdFailure")
	c := consumer.Consumer{
		Builder:   b,
		ErrLogger: log.New(buf, "", 0),
	}

	b.On("GetCommand", p, di, body).Return(new(TestCommand), fmt.Errorf("failed from test"))
	d.On("Body").Return(body)
	d.On("Nack", true, true).Return(nil)

	c.ProcessMessage(d, p, di)

	assert.Equal(t, "failed to create command: failed from test\n", buf.String())
	d.AssertExpectations(t)
	b.AssertExpectations(t)
}

func TestStrictDefault(t *testing.T) {
	fakeExit := func(code int) {
		panic(fmt.Sprintf("os.Exit called with: %v", code))
	}
	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	d := new(TestDelivery)
	b := new(TestBuilder)
	p := metadata.Properties{}
	di := metadata.DeliveryInfo{}
	cmd := new(TestCommand)
	body := []byte("strictDefault")
	c := consumer.Consumer{
		Builder:      b,
		Acknowledger: &consumer.StrictAcknowledger{},
		ErrLogger:    log.New(&bytes.Buffer{}, "", 0),
	}

	b.On("GetCommand", p, di, body).Return(cmd, nil)
	d.On("Body").Return(body)
	d.On("Nack", true, true).Return(nil)
	cmd.On("Run").Return(1)

	assert.PanicsWithValue(t, "os.Exit called with: 11", func() {
		c.ProcessMessage(d, p, di)
	}, "os.Exit was not called")

	d.AssertExpectations(t)
	b.AssertExpectations(t)
	cmd.AssertExpectations(t)
}

type TestBuilder struct {
	command.Builder
	mock.Mock
}

func (b *TestBuilder) SetOutputLogger(l *log.Logger) {
	b.Called(l)
}
func (b *TestBuilder) SetErrorLogger(l *log.Logger) {
	b.Called(l)
}
func (b *TestBuilder) SetOutputWriter(w io.Writer) {
	b.Called(w)
}
func (b *TestBuilder) SetErrorWriter(lw io.Writer) {
	b.Called(lw)
}
func (b *TestBuilder) SetCaptureOutput(capture bool) {
	b.Called(capture)
}
func (b *TestBuilder) SetCommand(cmd string) {
	b.Called(cmd)
}

func (b *TestBuilder) GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte) (command.Command, error) {
	argsT := b.Called(p, d, body)

	return argsT.Get(0).(command.Command), argsT.Error(1)
}

type TestCommand struct {
	command.Command
	mock.Mock
}

func (t TestCommand) Run() int {
	return t.Called().Int(0)
}

func (t TestCommand) Cmd() *exec.Cmd {
	return t.Called().Get(0).(*exec.Cmd)
}

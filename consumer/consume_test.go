package consumer_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/bouk/monkey"
	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thockin/logr"
)

var processingTests = []struct {
	name      string
	ackMethod string
	ackArgs   []interface{}
	exit      int
	strict    bool
	onFailure int
}{
	{"ack", "Ack", []interface{}{}, 0, false, 0},
	{"reject", "Reject", []interface{}{false}, 1, false, 3},
	{"rejectRequeue", "Reject", []interface{}{true}, 1, false, 4},
	{"nack", "Nack", []interface{}{false}, 1, false, 5},
	{"nackRequeue", "Nack", []interface{}{true}, 1, false, 6},
	{"fallback", "Nack", []interface{}{true}, 1, false, 0},
	{"strictAck", "Ack", []interface{}{}, 0, true, 0},
	{"strictReject", "Reject", []interface{}{false}, 3, true, 0},
	{"strictRejectRequeue", "Reject", []interface{}{true}, 4, true, 0},
	{"strictNack", "Nack", []interface{}{false}, 5, true, 0},
	{"strictNackRequeue", "Nack", []interface{}{true}, 6, true, 0},
}

func TestProcessing(t *testing.T) {
	for _, test := range processingTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			b := new(TestBuilder)
			p := delivery.Properties{}
			di := delivery.Info{}
			cmd := new(TestCommand)
			body := []byte(test.name)
			c := consumer.Consumer{
				Builder:      b,
				Acknowledger: acknowledger.New(test.strict, test.onFailure),
			}

			b.On("GetCommand", p, di, body).Return(cmd, nil)
			d.On("Body").Return(body)
			d.On(test.ackMethod, test.ackArgs...).Return(nil)
			d.On("Properties").Return(p)
			d.On("Info").Return(di)
			cmd.On("Run").Return(test.exit)

			c.ProcessMessage(d)

			d.AssertExpectations(t)
			b.AssertExpectations(t)
			cmd.AssertExpectations(t)
		})
	}
}

func TestCommandFailure(t *testing.T) {
	l := log.New(0)
	d := new(TestDelivery)
	b := new(TestBuilder)
	p := delivery.Properties{}
	di := delivery.Info{}
	body := []byte("cmdFailure")
	c := consumer.Consumer{
		Builder: b,
		Log:     l,
	}

	b.On("GetCommand", p, di, body).Return(new(TestCommand), fmt.Errorf("failed from test"))
	d.On("Body").Return(body)
	d.On("Nack", true).Return(nil)
	d.On("Properties").Return(p)
	d.On("Info").Return(di)

	c.ProcessMessage(d)

	assert.Equal(t, "ERROR failed to create command: failed from test\n", l.Buf().String())
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
	p := delivery.Properties{}
	di := delivery.Info{}
	cmd := new(TestCommand)
	body := []byte("strictDefault")
	c := consumer.Consumer{
		Builder:      b,
		Acknowledger: &acknowledger.Strict{},
		Log:          log.New(0),
	}

	b.On("GetCommand", p, di, body).Return(cmd, nil)
	d.On("Body").Return(body)
	d.On("Nack", true).Return(nil)
	d.On("Properties").Return(p)
	d.On("Info").Return(di)
	cmd.On("Run").Return(1)

	assert.PanicsWithValue(t, "os.Exit called with: 11", func() {
		c.ProcessMessage(d)
	}, "os.Exit was not called")

	d.AssertExpectations(t)
	b.AssertExpectations(t)
	cmd.AssertExpectations(t)
}

type TestBuilder struct {
	command.Builder
	mock.Mock
}

func (b *TestBuilder) SetLogger(l logr.Logger) {
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

func (b *TestBuilder) GetCommand(p delivery.Properties, d delivery.Info, body []byte) (command.Command, error) {
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

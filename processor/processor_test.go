package processor

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/bketelsen/logr"
	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/sebdah/goldie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var execCommandRunTests = []struct {
	name string
	cmd  *exec.Cmd
	code int
}{
	{
		"success",
		testCommand("echo", false, []string{}...),
		0,
	},
	{
		"error",
		testCommand("error", false, "lorem", "ipsum"),
		1,
	},
	{
		"errorCapture",
		testCommand("error", true, "dolor", "sit"),
		1,
	},
}

func TestProcessor_Run(t *testing.T) {
	for _, test := range execCommandRunTests {
		t.Run(test.name, func(t *testing.T) {
			l := log.New(0)
			p := processor{log: l, cmd: test.cmd}

			assert.Equal(t, p.run(), test.code)
			goldie.Assert(t, t.Name(), l.Buf().Bytes())
		})
	}
}

var properties = delivery.Properties{}
var info = delivery.Info{}

func TestProcessing(t *testing.T) {
	testProcessing(t, "happyPath", func(t *testing.T, a *TestAcknowledger, b *TestBuilder, d *TestDelivery) string {
		cmd := testCommand("exit", true, fmt.Sprintf("%d", 42))

		a.On("Ack", d, 42).Return(nil)
		b.On("GetCommand", properties, info, []byte(t.Name())).Return(cmd, nil)

		return ""
	})
	testProcessing(t, "GetCommandError", func(t *testing.T, a *TestAcknowledger, b *TestBuilder, d *TestDelivery) string {
		var cmd *exec.Cmd

		b.On("GetCommand", properties, info, []byte(t.Name())).Return(cmd, errors.New("invalid json"))
		d.On("Nack", true).Return(nil)

		return "failed to register a consumer: invalid json"
	})
	testProcessing(t, "AckError", func(t *testing.T, a *TestAcknowledger, b *TestBuilder, d *TestDelivery) string {
		cmd := testCommand("exit", true, fmt.Sprintf("%d", 42))

		a.On("Ack", d, 42).Return(errors.New("unexpected exit code 42"))
		b.On("GetCommand", properties, info, []byte(t.Name())).Return(cmd, nil)

		return "failed to aknowledge message: unexpected exit code 42"
	})
}

func testProcessing(t *testing.T, name string, setup func(t *testing.T, a *TestAcknowledger, b *TestBuilder, d *TestDelivery) string) {
	t.Run(name, func(t *testing.T) {
		di := delivery.Info{}
		pr := delivery.Properties{}
		body := []byte(t.Name())

		a := new(TestAcknowledger)
		b := new(TestBuilder)
		d := new(TestDelivery)
		p := New(b, a, log.New(0))

		d.On("Body").Return(body)
		d.On("Properties").Return(pr)
		d.On("Info").Return(di)

		exp := setup(t, a, b, d)
		err := p.Process(d)

		if len(exp) > 0 {
			assert.Equal(t, exp, err.Error())
		}
		a.AssertExpectations(t)
		b.AssertExpectations(t)
		d.AssertExpectations(t)
	})
}

func testCommand(command string, capture bool, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if capture {
		cmd.Stdout = ioutil.Discard
		cmd.Stderr = ioutil.Discard
	}

	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := helperProcessArgs()
	helperProcessAssertArgs(args)

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		helperProcessCmdEcho(args, 0)

	case "error":
		helperProcessCmdEcho(args, 1)

	case "exit":
		code, err := strconv.Atoi(args[0])
		if err != nil {
			code = 0
		}
		helperProcessCmdEcho(args[1:], code)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func helperProcessArgs() []string {
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	return args
}

func helperProcessAssertArgs(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No command")
		os.Exit(2)
	}
}

func helperProcessCmdEcho(args []string, code int) {
	for _, a := range args {
		fmt.Println(a)
	}
	os.Exit(code)
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

func (b *TestBuilder) GetCommand(p delivery.Properties, d delivery.Info, body []byte) (*exec.Cmd, error) {
	argsT := b.Called(p, d, body)

	return argsT.Get(0).(*exec.Cmd), argsT.Error(1)
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

type TestAcknowledger struct {
	mock.Mock
}

func (t *TestAcknowledger) Ack(d delivery.Delivery, code int) error {
	argsT := t.Called(d, code)

	return argsT.Error(0)
}

package processor

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"testing"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/sebdah/goldie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thockin/logr"
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
			p := processor{l: l, cmd: test.cmd}

			assert.Equal(t, p.run(), test.code)
			goldie.Assert(t, t.Name(), l.Buf().Bytes())
		})
	}
}

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
			pr := delivery.Properties{}
			di := delivery.Info{}
			cmd := testCommand("exit", true, fmt.Sprintf("%d", test.exit))
			body := []byte(test.name)
			p := &processor{
				b: b,
				a: acknowledger.New(test.strict, test.onFailure),
				l: log.New(0),
			}

			b.On("GetCommand", pr, di, body).Return(cmd, nil)
			d.On("Body").Return(body)
			d.On(test.ackMethod, test.ackArgs...).Return(nil)
			d.On("Properties").Return(pr)
			d.On("Info").Return(di)

			p.Process(d)

			d.AssertExpectations(t)
			b.AssertExpectations(t)
		})
	}
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
	delivery.Delivery
	mock.Mock
}

func (t *TestDelivery) Ack(multiple bool) error {
	argstT := t.Called(multiple)

	return argstT.Error(0)
}

func (t *TestDelivery) Nack(multiple bool, requeue bool) error {
	argsT := t.Called(multiple, requeue)

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

package exec_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker/exec"
	"github.com/sebdah/goldie"
	"github.com/stretchr/testify/assert"
)

var helper = flag.Bool("helper", false, "actually executes the test helper")

var processTests = map[string]struct {
	cmd string
	ack worker.Acknowledgment
	err string
}{
	"success": {
		testCommand("echo success"),
		worker.Ack,
		"",
	},
	"error": {
		testCommand("error requeue"),
		worker.Requeue,
		"exit status 1",
	},
	"reject": {
		testCommand("exit 42 reject"),
		worker.Reject,
		"exit status 42",
	},
	"unknown command": {
		"does not exist",
		worker.Requeue,
		"failed to start command: exec: \"does\": executable file not found in $PATH",
	},
}

func TestProcessor_Process(t *testing.T) {
	for name, test := range processTests {
		t.Run(name, func(t *testing.T) {
			t.Run("output", func(t *testing.T) {
				testProcess(t, test.cmd, test.ack, test.err, true)
			})
			t.Run("silent", func(t *testing.T) {
				testProcess(t, test.cmd, test.ack, test.err, false)
			})
		})
	}
}

func testProcess(t *testing.T, cmd string, expectedAck worker.Acknowledgment, expectedErr string, captured bool) {
	log := buffered.New(0)
	process := exec.New(cmd, []int{42}, captured, log)
	ack, err := process(&attributes{}, &bytes.Buffer{}, log)
	if len(expectedErr) > 0 {
		assert.EqualError(t, err, expectedErr)
	} else {
		assert.NoError(t, err)
	}
	assert.Equal(t, expectedAck, ack)
	if captured {
		goldie.Assert(t, t.Name(), log.Buf().Bytes())
	} else {
		assert.Empty(t, log.Buf().Bytes())
	}
}

func testCommand(command string) string {
	return fmt.Sprintf("%s -test.run=TestHelperProcess -helper -- %s", os.Args[0], command)
}

func TestHelperProcess(t *testing.T) {
	if !*helper {
		return
	}
	defer os.Exit(0)

	args := helperProcessArgs()

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		helperProcessCmdEcho(args, 0, os.Stdout)

	case "error":
		helperProcessCmdEcho(args[0:], 1, os.Stderr)

	case "exit":
		code, _ := strconv.Atoi(args[0])
		helperProcessCmdEcho(args[1:], code, os.Stderr)

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

func helperProcessCmdEcho(args []string, code int, w io.Writer) {
	for _, a := range args {
		fmt.Fprintln(w, a)
	}
	os.Exit(code)
}

type attributes struct {
	appId           string
	consumerTag     string
	contentEncoding string
	contentType     string
	correlationId   string
	deliveryMode    uint8
	deliveryTag     uint64
	exchange        string
	expiration      string
	headers         map[string]interface{}
	messageId       string
	priority        uint8
	redelivered     bool
	replyTo         string
	routingKey      string
	timestamp       time.Time
	msgType         string
	userId          string
}

func (a *attributes) AppId() string                   { return a.appId }
func (a *attributes) ConsumerTag() string             { return a.consumerTag }
func (a *attributes) ContentEncoding() string         { return a.contentEncoding }
func (a *attributes) ContentType() string             { return a.contentType }
func (a *attributes) CorrelationId() string           { return a.correlationId }
func (a *attributes) DeliveryMode() uint8             { return a.deliveryMode }
func (a *attributes) DeliveryTag() uint64             { return a.deliveryTag }
func (a *attributes) Exchange() string                { return a.exchange }
func (a *attributes) Expiration() string              { return a.expiration }
func (a *attributes) Headers() map[string]interface{} { return a.headers }
func (a *attributes) MessageId() string               { return a.messageId }
func (a *attributes) Priority() uint8                 { return a.priority }
func (a *attributes) Redelivered() bool               { return a.redelivered }
func (a *attributes) ReplyTo() string                 { return a.replyTo }
func (a *attributes) RoutingKey() string              { return a.routingKey }
func (a *attributes) Timestamp() time.Time            { return a.timestamp }
func (a *attributes) Type() string                    { return a.msgType }
func (a *attributes) UserId() string                  { return a.userId }
func (a *attributes) JSON() io.Reader {
	bs, _ := json.Marshal(a)
	return bytes.NewBuffer(bs)
}

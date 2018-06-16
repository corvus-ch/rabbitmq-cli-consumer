// +build integration

package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sebdah/goldie"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

var (
	command  = os.Args[0] + " -test.run=TestHelperProcess -- "
	amqpArgs = amqp.Table{
		"x-message-ttl":  int32(42),
		"x-max-priority": int32(42),
	}
)

var tests = []struct {
	name string
	// The arguments passed to the consumer command.
	args []string
	// The queue name
	queue string
	// The AMQ message sent.
	msg amqp.Publishing
	// The commands environment
	env []string
}{
	{
		"default",
		[]string{"-V", "-no-datetime", "-e", command, "-c", "fixtures/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("default")},
		[]string{},
	},
	{
		"compressed",
		[]string{"-V", "-no-datetime", "-e", command + "-comp", "-c", "fixtures/compressed.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("compressed")},
		[]string{},
	},
	{
		"output",
		[]string{"-V", "-no-datetime", "-o", "-e", command + "-output=-", "-c", "fixtures/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("output")},
		[]string{},
	},
	{
		"noLogs",
		[]string{"-V", "-no-datetime", "-o", "-e", command + "-output=-", "-c", "fixtures/no_logs.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("noLogs")},
		[]string{},
	},
	{
		"queueName",
		[]string{"-V", "-no-datetime", "-q", "altTest", "-e", command, "-c", "fixtures/default.conf"},
		"altTest",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("queueName")},
		[]string{},
	},
	{
		"properties",
		[]string{"-V", "-no-datetime", "-i", "-e", command, "-c", "fixtures/default.conf"},
		"test",
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: "679eaffe-e290-4565-a223-8b1ec10f6b26",
			Body:          []byte("properties"),
		},
		[]string{},
	},
	{
		"amqpUrl",
		[]string{"-V", "-no-datetime", "-e", command, "-c", "fixtures/amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("amqpUrl")},
		[]string{},
	},
	{
		"noAmqpUrl",
		[]string{"-V", "-no-datetime", "-e", command, "-c", "fixtures/no_amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("noAmqpUrl")},
		[]string{},
	},
	{
		"envAmqpUrl",
		[]string{"-V", "-no-datetime", "-e", command, "-c", "fixtures/no_amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("envAmqpUrl")},
		[]string{"AMQP_URL=amqp://guest:guest@localhost"},
	},
	{
		"envAmqpUrlNoConfig",
		[]string{"-V", "-no-datetime", "-e", command, "-q", "test"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("envAmqpUrlNoConfig")},
		[]string{"AMQP_URL=amqp://guest:guest@localhost"},
	},
	{
		"pipe",
		[]string{"-V", "-no-datetime", "-pipe", "-e", command + "-pipe", "-c", "fixtures/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("pipe")},
		[]string{},
	},
}

var noDeclareTests = []struct {
	name string
	// The arguments passed to the consumer command.
	args []string
}{
	{"noDeclare", []string{"-V", "-no-datetime", "-q", "noDeclare", "-e", command, "-no-declare"}},
	{"noDeclareConfig", []string{"-V", "-no-datetime", "-q", "noDeclareConfig", "-e", command, "-c", "fixtures/no_declare.conf"}},
}

func TestEndToEnd(t *testing.T) {
	conn, ch := prepare(t)
	defer conn.Close()
	defer ch.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("./command.log")
			cmd, stdout, stderr := startConsumer(t, test.env, test.args...)
			declareQueueAndPublish(t, ch, test.queue, test.msg)
			waitForOutput(t, stdout, "Processed!")
			stopConsumer(t, cmd)

			output, _ := ioutil.ReadFile("./command.log")
			goldie.Assert(t, t.Name()+"Command", output)
			assertOutput(t, stdout, stderr)
		})
	}

	for _, test := range noDeclareTests {
		t.Run(test.name, func(t *testing.T) {
			declareQueue(t, ch, test.name, amqpArgs)

			cmd, stdout, stderr := startConsumer(t, []string{}, test.args...)
			waitForOutput(t, stdout, "Waiting for messages...")
			stopConsumer(t, cmd)

			assertOutput(t, stdout, stderr)
		})
	}

	t.Run("declareError", func(t *testing.T) {
		declareQueue(t, ch, t.Name(), amqpArgs)

		cmd, _, _ := startConsumer(t, []string{}, "-V", "-no-datetime", "-q", t.Name(), "-e", command)
		exitErr := cmd.Wait()

		assert.NotNil(t, exitErr)
		assert.Equal(t, "exit status 1", exitErr.Error())
	})
}

func assertOutput(t *testing.T, stdout, stderr *bytes.Buffer) {
	goldie.Assert(t, t.Name()+"Output", bytes.Trim(stdout.Bytes(), "\x00"))
	goldie.Assert(t, t.Name()+"Error", bytes.Trim(stderr.Bytes(), "\x00"))
}

func prepare(t *testing.T) (*amqp.Connection, *amqp.Channel) {
	makeCmd := exec.Command("make", "build")
	if err := makeCmd.Run(); err != nil {
		t.Fatalf("could not build binary for: %v", err)
	}

	stopCmd := exec.Command("docker-compose", "down", "--volumes", "--remove-orphans")
	if err := stopCmd.Run(); err != nil {
		t.Fatalf("failed to stop docker stack: %v", err)
	}

	upCmd := exec.Command("docker-compose", "up", "-d")
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start docker stack: %v", err)
	}

	conn, err := connect("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatalf("failed to open AMQP connection: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}

	return conn, ch
}

func connect(url string) (*amqp.Connection, error) {
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			ticker.Stop()
			return nil, fmt.Errorf("timeout while trying to connect to RabbitMQ")

		case <-ticker.C:
			conn, err := amqp.Dial(url)
			if err == nil {
				return conn, nil
			}
		}
	}
}

func declareQueue(t *testing.T, ch *amqp.Channel, name string, args amqp.Table) amqp.Queue {
	q, err := ch.QueueDeclare(name, true, false, false, false, args)
	if err != nil {
		t.Errorf("failed to declare queue; %v", err)
	}

	return q
}

func declareQueueAndPublish(t *testing.T, ch *amqp.Channel, name string, msg amqp.Publishing) {
	q := declareQueue(t, ch, name, nil)
	if err := ch.Publish("", q.Name, false, false, msg); nil != err {
		t.Errorf("failed to publish message: %v", err)
	}
}

func startConsumer(t *testing.T, env []string, arg ...string) (cmd *exec.Cmd, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd = exec.Command("./rabbitmq-cli-consumer", arg...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(append(os.Environ(), "GO_WANT_HELPER_PROCESS=1"), env...)

	if err := cmd.Start(); err != nil {
		t.Errorf("failed to start consumer: %v", err)
	}

	return cmd, stdout, stderr
}

func stopConsumer(t *testing.T, cmd *exec.Cmd) {
	if err := cmd.Process.Kill(); err != nil {
		t.Errorf("failed to stop consumer: %v", err)
	}
}

func waitForOutput(t *testing.T, buf *bytes.Buffer, expect string) {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Errorf("timeout while waiting for output \"%s\"", expect)
			return

		case <-ticker.C:
			if strings.Contains(buf.String(), expect) {
				return
			}
		}
	}
}

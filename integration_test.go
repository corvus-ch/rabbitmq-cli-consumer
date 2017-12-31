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
)

var command = os.Args[0] + " -test.run=TestHelperProcess -- "

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
		"pipe",
		[]string{"-V", "-no-datetime", "-pipe", "-e", command + "-pipe", "-c", "fixtures/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("pipe")},
		[]string{},
	},
}

func TestEndToEnd(t *testing.T) {
	conn := prepare()
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Errorf("failed to open channel: %v", err)
	}
	defer ch.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("./command.log")
			cmd, stdout, stderr := startConsumer(t, test.env, test.args...)
			declareQueueAndPublish(t, ch, test.queue, test.msg)
			waitMessageProcessed(t, stdout)
			stopConsumer(t, cmd)

			output, _ := ioutil.ReadFile("./command.log")
			goldie.Assert(t, t.Name()+"Command", output)
			goldie.Assert(t, t.Name()+"Output", stdout.Bytes())
			goldie.Assert(t, t.Name()+"Error", stderr.Bytes())
		})
	}
}

func prepare() *amqp.Connection {
	makeCmd := exec.Command("make", "build")
	if err := makeCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "could not build binary for: %v\n", err)
		os.Exit(1)
	}

	stopCmd := exec.Command("docker-compose", "down", "--volumes", "--remove-orphans")
	if err := stopCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop docker stack: %v\n", err)
		os.Exit(1)
	}

	upCmd := exec.Command("docker-compose", "up", "-d")
	if err := upCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start docker stack: %v\n", err)
		os.Exit(1)
	}

	conn, err := connect("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open AMQP connection: %v", err)
		os.Exit(1)
	}

	return conn
}

func connect(url string) (*amqp.Connection, error) {
	timeout := time.After(10 * time.Second)
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

func declareQueueAndPublish(t *testing.T, ch *amqp.Channel, name string, msg amqp.Publishing) {
	q, err := ch.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		t.Errorf("failed to declare queue; %v", err)
	}

	err = ch.Publish("", q.Name, false, false, msg)
	if err != nil {
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

func waitMessageProcessed(t *testing.T, buf *bytes.Buffer) {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Error("timeout while waiting for message processing")
			return

		case <-ticker.C:
			if strings.Contains(buf.String(), "Processed!") {
				return
			}
		}
	}
}

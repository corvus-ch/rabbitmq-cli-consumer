// +build integration

package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/sebdah/goldie"
	"github.com/streadway/amqp"
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
		[]string{"-V", "-no-datetime", "-e", "go run test/command.go", "-c", "test/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("default")},
		[]string{},
	},
	{
		"compressed",
		[]string{"-V", "-no-datetime", "-e", "go run test/command.go -comp", "-c", "test/compressed.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("compressed")},
		[]string{},
	},
	{
		"output",
		[]string{"-V", "-no-datetime", "-o", "-e", "go run test/command.go -output=-", "-c", "test/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("output")},
		[]string{},
	},
	{
		"queueName",
		[]string{"-V", "-no-datetime", "-q", "altTest", "-e", "go run test/command.go", "-c", "test/default.conf"},
		"altTest",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("queueName")},
		[]string{},
	},
	{
		"mute",
		[]string{"-o", "-e", "go run test/command.go -output=-", "-c", "test/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("mute")},
		[]string{},
	},
	{
		"properties",
		[]string{"-V", "-no-datetime", "-i", "-e", "go run test/command.go", "-c", "test/default.conf"},
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
		[]string{"-V", "-no-datetime", "-e", "go run test/command.go", "-c", "test/amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("amqpUrl")},
		[]string{},
	},
	{
		"noAmqpUrl",
		[]string{"-V", "-no-datetime", "-e", "go run test/command.go", "-c", "test/no_amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("noAmqpUrl")},
		[]string{},
	},
	{
		"envAmqpUrl",
		[]string{"-V", "-no-datetime", "-e", "go run test/command.go", "-c", "test/no_amqp_url.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("envAmqpUrl")},
		[]string{"AMQP_URL=amqp://guest:guest@localhost"},
	},
	{
		"pipe",
		[]string{"-V", "-no-datetime", "-pipe", "-e", "go run test/command.go -pipe", "-c", "test/default.conf"},
		"test",
		amqp.Publishing{ContentType: "text/plain", Body: []byte("pipe")},
		[]string{},
	},
}

func TestMain(m *testing.M) {
	makeCmd := exec.Command("make", "build")
	if err := makeCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "could not build binary for: %v\n", err)
		os.Exit(1)
	}

	stopCmd := exec.Command("docker-compose", "stop")
	if err := stopCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop docker stack: %v\n", err)
		os.Exit(1)
	}

	upCmd := exec.Command("docker-compose", "up", "-d")
	if err := upCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start docker stack: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestEndToEnd(t *testing.T) {
	conn, err := newConnection("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Errorf("failed to open AMQP connection: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Errorf("failed to open channel: %v", err)
	}
	defer ch.Close()

	for _, test := range tests {
		os.Remove("./command.log")
		q, err := ch.QueueDeclare(test.queue, true, false, false, false, nil)
		if err != nil {
			t.Errorf("failed to declare queue; %v", err)
		}

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		cmd := exec.Command("./rabbitmq-cli-consumer", test.args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Env = append(os.Environ(), test.env...)

		if err := cmd.Start(); err != nil {
			t.Errorf("failed to start consumer: %v", err)
		}

		err = ch.Publish("", q.Name, false, false, test.msg)
		if err != nil {
			t.Errorf("failed to publish message: %v", err)
		}

		// Wait for message to be processed.
		// TODO: reliably detect if message actually was processed.
		time.Sleep(time.Second)

		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("failed to stop consumer: %v", err)
		}

		output, _ := ioutil.ReadFile("./command.log")
		goldie.Assert(t, test.name+"Command", output)
		goldie.Assert(t, test.name+"Output", stdout.Bytes())
		goldie.Assert(t, test.name+"Error", stderr.Bytes())
	}
}

func newConnection(url string) (*amqp.Connection, error) {
	time.Sleep(4 * time.Second)
	ticker := time.NewTicker(time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			conn, err := amqp.Dial(url)
			if err != nil {
				fmt.Println("trying to reconnect in one second...")
			} else {
				return conn, nil
			}
		case <-quit:
			ticker.Stop()

			return nil, fmt.Errorf("timeout while trying to connect to RabbitMQ")
		}
	}
}

// +build integration

package main_test

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
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

	conn, err := newConnection("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open AMQP connection: %v", err)
		os.Exit(1)
	}

	return conn
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
			declareQueue(t, ch, test.queue)

			cmd, stdout, stderr := newCommand(test.env, test.args...)
			if err := cmd.Start(); err != nil {
				t.Errorf("failed to start consumer: %v", err)
			}

			err = ch.Publish("", test.queue, false, false, test.msg)
			if err != nil {
				t.Errorf("failed to publish message: %v", err)
			}

			if waitMessageProcessed(stdout) {
				t.Errorf("timeout while waiting for message processing")
			}

			if err := cmd.Process.Kill(); err != nil {
				t.Errorf("failed to stop consumer: %v", err)
			}

			output, _ := ioutil.ReadFile("./command.log")
			goldie.Assert(t, t.Name()+"Command", output)
			goldie.Assert(t, t.Name()+"Output", stdout.Bytes())
			goldie.Assert(t, t.Name()+"Error", stderr.Bytes())
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	cli := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	outputFile := cli.String("output", "./command.log", "the output file")
	isCompressed := cli.Bool("comp", false, "whether the argument is compressed or not")
	isPipe := cli.Bool("pipe", false, "whether the argument passed via stdin (TRUE) or argument (FALSE)")
	cli.Parse(args)

	var f io.Writer
	f = os.Stdout
	if *outputFile != "-" {
		of, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file: %v\n", err)
			os.Exit(1)
		}
		defer of.Close()
		f = of
	}
	f.Write([]byte("Got executed\n"))

	var message []byte
	if *isPipe {
		var err error

		pipe := os.NewFile(3, "/proc/self/fd/3")
		defer pipe.Close()
		metadata, err := ioutil.ReadAll(pipe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read metadata from pipe: %v", err)
			os.Exit(1)
		}

		f.Write(metadata)
		f.Write([]byte("\n"))

		message, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read body from pipe: %v", err)
			os.Exit(1)
		}
	} else {
		message = []byte(os.Args[len(os.Args)-1])
	}

	f.Write(message)
	f.Write([]byte("\n"))

	if !*isPipe {
		var r io.Reader
		r = base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(message))
		if *isCompressed {
			zr, err := zlib.NewReader(r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create zlib reader: %v\n", err)
				os.Exit(1)
			}
			defer zr.Close()
			r = zr
		}
		original, err := ioutil.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to decompress input: %v\n", err)
			os.Exit(1)
		}
		f.Write(original)
		f.Write([]byte("\n"))
	}
}

func newConnection(url string) (*amqp.Connection, error) {
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

func declareQueue(t *testing.T, ch *amqp.Channel, name string) amqp.Queue {
	q, err := ch.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		t.Errorf("failed to declare queue; %v", err)
	}
	return q
}

func newCommand(env []string, arg ...string) (cmd *exec.Cmd, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd = exec.Command("./rabbitmq-cli-consumer", arg...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(append(os.Environ(), "GO_WANT_HELPER_PROCESS=1"), env...)
	return cmd, stdout, stderr
}

func waitMessageProcessed(buf *bytes.Buffer) bool {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			ticker.Stop()
			return true

		case <-ticker.C:
			if strings.Contains(buf.String(), "Processed!") {
				return false
			}
		}
	}
}

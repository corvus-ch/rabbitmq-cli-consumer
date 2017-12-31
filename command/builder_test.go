package command_test

import (
	"io"
	"log"
	"os/exec"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/stretchr/testify/assert"
	"bytes"
)

func assertLogger(t *testing.T, exp *log.Logger, got io.Writer, captured bool) {
	if captured {
		assert.IsType(t, &command.LogWriter{}, got)
		assert.Equal(t, exp, got.(*command.LogWriter).Logger)
	} else {
		assert.Nil(t, got)
	}
}

func createAndAssertCommand(t *testing.T, b command.Builder, body []byte) *exec.Cmd {
	c, err := b.GetCommand(metadata.Properties{}, metadata.DeliveryInfo{}, body)
	if err != nil {
		t.Errorf("failed to create command: %v", err)
	}
	assert.IsType(t, &command.ExecCommand{}, c)

	return c.Cmd()
}

func createAndAssertBuilder(t *testing.T, b command.Builder, name string, capture bool) (command.Builder, *log.Logger, *log.Logger) {
	outLog := log.New(&bytes.Buffer{}, "", 0)
	errLog := log.New(&bytes.Buffer{}, "", 0)
	builder, err := command.NewBuilder(b, name, capture, outLog, errLog)
	if err != nil {
		t.Errorf("failed to create builder: %v", err)
	}
	assert.Equal(t, b, builder)

	return builder, outLog, errLog
}

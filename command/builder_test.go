package command_test

import (
	"bytes"
	"io"
	"os/exec"
	"testing"

	log "github.com/corvus-ch/logr/buffered"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/stretchr/testify/assert"
)

func assertWriter(t *testing.T, exp *bytes.Buffer, got io.Writer, captured bool) {
	if captured {
		assert.Equal(t, exp, got)
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

func createAndAssertBuilder(t *testing.T, b command.Builder, name string, capture bool) (command.Builder, *bytes.Buffer, *bytes.Buffer) {
	var iBuf *bytes.Buffer
	var eBuf *bytes.Buffer
	l := log.New(0)
	builder, err := command.NewBuilder(b, name, capture, l, iBuf, eBuf)
	if err != nil {
		t.Errorf("failed to create builder: %v", err)
	}
	assert.Equal(t, b, builder)

	return builder, iBuf, eBuf
}

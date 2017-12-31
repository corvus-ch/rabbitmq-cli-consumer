package command_test

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
)

func assertLogger(t *testing.T, exp *log.Logger, got io.Writer, captured bool) {
	if captured {
		assert.IsType(t, &command.LogWriter{}, got)
		assert.Equal(t, exp, got.(*command.LogWriter).Logger)
	} else {
		assert.Nil(t, got)
	}
}

package command

import (
	"io"
	"os/exec"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

// Builder describes the interface used to build the command to be executed for a received message.
type Builder interface {
	// SetLogger sets the logger passed to the command used while building.
	SetLogger(l logr.Logger)

	// SetOutputWriter sets the writer used for capturing the commands STDOUT when capturing is enabled.
	SetOutputWriter(w io.Writer)

	// SetErrorWriter sets the writer used for capturing the commands STDERR when capturing is enabled.
	SetErrorWriter(lw io.Writer)

	// SetCaptureOutput enables/disables output capturing of the command.
	SetCaptureOutput(captrue bool)

	// SetCommand sets the command to be executed for each received message.
	SetCommand(cmd string)

	// GetCommand gets the executable command for the given message data.
	GetCommand(p delivery.Properties, d delivery.Info, body []byte) (*exec.Cmd, error)
}

// NewBuilder ensures a builder struct is setup and ready to be used.
func NewBuilder(b Builder, cmd string, capture bool, l logr.Logger, infoW, errW io.Writer) (Builder, error) {
	b.SetCommand(cmd)
	b.SetLogger(l)
	b.SetOutputWriter(infoW)
	b.SetErrorWriter(errW)
	b.SetCaptureOutput(capture)

	return b, nil
}

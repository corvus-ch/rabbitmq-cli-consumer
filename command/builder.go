package command

import (
	"io"

	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
	"github.com/thockin/logr"
)

// Builder describes the interface used to build the command to be executed for a received message.
type Builder interface {
	// SetLogger sets the logger passed to the command used while building.
	SetLogger(l logr.Logger)
	SetOutputWriter(w io.Writer)
	SetErrorWriter(lw io.Writer)
	SetCaptureOutput(captrue bool)
	SetCommand(cmd string)
	GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte) (Command, error)
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

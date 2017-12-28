package command

import (
	"io"
	"log"

	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
)

type Builder interface {
	SetOutputLogger(l *log.Logger)
	SetErrorLogger(l *log.Logger)
	SetOutputWriter(w io.Writer)
	SetErrorWriter(lw io.Writer)
	SetCaptureOutput(captrue bool)
	SetCommand(cmd string)
	GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte) (*Command, error)
}

func NewBuilder(b Builder, cmd string, capture bool, outLogger, errLogger *log.Logger) (Builder, error) {
	b.SetCommand(cmd)
	b.SetOutputLogger(outLogger)
	b.SetErrorLogger(errLogger)
	b.SetOutputWriter(NewLogWriter(outLogger))
	b.SetErrorWriter(NewLogWriter(errLogger))
	b.SetCaptureOutput(capture)

	return b, nil
}

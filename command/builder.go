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
	SetCommand(cmd string)
	GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte, capture bool) (*Command, error)
}

func NewBuilder(b Builder, cmd string, outLogger, errLogger *log.Logger) (Builder, error) {
	b.SetCommand(cmd)
	b.SetOutputLogger(outLogger)
	b.SetErrorLogger(errLogger)
	b.SetOutputWriter(NewLogWriter(outLogger))
	b.SetErrorWriter(NewLogWriter(errLogger))

	return b, nil
}

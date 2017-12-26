package command

import (
	"io"
	"log"
	"os/exec"
	"strings"
)

type PipeBuilder struct {
	Builder
	outLogger    *log.Logger
	errLogger    *log.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
}

func (b *PipeBuilder) SetOutputLogger(l *log.Logger) {
	b.outLogger = l
}

func (b *PipeBuilder) SetErrorLogger(l *log.Logger) {
	b.errLogger = l
}

func (b *PipeBuilder) SetOutputWriter(w io.Writer) {
	b.outputWriter = w
}

func (b *PipeBuilder) SetErrorWriter(w io.Writer) {
	b.errorWriter = w
}

func (b *PipeBuilder) SetCommand(cmd string) {
	var args []string

	if split := strings.Split(cmd, " "); len(split) > 1 {
		cmd, args = split[0], split[1:]
	}

	b.cmd = cmd
	b.args = args
}

func (b *PipeBuilder) GetCommand(payload io.Reader, capture bool) (*Command, error) {
	c := &Command{
		outLogger: b.outLogger,
		errLogger: b.errLogger,
		cmd:       exec.Command(b.cmd, b.args...),
	}

	c.cmd.Stdin = payload

	if capture {
		c.cmd.Stdout = b.outputWriter
		c.cmd.Stderr = b.errorWriter
	}

	return c, nil
}

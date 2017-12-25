package command

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

type ArgumentBuilder struct {
	Builder
	outLogger    *log.Logger
	errLogger    *log.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
}

func (b *ArgumentBuilder) SetOutputLogger(l *log.Logger) {
	b.outLogger = l
}

func (b *ArgumentBuilder) SetErrorLogger(l *log.Logger) {
	b.errLogger = l
}

func (b *ArgumentBuilder) SetOutputWriter(w io.Writer) {
	b.outputWriter = w
}

func (b *ArgumentBuilder) SetErrorWriter(w io.Writer) {
	b.errorWriter = w
}

func (b *ArgumentBuilder) SetCommand(cmd string) {
	var args []string

	if split := strings.Split(cmd, " "); len(split) > 1 {
		cmd, args = split[0], split[1:]
	}

	b.cmd = cmd
	b.args = args
}

func (b *ArgumentBuilder) GetCommand(payload io.Reader, capture bool) (*Command, error) {
	arg, err := ioutil.ReadAll(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %v", err)
	}

	c := &Command{
		outLogger: b.outLogger,
		errLogger: b.errLogger,
		cmd:       exec.Command(b.cmd, append(b.args, string(arg))...),
	}

	if capture {
		c.cmd.Stdout = b.outputWriter
		c.cmd.Stderr = b.errorWriter
	}

	return c, nil
}

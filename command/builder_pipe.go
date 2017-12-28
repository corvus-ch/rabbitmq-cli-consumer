package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
)

type PipeBuilder struct {
	Builder
	outLogger    *log.Logger
	errLogger    *log.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
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

func (b *PipeBuilder) SetCaptureOutput(capture bool) {
	b.capture = capture
}

func (b *PipeBuilder) GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte) (*Command, error) {

	meta, err := json.Marshal(&struct {
		Properties   metadata.Properties   `json:"properties"`
		DeliveryInfo metadata.DeliveryInfo `json:"delivery_info"`
	}{

		Properties:   p,
		DeliveryInfo: d,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshall matadata: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %v", err)
	}

	c := &Command{
		outLogger: b.outLogger,
		errLogger: b.errLogger,
		cmd:       exec.Command(b.cmd, b.args...),
	}

	c.cmd.Stdin = bytes.NewBuffer(body)
	c.cmd.ExtraFiles = []*os.File{r}

	if b.capture {
		c.cmd.Stdout = b.outputWriter
		c.cmd.Stderr = b.errorWriter
	}

	w.Write(meta)
	w.Close()

	return c, nil
}

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

type PipeBuilder struct {
	Builder
	log          logr.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
}

// SetLogger is part of Builder.
func (b *PipeBuilder) SetLogger(l logr.Logger) {
	b.log = l
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

func (b *PipeBuilder) GetCommand(p delivery.Properties, d delivery.Info, body []byte) (*exec.Cmd, error) {

	meta, err := json.Marshal(&struct {
		Properties   delivery.Properties `json:"properties"`
		DeliveryInfo delivery.Info       `json:"delivery_info"`
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

	cmd := exec.Command(b.cmd, b.args...)

	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewBuffer(body)
	cmd.ExtraFiles = []*os.File{r}

	if b.capture {
		cmd.Stdout = b.outputWriter
		cmd.Stderr = b.errorWriter
	}

	w.Write(meta)
	w.Close()

	return cmd, nil
}

package command

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

type ArgumentBuilder struct {
	Builder
	Compressed   bool
	WithMetadata bool
	log          logr.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
}

// SetLogger is part of Builder.
func (b *ArgumentBuilder) SetLogger(l logr.Logger) {
	b.log = l
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

func (b *ArgumentBuilder) SetCaptureOutput(capture bool) {
	b.capture = capture
}

func (b *ArgumentBuilder) GetCommand(p delivery.Properties, d delivery.Info, body []byte) (*exec.Cmd, error) {
	var err error
	payload := body
	if b.WithMetadata {
		payload, err = json.Marshal(&struct {
			Properties   delivery.Properties `json:"properties"`
			DeliveryInfo delivery.Info       `json:"delivery_info"`
			Body         string              `json:"body"`
		}{

			Properties:   p,
			DeliveryInfo: d,
			Body:         string(body),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshall payload: %v", err)
		}
	}

	buf, err := b.payloadBuffer(payload)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(b.cmd, append(b.args, buf.String())...)
	cmd.Env = os.Environ()

	if b.capture {
		cmd.Stdout = b.outputWriter
		cmd.Stderr = b.errorWriter
	}

	return cmd, nil
}

func (b *ArgumentBuilder) payloadBuffer(payload []byte) (*bytes.Buffer, error) {
	var w io.Writer
	buf := &bytes.Buffer{}

	enc := base64.NewEncoder(base64.StdEncoding, buf)
	defer enc.Close()
	w = enc

	if b.Compressed {
		comp, err := zlib.NewWriterLevel(enc, zlib.BestCompression)
		defer comp.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to create zlib handler: %v", err)
		}
		w = comp
		b.log.Info("Compressed message")
	}
	if _, err := w.Write(payload); err != nil {
		return nil, fmt.Errorf("failed to write payload: %v", err)
	}

	return buf, nil
}

package command

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
)

type ArgumentBuilder struct {
	Builder
	Compressed   bool
	WithMetadata bool
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

func (b *ArgumentBuilder) GetCommand(p metadata.Properties, d metadata.DeliveryInfo, body []byte, capture bool) (*Command, error) {
	var err error
	payload := body
	if b.WithMetadata {
		payload, err = json.Marshal(&struct {
			Properties   metadata.Properties   `json:"properties"`
			DeliveryInfo metadata.DeliveryInfo `json:"delivery_info"`
			Body         string                `json:"body"`
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

	c := &Command{
		outLogger: b.outLogger,
		errLogger: b.errLogger,
		cmd:       exec.Command(b.cmd, append(b.args, buf.String())...),
	}

	if capture {
		c.cmd.Stdout = b.outputWriter
		c.cmd.Stderr = b.errorWriter
	}

	return c, nil
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
		b.outLogger.Println("Compressed message")
	}
	if _, err := w.Write(payload); err != nil {
		return nil, fmt.Errorf("failed to write payload: %v", err)
	}

	return buf, nil
}

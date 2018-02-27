package processor

import (
	"os/exec"
	"sync"
	"syscall"

	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/thockin/logr"
)

type Processor interface {
	Process(delivery.Delivery) error
	Cancel() error
}

func New(b command.Builder, a acknowledger.Acknowledger, l logr.Logger) Processor {
	return &processor{b: b, a: a, l: l}
}

type processor struct {
	Processor
	b   command.Builder
	a   acknowledger.Acknowledger
	l   logr.Logger
	mu  sync.Mutex
	cmd *exec.Cmd
}

func (p *processor) Process(d delivery.Delivery) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error

	p.cmd, err = p.b.GetCommand(d.Properties(), d.Info(), d.Body())
	defer func() {
		p.cmd = nil
	}()
	if err != nil {
		d.Nack(true, true)
		return NewCreateCommandError(err)
	}

	exitCode := p.run()

	if err := p.a.Ack(d, exitCode); err != nil {
		return NewAcknowledgmentError(err)
	}

	return nil
}

func (p *processor) Cancel() error {
	if p.cmd != nil {
		p.cmd.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

func (p *processor) run() int {
	var err error

	p.l.Info("Processing message...")
	defer p.l.Info("Processed!")

	capture := p.cmd.Stdout == nil && p.cmd.Stderr == nil

	var out []byte

	if capture {
		out, err = p.cmd.CombinedOutput()
	} else {
		err = p.cmd.Run()
	}

	if err != nil {
		p.l.Info("Failed. Check error log for details.")
		p.l.Errorf("Error: %s\n", err)
		if capture {
			p.l.Errorf("Failed: %s", string(out))
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return 1
	}

	return 0
}

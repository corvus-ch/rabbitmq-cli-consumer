package processor

import (
	"os/exec"
	"sync"
	"syscall"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
)

// Processor describes the interface used by the consumer to process messages.
type Processor interface {
	Process(delivery.Delivery) error
}

// New creates a new processor instance.
func New(b command.Builder, a acknowledger.Acknowledger, l logr.Logger) Processor {
	return &processor{builder: b, ack: a, log: l}
}

type processor struct {
	Processor
	builder command.Builder
	ack     acknowledger.Acknowledger
	log     logr.Logger
	mu      sync.Mutex
	cmd     *exec.Cmd
}

// Process creates a new exec command using the builder and executes the command. The message gets acknowledged
// according to the commands exit code using the acknowledger.
func (p *processor) Process(d delivery.Delivery) error {
	var err error

	cmd, err := p.builder.GetCommand(d.Properties(), d.Info(), d.Body())
	if err != nil {
		d.Nack(true)
		return NewCreateCommandError(err)
	}

	exitCode := p.run(cmd)

	if err := p.ack.Ack(d, exitCode); err != nil {
		return NewAcknowledgmentError(err)
	}

	return nil
}

func (p *processor) run(cmd *exec.Cmd) int {
	var out []byte
	var err error
	capture := cmd.Stdout == nil && cmd.Stderr == nil

	if capture {
		out, err = cmd.CombinedOutput()
	} else {
		err = cmd.Run()
	}

	if err != nil {
		p.log.Info("Failed. Check error log for details.")
		p.log.Errorf("Error: %s\n", err)
		if capture {
			p.log.Errorf("Failed: %s", string(out))
		}

		return exitCode(err)
	}

	return 0
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return 1
}

package processor

import (
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/acknowledger"
	"github.com/corvus-ch/rabbitmq-cli-consumer/collector"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/delivery"
	"github.com/prometheus/client_golang/prometheus"
)

// Processor describes the interface used by the consumer to process messages.
type Processor interface {
	Process(int, delivery.Delivery) error
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
}

// Process creates a new exec command using the builder and executes the command. The message gets acknowledged
// according to the commands exit code using the acknowledger.
func (p *processor) Process(channel int, d delivery.Delivery) error {
	var err error

	cmd, err := p.builder.GetCommand(d.Properties(), d.Info(), d.Body())
	defer func() {
		cmd = nil
	}()
	if err != nil {
		d.Nack(true)
		return NewCreateCommandError(err)
	}

	start := time.Now()
	exitCode := p.run(cmd, channel)

	collector.ProcessCounter.With(prometheus.Labels{"exit_code": strconv.Itoa(exitCode)}).Inc()
	collector.ProcessDuration.Observe(time.Since(start).Seconds())
	if !d.Properties().Timestamp.IsZero() {
		collector.MessageDuration.Observe(time.Since(d.Properties().Timestamp).Seconds())
	}

	if err := p.ack.Ack(d, exitCode); err != nil {
		return NewAcknowledgmentError(err)
	}

	return nil
}

func (p *processor) run(cmd *exec.Cmd, channel int) int {
	p.log.Infof("[Channel %d] Processing message...", channel)
	defer p.log.Infof("[Channel %d] Processed!", channel)

	var out []byte
	var err error
	capture := cmd.Stdout == nil && cmd.Stderr == nil

	if capture {
		out, err = cmd.CombinedOutput()
	} else {
		err = cmd.Run()
	}

	if err != nil {
		p.log.Infof("[Channel %d] Failed. Check error log for details.", channel)
		p.log.Errorf("[Channel %d] Error: %s\n", channel, err)
		if capture {
			p.log.Errorf("[Channel %d] Failed: %s", channel, string(out))
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

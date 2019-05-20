// Package exec holds a worker implementation which runs a worker script as an
// external command.
package exec

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/logr/writer_adapter"
	"github.com/corvus-ch/rabbitmq-cli-consumer/collector"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type processor struct {
	cmdName     string
	cmdArgs     []string
	rejectCodes []int
	capture     bool
}

func New(cmd string, rejectCodes []int, capture bool, log logr.Logger) worker.Process {
	p := &processor{
		cmdName:     cmd,
		rejectCodes: rejectCodes,
		capture:     capture,
	}

	if split := strings.Split(cmd, " "); len(split) > 1 {
		p.cmdName, p.cmdArgs = split[0], split[1:]
	}

	return p.Process
}

func (p *processor) Process(attr worker.Attributes, payload io.Reader, log logr.Logger) (worker.Acknowledgment, error) {
	log.Info("Processing message...")
	defer log.Info("Processed!")

	cmd, w, err := p.createCommand(payload)
	if err != nil {
		return worker.Requeue, errors.Wrap(err, "failed to create command")
	}

	runner := p.runSilent

	if p.capture {
		runner = p.runOutputCaptured
	}

	start := time.Now()

	err = runner(cmd, w, attr, log)
	if err != nil && strings.Contains(err.Error(), "failed to start command") {
		return worker.Requeue, err
	}

	code := exitCode(err)

	collector.ProcessCounter.With(prometheus.Labels{"exit_code": strconv.Itoa(code)}).Inc()
	collector.ProcessDuration.Observe(time.Since(start).Seconds())
	if !attr.Timestamp().IsZero() {
		collector.MessageDuration.Observe(time.Since(attr.Timestamp()).Seconds())
	}

	return p.acknowledgement(code), err
}

func (p *processor) createCommand(in io.Reader) (*exec.Cmd, io.WriteCloser, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(p.cmdName, p.cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = in
	cmd.ExtraFiles = []*os.File{r}

	return cmd, w, nil
}

func (p *processor) runOutputCaptured(cmd *exec.Cmd, w io.WriteCloser, attr worker.Attributes, log logr.Logger) error {
	cmd.Stdout = writer_adapter.NewInfoWriter(log)
	cmd.Stderr = writer_adapter.NewErrorWriter(log)

	return p.run(cmd, w, attr, log)
}

func (p *processor) runSilent(cmd *exec.Cmd, w io.WriteCloser, attr worker.Attributes, log logr.Logger) error {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := p.run(cmd, w, attr, log)

	if err != nil && !strings.Contains(err.Error(), "failed to start command") {
		log.Errorf("Failed: %s", buf.String())
	}

	return err
}

func (p *processor) run(cmd *exec.Cmd, w io.WriteCloser, attr worker.Attributes, log logr.Logger) error {
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start command")
	}

	io.Copy(w, attr.JSON())
	w.Close()

	err = cmd.Wait()

	if err != nil {
		log.Info("Failed. Check error log for details.")
		log.Errorf("Error: %s\n", err)
	}

	return err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return 1
}

func (p *processor) acknowledgement(exitCode int) worker.Acknowledgment {
	if exitCode == 0 {
		return worker.Ack
	}

	for _, rejected := range p.rejectCodes {
		if exitCode == rejected {
			return worker.Reject
		}
	}

	return worker.Requeue
}

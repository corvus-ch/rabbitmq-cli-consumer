// Package exec holds a worker implementation which runs a worker script as an
// external command.
package exec

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/logr/writer_adapter"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/pkg/errors"
)

type processor struct {
	cmdName string
	cmdArgs []string

	rejectCodes []int

	errW io.Writer
	outW io.Writer
}

func New(cmd string, rejectCodes []int, capture bool, log logr.Logger) worker.Process {
	p := &processor{
		cmdName: cmd,

		rejectCodes: rejectCodes,
	}

	if capture {
		p.outW = writer_adapter.NewInfoWriter(log)
		p.errW = writer_adapter.NewErrorWriter(log)
	}

	if split := strings.Split(cmd, " "); len(split) > 1 {
		p.cmdName, p.cmdArgs = split[0], split[1:]
	}

	return p.Process
}

func (p *processor) Process(attr worker.Attributes, payload io.Reader, log logr.Logger) (worker.Acknowledgment, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return worker.Requeue, errors.Wrap(err, "failed to create pipe")
	}

	cmd := exec.Command(p.cmdName, p.cmdArgs...)
	cmd.Stdin = payload
	cmd.Stdout = p.outW
	cmd.Stderr = p.errW
	cmd.Env = os.Environ()
	cmd.ExtraFiles = []*os.File{r}

	err = cmd.Start()
	if err != nil {
		return worker.Requeue, errors.Wrap(err, "failed to start command")
	}

	io.Copy(w, attr.JSON())
	w.Close()

	err = cmd.Wait()
	code := exitCode(err)

	if code == 0 {
		return worker.Ack, err
	}

	for _, rejected := range p.rejectCodes {
		if code == rejected {
			return worker.Reject, err
		}
	}

	return worker.Requeue, err
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

package command

import (
	"os/exec"
	"syscall"

	"github.com/thockin/logr"
)

type Command interface {
	Run() int
	Cmd() *exec.Cmd
}

type ExecCommand struct {
	Command
	cmd *exec.Cmd
	log logr.Logger
}

func NewExecCommand(cmd *exec.Cmd, l logr.Logger) Command {
	return &ExecCommand{
		cmd: cmd,
		log: l,
	}
}

func (ec ExecCommand) Cmd() *exec.Cmd {
	return ec.cmd
}

func (c ExecCommand) Run() int {
	var err error

	c.log.Info("Processing message...")
	defer c.log.Info("Processed!")

	capture := c.cmd.Stdout == nil && c.cmd.Stderr == nil

	var out []byte

	if capture {
		out, err = c.cmd.CombinedOutput()
	} else {
		err = c.cmd.Run()
	}

	if err != nil {
		c.log.Info("Failed. Check error log for details.")
		c.log.Errorf("Error: %s\n", err)
		if capture {
			c.log.Errorf("Failed: %s", string(out))
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

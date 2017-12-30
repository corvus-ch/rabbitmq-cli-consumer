package command

import (
	"log"
	"os/exec"
	"syscall"
)

type Command interface {
	Run() int
	Cmd() *exec.Cmd
}

type ExecCommand struct {
	Command
	cmd       *exec.Cmd
	outLogger *log.Logger
	errLogger *log.Logger
}

func NewExecCommand(cmd *exec.Cmd, outLog, errLog *log.Logger) Command {
	return &ExecCommand{
		cmd:       cmd,
		outLogger: outLog,
		errLogger: errLog,
	}
}

func (ec ExecCommand) Cmd() *exec.Cmd {
	return ec.cmd
}

func (c ExecCommand) Run() int {
	var err error

	c.outLogger.Println("Processing message...")
	defer c.outLogger.Println("Processed!")

	if c.cmd.Stdout == nil && c.cmd.Stderr == nil {
		var out []byte
		out, err = c.cmd.CombinedOutput()
		if err != nil {
			c.errLogger.Printf("Failed: %s\n", string(out))
		}
	} else {
		err = c.cmd.Run()
	}

	if err != nil {
		c.outLogger.Println("Failed. Check error log for details.")
		c.errLogger.Printf("Error: %s\n", err)

		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return 1
	}

	return 0
}

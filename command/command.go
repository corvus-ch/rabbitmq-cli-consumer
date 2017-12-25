package command

import (
	"log"
	"os/exec"
	"syscall"
)

type Command struct {
	cmd       *exec.Cmd
	outLogger *log.Logger
	errLogger *log.Logger
}

func (c Command) Run() int {
	var err error

	c.outLogger.Println("Processing message...")

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

	c.outLogger.Println("Processed!")

	return 0
}

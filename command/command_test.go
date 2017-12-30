package command_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/magiconair/properties/assert"
	"github.com/sebdah/goldie"
)

var execCommandRunTests = []struct {
	name string
	cmd  *exec.Cmd
	code int
}{
	{
		"success",
		fakeExecCommand("echo", false, []string{}...),
		0,
	},
	{
		"error",
		fakeExecCommand("error", false, "lorem", "ipsum"),
		1,
	},
	{
		"errorCapture",
		fakeExecCommand("error", true, "dolor", "sit"),
		1,
	},
}

func TestExecCommandRun(t *testing.T) {
	for _, test := range execCommandRunTests {
		t.Run(test.name, func(t *testing.T) {
			outBuf := &bytes.Buffer{}
			errBuf := &bytes.Buffer{}
			execCmd := command.NewExecCommand(test.cmd, log.New(outBuf, "", 0), log.New(errBuf, "", 0))

			assert.Equal(t, execCmd.Run(), test.code)
			goldie.Assert(t, t.Name()+"Stdout", outBuf.Bytes())
			goldie.Assert(t, t.Name()+"Stderr", errBuf.Bytes())
		})
	}
}

func fakeExecCommand(command string, capture bool, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if capture {
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}
	}

	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		for _, a := range args {
			fmt.Println(a)
		}
		os.Exit(0)

	case "error":
		for _, a := range args {
			fmt.Println(a)
		}
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

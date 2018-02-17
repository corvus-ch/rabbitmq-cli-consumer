package command_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	log "github.com/corvus-ch/logr/buffered"
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
			l := log.New(0)
			execCmd := command.NewExecCommand(test.cmd, l)

			assert.Equal(t, execCmd.Run(), test.code)
			goldie.Assert(t, t.Name(), l.Buf().Bytes())
		})
	}
}

func fakeExecCommand(command string, capture bool, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if capture {
		cmd.Stdout = ioutil.Discard
		cmd.Stderr = ioutil.Discard
	}

	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := helperProcessArgs()
	helperProcessAssertArgs(args)

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		helperProcessCmdEcho(args, 0)

	case "error":
		helperProcessCmdEcho(args, 1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func helperProcessArgs() []string {
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	return args
}

func helperProcessAssertArgs(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No command")
		os.Exit(2)
	}
}

func helperProcessCmdEcho(args []string, code int) {
	for _, a := range args {
		fmt.Println(a)
	}
	os.Exit(code)
}

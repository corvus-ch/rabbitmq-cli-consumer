package main_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/urfave/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var exitErrHandlerTests = []struct {
	name string
	err  error
	out  string
	exit string
}{
	{
		"empty",
		nil,
		"",
		"",
	},
	{
		"exitCode",
		cli.NewExitError("", 42),
		"",
		"os.Exit called with: 42",
	},
	{
		"output",
		fmt.Errorf("normal error"),
		"normal error\n",
		"os.Exit called with: 1",
	},
	{
		"exitCodeOutput",
		cli.NewExitError("exit code error", 42),
		"exit code error\n",
		"os.Exit called with: 42",
	},
	{
		"outputFormatted",
		errors.WithMessage(fmt.Errorf("error"), "nested"),
		"error\nnested\n",
		"os.Exit called with: 1",
	},
}

func TestExitErrHandler(t *testing.T) {
	log.SetFlags(0)
	patch := monkey.Patch(os.Exit, func(code int) {
		panic(fmt.Sprintf("os.Exit called with: %v", code))
	})
	defer patch.Unpatch()
	for _, test := range exitErrHandlerTests {
		t.Run(test.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.SetOutput(buf)
			h := func() {
				main.ExitErrHandler(nil, test.err)
			}
			if test.exit == "" {
				h()
			} else {
				assert.PanicsWithValue(t, test.exit, h, "os.Exit was not called")
			}
			assert.Equal(t, test.out, buf.String())
		})
	}

}

package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"testing"

	"github.com/bouk/monkey"
	"github.com/codegangsta/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer"
	cmd "github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/magiconair/properties/assert"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
)

var createBuilderTets = []struct {
	name        string
	pipe        bool
	compression bool
	metadata    bool
	want        cmd.Builder
}{
	{"default", false, false, false, &cmd.ArgumentBuilder{
		Compressed:   false,
		WithMetadata: false,
	}},
	{"compressed", false, true, false, &cmd.ArgumentBuilder{
		Compressed:   true,
		WithMetadata: false,
	}},
	{"include", false, false, true, &cmd.ArgumentBuilder{
		Compressed:   false,
		WithMetadata: true,
	}},
	{"compressedInclude", false, true, true, &cmd.ArgumentBuilder{
		Compressed:   true,
		WithMetadata: true,
	}},
	{"pipe", true, false, false, &cmd.PipeBuilder{}},
}

func TestCreateBuilder(t *testing.T) {
	for _, test := range createBuilderTets {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, main.CreateBuilder(test.pipe, test.compression, test.metadata), test.want)
		})
	}
}

var createLoggerTests = []struct {
	name       string
	verbose    bool
	noDateTime bool
	flags      int
}{
	{"default", false, false, 3},
	{"verbose", true, false, 3},
	{"flags", false, true, 0},
}

func TestCreateLogger(t *testing.T) {
	for _, test := range createLoggerTests {
		t.Run(test.name, func(t *testing.T) {
			l, f, buf, err := createLogger(test.name, test.verbose, test.noDateTime)
			if err != nil {
				t.Error(err)
			}
			defer f.Close()
			defer syscall.Unlink(f.Name())

			l.Println(test.name)

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Errorf("failed to read log output: %v", err)
			}

			expect := fmt.Sprintf(".*%s.*", test.name)

			assert.Matches(t, string(b), expect)
			if test.verbose {
				assert.Matches(t, buf.String(), expect)
			}
			assert.Equal(t, l.Flags(), test.flags)

		})
	}
}

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
				tassert.PanicsWithValue(t, test.exit, h, "os.Exit was not called")
			}
			tassert.Equal(t, test.out, buf.String())
		})
	}

}

func createLogger(name string, verbose, noDateTime bool) (*log.Logger, *os.File, *bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	f, err := ioutil.TempFile("", name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	l, err := main.CreateLogger(f.Name(), verbose, buf, noDateTime)

	return l, f, buf, err
}

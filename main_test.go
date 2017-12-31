package main_test

import (
	"bytes"
	"flag"
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
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
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

var loggersTests = []struct {
	name   string
	config string
	err    string
}{
	{
		"noErrorFile",
		"",
		"failed creating error log: open : no such file or directory",
	},
	{
		"noOutFile",
		`[logs]
error = ./error.log
`,
		"failed creating info log: open : no such file or directory",
	},
	{
		"success",
		`[logs]
error = ./error.log
info = ./info.log
`,
		"",
	},
}

func TestLoggers(t *testing.T) {
	set := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	c := cli.NewContext(nil, set, nil)
	for _, test := range loggersTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.CreateFromString(test.config)
			outLog, errLog, err := main.Loggers(c, cfg)
			if len(test.err) > 0 {
				tassert.Equal(t, err.Error(), test.err)
			} else {
				tassert.Nil(t, err)
				tassert.NotNil(t, outLog)
				tassert.NotNil(t, errLog)
			}
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

var loadConfigurationTest = []struct{
	name string
	flags func(set *flag.FlagSet)
	err error
	url string
	queue string
}{
	{
		"default",
		func(set *flag.FlagSet) {
			set.String("configuration", "fixtures/default.conf", "")
		},
		nil,
		"",
		"test",
	},
	{
		"missingFlags",
		func(set *flag.FlagSet) {},
		cli.NewExitError("", 1),
		"",
		"",
	},
	{
		"missingFile",
		func(set *flag.FlagSet) {
			set.String("configuration", "/does/not/exits.conf", "")
		},
		fmt.Errorf("failed parsing configuration: open /does/not/exits.conf: no such file or directory"),
		"",
		"",
	},
	{
		"url",
		func(set *flag.FlagSet) {
			set.String("configuration", "fixtures/default.conf", "")
			set.String("url", "amqp://amqp.example.com:1234", "")
		},
		nil,
		"amqp://amqp.example.com:1234",
		"test",
	},
	{
		"queue",
		func(set *flag.FlagSet) {
			set.String("configuration", "fixtures/default.conf", "")
			set.String("queue-name", "someOtherQueue", "")
		},
		nil,
		"",
		"someOtherQueue",
	},
}

func TestLoadConfiguration(t *testing.T) {
	app := main.NewApp()
	app.Writer = &bytes.Buffer{}
	for _, test := range loadConfigurationTest {
		t.Run(test.name, func(t *testing.T) {
			set := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			test.flags(set)
			c := cli.NewContext(app, set, nil)
			cfg, err := main.LoadConfiguration(c)
			tassert.Equal(t, err, test.err)
			if test.err == nil {
				tassert.NotNil(t, cfg)
				tassert.Equal(t, cfg.RabbitMq.AmqpUrl, test.url)
				tassert.Equal(t, cfg.RabbitMq.Queue, test.queue)
			}
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

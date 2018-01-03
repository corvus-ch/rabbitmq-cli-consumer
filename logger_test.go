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

	"github.com/codegangsta/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	passert "github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/assert"
)

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

			passert.Matches(t, string(b), expect)
			if test.verbose {
				passert.Matches(t, buf.String(), expect)
			}
			assert.Equal(t, l.Flags(), test.flags)

		})
	}
}

var loggersTests = []struct {
	name    string
	config  string
	err     string
	verbose bool
}{
	{
		"noErrorFile",
		"",
		"failed creating error log: open : no such file or directory",
		false,
	},
	{
		"noOutFile",
		`[logs]
error = ./error.log
`,
		"failed creating info log: open : no such file or directory",
		false,
	},
	{
		"success",
		`[logs]
error = ./error.log
info = ./info.log
`,
		"",
		false,
	},
	{
		"noLogFiles",
		"",
		"",
		true,
	},
}

func TestLoggers(t *testing.T) {
	for _, test := range loggersTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.CreateFromString(test.config)
			set := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			set.Bool("verbose", test.verbose, "")
			c := cli.NewContext(nil, set, nil)
			outLog, errLog, err := main.Loggers(c, cfg)
			if len(test.err) > 0 {
				assert.Equal(t, err.Error(), test.err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, outLog)
				assert.NotNil(t, errLog)
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

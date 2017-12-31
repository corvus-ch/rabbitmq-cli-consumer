package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"testing"

	main "github.com/corvus-ch/rabbitmq-cli-consumer"
	cmd "github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/magiconair/properties/assert"
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

func createLogger(name string, verbose, noDateTime bool) (*log.Logger, *os.File, *bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	f, err := ioutil.TempFile("", name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	l, err := main.CreateLogger(f.Name(), verbose, buf, noDateTime)

	return l, f, buf, err
}

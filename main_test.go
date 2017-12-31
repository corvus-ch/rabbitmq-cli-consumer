package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"syscall"
	"testing"

	main "github.com/corvus-ch/rabbitmq-cli-consumer"
	consumerCommand "github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/magiconair/properties/assert"
)

func TestCreateBuilder(t *testing.T) {
	tests := []struct {
		name        string
		pipe        bool
		compression bool
		metadata    bool
		want        consumerCommand.Builder
	}{
		{"default", false, false, false, &consumerCommand.ArgumentBuilder{
			Compressed:   false,
			WithMetadata: false,
		}},
		{"compressed", false, true, false, &consumerCommand.ArgumentBuilder{
			Compressed:   true,
			WithMetadata: false,
		}},
		{"include", false, false, true, &consumerCommand.ArgumentBuilder{
			Compressed:   false,
			WithMetadata: true,
		}},
		{"compressedInclude", false, true, true, &consumerCommand.ArgumentBuilder{
			Compressed:   true,
			WithMetadata: true,
		}},
		{"pipe", true, false, false, &consumerCommand.PipeBuilder{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, main.CreateBuilder(test.pipe, test.compression, test.metadata), test.want)
		})
	}
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name       string
		verbose    bool
		noDateTime bool
		flags      int
	}{
		{"default", false, false, 3},
		{"verbose", true, false, 3},
		{"flags", false, true, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			f, err := ioutil.TempFile("", test.name)
			if err != nil {
				t.Errorf("failed to create temp file: %v", err)
			}
			defer f.Close()
			defer syscall.Unlink(f.Name())

			l, err := main.CreateLogger(f.Name(), test.verbose, buf, test.noDateTime)
			if err != nil {
				t.Errorf("failed to create logger: %v", err)
			}
			l.Println(test.name)

			f.Seek(0, 0)
			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Errorf("failed to read log output: ")
			}
			assert.Matches(t, string(b), fmt.Sprintf(".*%s.*", test.name))
			if test.verbose {
				assert.Matches(t, buf.String(), fmt.Sprintf(".*%s.*", test.name))
			}
			assert.Equal(t, l.Flags(), test.flags)

		})
	}
}

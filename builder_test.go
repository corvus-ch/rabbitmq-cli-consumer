package main_test

import (
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer"
	cmd "github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/stretchr/testify/assert"
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

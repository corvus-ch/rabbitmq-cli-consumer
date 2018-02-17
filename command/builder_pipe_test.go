package command_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/stretchr/testify/assert"
)

const emptyPropertiesString = "{\"properties\":{\"application_headers\":null,\"content_type\":\"\",\"content_encoding\":\"\",\"delivery_mode\":0,\"priority\":0,\"correlation_id\":\"\",\"reply_to\":\"\",\"expiration\":\"\",\"message_id\":\"\",\"timestamp\":\"0001-01-01T00:00:00Z\",\"type\":\"\",\"user_id\":\"\",\"app_id\":\"\"},\"delivery_info\":{\"message_count\":0,\"consumer_tag\":\"\",\"delivery_tag\":0,\"redelivered\":false,\"exchange\":\"\",\"routing_key\":\"\"}}"

var pipeBuilderGetCommandtests = []struct {
	name    string
	capture bool
}{
	{
		"default",
		false,
	},
	{
		"complex command",
		false,
	},
	{
		"capture",
		true,
	},
}

func TestPipeBuilder_GetCommand(t *testing.T) {
	for _, test := range pipeBuilderGetCommandtests {
		t.Run(test.name, func(t *testing.T) {
			b, ib, eb := createAndAssertBuilder(t, &command.PipeBuilder{}, test.name, test.capture)
			cmd := createAndAssertCommand(t, b, []byte(test.name))
			assert.Equal(t, strings.Split(test.name, " "), cmd.Args)
			assert.NotNil(t, cmd.Stdin)
			input, _ := ioutil.ReadAll(cmd.Stdin)
			assert.Equal(t, test.name, string(input))
			assert.NotNil(t, cmd.ExtraFiles)
			metadata, _ := ioutil.ReadAll(cmd.ExtraFiles[0])
			assert.Equal(t, emptyPropertiesString, string(metadata))
			assert.Equal(t, os.Environ(), cmd.Env)
			assertWriter(t, ib, cmd.Stdout, test.capture)
			assertWriter(t, eb, cmd.Stderr, test.capture)
		})
	}
}

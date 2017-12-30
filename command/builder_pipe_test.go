package command_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/metadata"
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
			outLog := log.New(&bytes.Buffer{}, "", 0)
			errLog := log.New(&bytes.Buffer{}, "", 0)
			b, err := command.NewBuilder(&command.PipeBuilder{}, test.name, test.capture, outLog, errLog)
			if err != nil {
				t.Errorf("failed to create builder: %v", err)
			}

			c, err := b.GetCommand(metadata.Properties{}, metadata.DeliveryInfo{}, []byte(test.name))
			if err != nil {
				t.Errorf("failed to create command: %v", err)
			}

			assert.IsType(t, &command.ExecCommand{}, c)

			cmd := c.Cmd()
			assert.Equal(t, strings.Split(test.name, " "), cmd.Args)
			assert.NotNil(t, cmd.Stdin)
			input, _ := ioutil.ReadAll(cmd.Stdin)
			assert.Equal(t, test.name, string(input))
			assert.NotNil(t, cmd.ExtraFiles)
			metadata, _ := ioutil.ReadAll(cmd.ExtraFiles[0])
			assert.Equal(t, emptyPropertiesString, string(metadata))
			assert.Equal(t, os.Environ(), cmd.Env)
			if test.capture {
				outW, ok := cmd.Stdout.(*command.LogWriter)
				if !ok {
					t.Errorf("expected STDOUT to be of type *command.LogWriter")
				}
				assert.Equal(t, outLog, outW.Logger)

				errW, ok := cmd.Stderr.(*command.LogWriter)
				if !ok {
					t.Errorf("expected STDERR to be of type *command.LogWriter")
				}
				assert.Equal(t, errLog, errW.Logger)
			} else {
				assert.Nil(t, cmd.Stdout)
				assert.Nil(t, cmd.Stderr)
			}
		})
	}
}

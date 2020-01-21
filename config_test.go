package main_test

import (
	"bytes"
	"flag"
	"fmt"
	"testing"

	"github.com/urfave/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer"
	"github.com/stretchr/testify/assert"
)

var loadConfigurationTest = []struct {
	name  string
	flags func(set *flag.FlagSet)
	err   error
	url   string
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
			assert.Equal(t, err, test.err)
			if test.err == nil {
				assert.NotNil(t, cfg)
				assert.Equal(t, cfg.RabbitMq.AmqpUrl, test.url)
				assert.Equal(t, cfg.RabbitMq.Queue, test.queue)
			}
		})
	}
}

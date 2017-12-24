package consumer_test

import (
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/magiconair/properties/assert"
)

var uriTests = []struct {
	config string
	uri    string
}{
	{
		`[rabbitmq]
host = localhost
vhost = /vhost`,
		"amqp://localhost/vhost",
	},
	{
		`[rabbitmq]
host = localhost
vhost = vhost`,
		"amqp://localhost/vhost",
	},
	{
		`[rabbitmq]
host = 127.0.0.1`,
		"amqp://127.0.0.1",
	},
	{
		`[rabbitmq]
host = localhost
port = 1234`,
		"amqp://localhost:1234",
	},
	{
		`[rabbitmq]
password = seecret
host = localhost`,
		"amqp://localhost",
	},
	{
		`[rabbitmq]
username = richard
host = localhost`,
		"amqp://richard@localhost",
	},
	{
		`[rabbitmq]
username = richard
password = seecret
host = localhost`,
		"amqp://richard:seecret@localhost",
	},
	{
		`[rabbitmq]
username = richard
password = my@:secr%t
host = localhost`,
		"amqp://richard:my%40%3Asecr%25t@localhost",
	},
	{
		`[rabbitmq]
username = richard
password = my@:secr%t
host = example.com
port = 1234
vhost = myhost`,
		"amqp://richard:my%40%3Asecr%25t@example.com:1234/myhost",
	},
}

func TestParseAndEscapeUri(t *testing.T) {
	for _, data := range uriTests {
		cfg, err := config.CreateFromString(data.config)
		if err != nil {
			t.Errorf("failed to load config: %v", err)
		}
		assert.Equal(t, data.uri, consumer.AmqpURI(cfg))
	}
}

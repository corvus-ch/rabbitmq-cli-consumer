package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/gcfg.v1"
)

type Config struct {
	RabbitMq struct {
		AmqpUrl      string
		Host         string
		Username     string
		Password     string
		Port         string
		Vhost        string
		Queue        string
		Compression  bool
		Onfailure    int
		Stricfailure bool
	}
	Prefetch struct {
		Count  int
		Global bool
	}
	QueueSettings struct {
		Routingkey           []string
		MessageTTL           int
		DeadLetterExchange   string
		DeadLetterRoutingKey string
		Priority             int
		Nodeclare            bool
	}
	Exchange struct {
		Name       string
		Autodelete bool
		Type       string
		Durable    bool
	}
	Logs struct {
		Error      string
		Info       string
		NoDateTime bool
		Verbose    bool
	}
}

func (c *Config) AmqpUrl() string {
	if len(c.RabbitMq.AmqpUrl) > 0 {
		return c.RabbitMq.AmqpUrl
	}

	host := c.RabbitMq.Host
	if len(c.RabbitMq.Port) > 0 {
		host = fmt.Sprintf("%s:%s", host, c.RabbitMq.Port)
	}

	uri := url.URL{
		Scheme: "amqp",
		Host:   host,
		Path:   c.RabbitMq.Vhost,
	}

	if len(c.RabbitMq.Username) > 0 {
		if len(c.RabbitMq.Password) > 0 {
			uri.User = url.UserPassword(c.RabbitMq.Username, c.RabbitMq.Password)
		} else {
			uri.User = url.User(c.RabbitMq.Username)
		}
	}

	c.RabbitMq.AmqpUrl = uri.String()

	return c.RabbitMq.AmqpUrl
}

// QueueName returns the name of toe queue to bind with.
func (c Config) QueueName() string {
	return c.RabbitMq.Queue
}

// MustDeclareQueue return if the consumer should declare the queue or if the queue is expected to be already declared.
func (c Config) MustDeclareQueue() bool {
	return !c.QueueSettings.Nodeclare
}

// HasExchange checks if an exchange is configured.
func (c Config) HasExchange() bool {
	return c.Exchange.Name != ""
}

// ExchangeName returns the name of the configured exchange.
func (c Config) ExchangeName() string {
	return transformToStringValue(c.Exchange.Name)
}

// ExchangeType checks the configuration and returns the appropriate exchange type.
func (c Config) ExchangeType() string {
	// Check for missing exchange settings to preserve BC
	if "" == c.Exchange.Name && "" == c.Exchange.Type && !c.Exchange.Durable && !c.Exchange.Autodelete {
		return "direct"
	}

	return c.Exchange.Type
}

// ExchangeIsDurable returns whether the exchange should be durable or not.
func (c Config) ExchangeIsDurable() bool {
	return c.Exchange.Durable
}

// ExchangeIsAutoDelete return whether the exchange should be auto deleted or not.
func (c Config) ExchangeIsAutoDelete() bool {
	return c.Exchange.Autodelete
}

// PrefetchCount returns the configured prefetch count of the QoS settings.
func (c Config) PrefetchCount() int {
	// Attempt to preserve BC here
	if c.Prefetch.Count == 0 {
		return 3
	}

	return c.Prefetch.Count
}

// PrefetchIsGlobal returns if the prefetch count is defined globally for all consumers or locally for just each single
// consumer.
func (c Config) PrefetchIsGlobal() bool {
	return c.Prefetch.Global
}

// HasMessageTTL checks if a message TTL is configured.
func (c Config) HasMessageTTL() bool {
	return c.QueueSettings.MessageTTL > 0
}

// MessageTTL returns the configured message TTL.
func (c Config) MessageTTL() int32 {
	return int32(c.QueueSettings.MessageTTL)
}

// RoutingKeys returns the configured keys for message routing.
func (c Config) RoutingKeys() []string {
	if len(c.QueueSettings.Routingkey) < 1 {
		return []string{""}
	}

	return transformArrayOfStringToStringValue(c.QueueSettings.Routingkey)
}

// HasDeadLetterExchange checks if a dead letter exchange is configured.
func (c Config) HasDeadLetterExchange() bool {
	return c.QueueSettings.DeadLetterExchange != ""
}

// DeadLetterExchange returns the configured dead letter exchange name.
func (c Config) DeadLetterExchange() string {
	return transformToStringValue(c.QueueSettings.DeadLetterExchange)
}

// HasDeadLetterRouting checks if a dead letter routing key is configured.
func (c Config) HasDeadLetterRouting() bool {
	return c.QueueSettings.DeadLetterRoutingKey != ""
}

// DeadLetterRoutingKey returns the configured key for the dead letter routing.
func (c Config) DeadLetterRoutingKey() string {
	return transformToStringValue(c.QueueSettings.DeadLetterRoutingKey)
}

// HasPriority checks if priority is configured
func (c Config) HasPriority() bool {
	return c.QueueSettings.Priority > 0
}

// Priority returns the priority
func (c Config) Priority() int32 {
	return int32(c.QueueSettings.Priority)
}

// IsVerbose checks if verbose logging is enabled.
func (c Config) IsVerbose() bool {
	return c.Logs.Verbose
}

// WithDateTime checks if log entries should be logged with date and time.
func (c Config) WithDateTime() bool {
	return !c.Logs.NoDateTime
}

// ConsumerTag returns the tag used to identify the consumer.
func (c Config) ConsumerTag() string {
	if v, set := os.LookupEnv("GO_WANT_HELPER_PROCESS"); set && v == "1" {
		return ""
	}

	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return fmt.Sprintf("ctag-%s-%d@%s", os.Args[0], os.Getpid(), host)
}

// LoadAndParse creates a new instance of config by parsing the content of teh given file.
func LoadAndParse(location string) (*Config, error) {
	if !filepath.IsAbs(location) {
		location, err := filepath.Abs(location)

		if err != nil {
			return nil, err
		}

		location = location
	}

	cfg := Config{}
	if err := gcfg.ReadFileInto(&cfg, location); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func CreateFromString(data string) (*Config, error) {
	cfg := &Config{}
	if err := gcfg.ReadStringInto(cfg, data); err != nil {
		return nil, err
	}

	return cfg, nil
}

func transformToStringValue(val string) string {
	if val == "<empty>" {
		return ""
	}

	return val
}

func transformArrayOfStringToStringValue(iterable []string) []string {
	var ret []string

	for _, str := range iterable {
		ret = append(ret, transformToStringValue(str))
	}

	return ret
}

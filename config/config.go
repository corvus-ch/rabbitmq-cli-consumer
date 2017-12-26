package config

import (
	"path/filepath"

	"gopkg.in/gcfg.v1"
	"net/url"
	"fmt"
)

type Config struct {
	RabbitMq struct {
		AmqpUrl     string
		Host        string
		Username    string
		Password    string
		Port        string
		Vhost       string
		Queue       string
		Compression bool
		Onfailure   int
	}
	Prefetch struct {
		Count  int
		Global bool
	}
	QueueSettings struct {
		Routingkey           string
		MessageTTL           int
		DeadLetterExchange   string
		DeadLetterRoutingKey string
	}
	Exchange struct {
		Name       string
		Autodelete bool
		Type       string
		Durable    bool
	}
	Logs struct {
		Error string
		Info  string
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

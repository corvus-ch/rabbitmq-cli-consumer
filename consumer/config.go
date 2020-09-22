package consumer

import (
	"crypto/tls"
)

// Config defines the interface to present configurations to the consumer.
type Config interface {
	AmqpUrl() string
	TLSConfig() *tls.Config
	ConsumerTag() string
	DeadLetterExchange() string
	DeadLetterRoutingKey() string
	ExchangeIsAutoDelete() bool
	ExchangeIsDurable() bool
	ExchangeName() string
	ExchangeType() string
	HasDeadLetterExchange() bool
	HasDeadLetterRouting() bool
	HasExchange() bool
	HasMessageTTL() bool
	HasPriority() bool
	MessageTTL() int32
	MustDeclareQueue() bool
	PrefetchCount() int
	PrefetchIsGlobal() bool
	Priority() int32
	QueueName() string
	RoutingKeys() []string
	QueueIsDurable() bool
	QueueIsExclusive() bool
	QueueIsAutoDelete() bool
	QueueIsNoWait() bool
}

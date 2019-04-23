package consumer

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/streadway/amqp"
)

type amqpAttributes struct {
	d    amqp.Delivery
	json []byte
}

type jsonAttributes struct {
	AppId           string                 `json:"app_id"`
	ConsumerTag     string                 `json:"consumer_tag"`
	ContentEncoding string                 `json:"content_encoding"`
	ContentType     string                 `json:"content_type"`
	CorrelationID   string                 `json:"correlation_id"`
	DeliveryMode    uint8                  `json:"delivery_mode"`
	DeliveryTag     uint64                 `json:"delivery_tag"`
	Exchange        string                 `json:"exchange"`
	Expiration      string                 `json:"expiration"`
	Headers         map[string]interface{} `json:"application_headers"`
	MessageID       string                 `json:"message_id"`
	MsgType         string                 `json:"type"`
	Priority        uint8                  `json:"priority"`
	Redelivered     bool                   `json:"redelivered"`
	ReplyTo         string                 `json:"reply_to"`
	RoutingKey      string                 `json:"routing_key"`
	Timestamp       time.Time              `json:"timestamp"`
	UserID          string                 `json:"user_id"`
}

func newAttributes(d amqp.Delivery) (worker.Attributes, error) {
	data, err := json.Marshal(jsonAttributes{
		AppId:           d.AppId,
		ConsumerTag:     d.ConsumerTag,
		ContentEncoding: d.ContentEncoding,
		ContentType:     d.ContentType,
		CorrelationID:   d.CorrelationId,
		DeliveryMode:    d.DeliveryMode,
		DeliveryTag:     d.DeliveryTag,
		Exchange:        d.Exchange,
		Expiration:      d.Expiration,
		Headers:         d.Headers,
		MessageID:       d.MessageId,
		MsgType:         d.Type,
		Priority:        d.Priority,
		Redelivered:     d.Redelivered,
		ReplyTo:         d.ReplyTo,
		RoutingKey:      d.RoutingKey,
		Timestamp:       d.Timestamp,
		UserID:          d.UserId,
	})
	if err != nil {
		return nil, err
	}

	return &amqpAttributes{d: d, json: data}, nil
}

func (a *amqpAttributes) AppId() string                   { return a.d.AppId }
func (a *amqpAttributes) ConsumerTag() string             { return a.d.ConsumerTag }
func (a *amqpAttributes) ContentEncoding() string         { return a.d.ContentEncoding }
func (a *amqpAttributes) ContentType() string             { return a.d.ContentType }
func (a *amqpAttributes) CorrelationId() string           { return a.d.CorrelationId }
func (a *amqpAttributes) DeliveryMode() uint8             { return a.d.DeliveryMode }
func (a *amqpAttributes) DeliveryTag() uint64             { return a.d.DeliveryTag }
func (a *amqpAttributes) Exchange() string                { return a.d.Exchange }
func (a *amqpAttributes) Expiration() string              { return a.d.Expiration }
func (a *amqpAttributes) Headers() map[string]interface{} { return a.d.Headers }
func (a *amqpAttributes) MessageId() string               { return a.d.MessageId }
func (a *amqpAttributes) Priority() uint8                 { return a.d.Priority }
func (a *amqpAttributes) Redelivered() bool               { return a.d.Redelivered }
func (a *amqpAttributes) ReplyTo() string                 { return a.d.ReplyTo }
func (a *amqpAttributes) RoutingKey() string              { return a.d.RoutingKey }
func (a *amqpAttributes) Timestamp() time.Time            { return a.d.Timestamp }
func (a *amqpAttributes) Type() string                    { return a.d.Type }
func (a *amqpAttributes) UserId() string                  { return a.d.UserId }
func (a *amqpAttributes) JSON() io.Reader                 { return bytes.NewBuffer(a.json) }

package delivery

import (
	"time"

	"github.com/streadway/amqp"
)

// Properties represents the properties of an AMQP message.
type Properties struct {
	Headers         amqp.Table `json:"application_headers"`
	ContentType     string     `json:"content_type"`
	ContentEncoding string     `json:"content_encoding"`
	DeliveryMode    uint8      `json:"delivery_mode"`
	Priority        uint8      `json:"priority"`
	CorrelationID   string     `json:"correlation_id"`
	ReplyTo         string     `json:"reply_to"`
	Expiration      string     `json:"expiration"`
	MessageID       string     `json:"message_id"`
	Timestamp       time.Time  `json:"timestamp"`
	Type            string     `json:"type"`
	UserID          string     `json:"user_id"`
	AppID           string     `json:"app_id"`
}

// NewProperties creates a new properties struct from the AMQP message.
func NewProperties(d amqp.Delivery) Properties {
	return Properties{
		Headers:         d.Headers,
		ContentType:     d.ContentType,
		ContentEncoding: d.ContentEncoding,
		DeliveryMode:    d.DeliveryMode,
		Priority:        d.Priority,
		CorrelationID:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Expiration:      d.Expiration,
		MessageID:       d.MessageId,
		Timestamp:       d.Timestamp,
		Type:            d.Type,
		AppID:           d.AppId,
		UserID:          d.UserId,
	}
}

package metadata

import (
	"time"

	"github.com/streadway/amqp"
)

type Properties struct {
	Headers         amqp.Table `json:"application_headers"`
	ContentType     string     `json:"content_type"`
	ContentEncoding string     `json:"content_encoding"`
	DeliveryMode    uint8      `json:"delivery_mode"`
	Priority        uint8      `json:"priority"`
	CorrelationId   string     `json:"correlation_id"`
	ReplyTo         string     `json:"reply_to"`
	Expiration      string     `json:"expiration"`
	MessageId       string     `json:"message_id"`
	Timestamp       time.Time  `json:"timestamp"`
	Type            string     `json:"type"`
	UserId          string     `json:"user_id"`
	AppId           string     `json:"app_id"`
}

func NewProperties(d amqp.Delivery) Properties {
	return Properties{
		Headers:         d.Headers,
		ContentType:     d.ContentType,
		ContentEncoding: d.ContentEncoding,
		DeliveryMode:    d.DeliveryMode,
		Priority:        d.Priority,
		CorrelationId:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Expiration:      d.Expiration,
		MessageId:       d.MessageId,
		Timestamp:       d.Timestamp,
		Type:            d.Type,
		AppId:           d.AppId,
		UserId:          d.UserId,
	}
}

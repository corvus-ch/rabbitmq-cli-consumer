// Package exec holds a worker implementation which runs a worker Script trough
// FastCGI.
package fastcgi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/tomasen/fcgi_client"
)

type Properties struct {
	Headers         map[string]interface{} `json:"application_headers"`
	ContentType     string                 `json:"content_type"`
	ContentEncoding string                 `json:"content_encoding"`
	DeliveryMode    uint8                  `json:"delivery_mode"`
	Priority        uint8                  `json:"priority"`
	CorrelationID   string                 `json:"correlation_id"`
	ReplyTo         string                 `json:"reply_to"`
	Expiration      string                 `json:"expiration"`
	MessageID       string                 `json:"message_id"`
	Timestamp       time.Time              `json:"timestamp"`
	MsgType         string                 `json:"type"`
	UserID          string                 `json:"user_id"`
	AppId           string                 `json:"app_id"`
}

type DeliveryInfo struct {
	ConsumerTag     string                 `json:"consumer_tag"`
	DeliveryTag     uint64                 `json:"delivery_tag"`
	Redelivered     bool                   `json:"redelivered"`
	Exchange        string                 `json:"exchange"`
	RoutingKey      string                 `json:"routing_key"`
}

type jsonPayload struct {
	Properties     Properties              `json:"properties"`
	DeliveryInfo   DeliveryInfo            `json:"delivery_info"`
	Body           []byte                  `json:"body"`
}

type Processor struct {
	Network      string
	Address      string
	Script       string
	Method       string
	Uri          string
	Include      bool
	Log          logr.Logger
}

type Config interface {
	Network() string
	Address() string
	Script()  string
	Method()  string
	Uri()     string
	Include() bool
}

func New(config Config, log logr.Logger) (worker.Process, error) {
	p := &Processor{
		Network: "tcp",
		Address: "127.0.0.1:9000",
		Script:  "index.php",
		Method:  "POST",
		Uri:     "/",
		Log:     log,
	}

	if config.Network() != "" {
		p.Network = config.Network()
	}

	if config.Address() != "" {
		p.Address = config.Address()
	}

	if config.Script() != "" {
		p.Script = config.Script()
	}

	if config.Method() != "" {
		p.Method = strings.ToUpper(config.Method())
	}

	if config.Uri() != "" {
		p.Uri = config.Uri()
	}

	p.Include = config.Include()

	return p.Process, nil
}

func (p *Processor) Process(attributes worker.Attributes, payload io.Reader, log logr.Logger) (worker.Acknowledgment, error) {
	log.Info("Processing message...")
	defer log.Info("Processed!")

	fcgi, err := fcgiclient.Dial(p.Network, p.Address)

	if err != nil {
		p.Log.Errorf("Failed to initiate connexion to fast-cgi.\nError: %s\n", err)

		return worker.Requeue, err
	}

	toSend, err := p.newPayload(payload)

	if err != nil {
		p.Log.Errorf("Failed to make payload.\nError: %s\n", err)

		return worker.Requeue, err
	}

	env := make(map[string]string)
	env["REQUEST_METHOD"] = p.Method
	env["REQUEST_URI"] = p.Uri
	env["SCRIPT_FILENAME"] = p.Script
	env["CONTENT_LENGTH"] = fmt.Sprint(len(toSend))

	if p.Include == true {
		env["HTTP_RMQ_METADATA"], _ = p.newJsonMetadata(attributes)
	}


	response, errResp := fcgi.Request(env, bytes.NewReader(append(toSend, 4)))

	fcgi.Close()

	if errResp != nil {
		p.Log.Errorf("Failed to make payload.\nError: %s\n", err)

		return worker.Requeue, errResp
	}

	return p.acknowledgement(response), nil
}

func (p *Processor) acknowledgement(response *http.Response) worker.Acknowledgment {
	if response.StatusCode / 100 == 2 {
		return worker.Ack
	}

	if response.StatusCode == 449 {
		return worker.Requeue
	}

	return worker.Reject
}

func (p *Processor) newPayload(payload io.Reader) ([]byte, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(payload)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *Processor) newJsonMetadata(attributes worker.Attributes) (string, error) {
	data, err := json.Marshal(jsonPayload{
		Properties: Properties {
			Headers: attributes.Headers(),
			ContentType: attributes.ContentType(),
			ContentEncoding: attributes.ContentEncoding(),
			DeliveryMode: attributes.DeliveryMode(),
			Priority: attributes.Priority(),
			CorrelationID: attributes.CorrelationId(),
			ReplyTo: attributes.ReplyTo(),
			Expiration: attributes.Expiration(),
			MessageID: attributes.MessageId(),
			Timestamp: attributes.Timestamp(),
			MsgType: attributes.Type(),
			UserID: attributes.UserId(),
			AppId: attributes.AppId(),
		},
		DeliveryInfo: DeliveryInfo {
			ConsumerTag: attributes.ConsumerTag(),
			DeliveryTag: attributes.DeliveryTag(),
			Redelivered: attributes.Redelivered(),
			Exchange: attributes.Exchange(),
			RoutingKey: attributes.RoutingKey(),
		},
	})

	if err != nil {
		return "", err
	}

	return string(data), nil
}

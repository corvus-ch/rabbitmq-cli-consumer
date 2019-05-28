// Package exec holds a worker implementation which runs a worker script trough
// FastCGI.
package fastcgi

import (
	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
)

type Config interface {
	// TODO add functions as needed.
}

func New(cfg Config, log logr.Logger) (worker.Process, error) {
	// TODO implement.
	// Add parameters for logger, config etc. as needed.
	return nil, nil
}

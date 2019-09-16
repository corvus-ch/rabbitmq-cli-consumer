package factory

import (
	"fmt"

	"github.com/bketelsen/logr"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker/exec"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker/fastcgi"
)

const Executable string = "executable"
const Fastcgi string = "fast-cgi"

func NewProcess(config *config.Config, logger logr.Logger) (worker.Process, error) {
	switch config.Worker() {
	case Executable:
		// TODO Make reject/requeue conditions configurable.
		return exec.New(config.Executable(), []int{}, config.Output(), logger), nil
	case Fastcgi:
		fastcgiProcess, error := fastcgi.New(config, logger)
		if error != nil {
			return nil, fmt.Errorf("issue while instantiating fast-cgi processor", error)
		}

		return fastcgiProcess, nil
	default:
		return nil, fmt.Errorf("no worker was choosen, must be %s or %s", Executable, Fastcgi)
	}
}

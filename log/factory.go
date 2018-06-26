package log

import (
	"fmt"
	"io"
	stdlog "log"
	"os"

	"github.com/bketelsen/logr"
	log "github.com/corvus-ch/logr/std"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
)

// NewFromConfig crates a logger according to the given config.
func NewFromConfig(cfg *config.Config) (logr.Logger, io.Writer, io.Writer, error) {
	errW, err := newWriter(cfg.Logs.Error, cfg.IsVerbose(), os.Stderr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed creating error log: %s", err)
	}

	outW, err := newWriter(cfg.Logs.Info, cfg.IsVerbose(), os.Stdout)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed creating info log: %s", err)
	}

	infL := stdlog.New(outW, "", flag(cfg.WithDateTime()))
	errL := stdlog.New(errW, "", flag(cfg.WithDateTime()))

	return log.New(0, errL, infL), outW, errW, nil
}

// newWriter creates a new writer for the given file.
// If verbose is set to true, in addition to the file, the logger will also write to writer passed as the out argument.
func newWriter(filename string, verbose bool, out io.Writer) (io.Writer, error) {
	writers := make([]io.Writer, 0)
	if len(filename) > 0 || !verbose {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

		if err != nil {
			return nil, err
		}

		writers = append(writers, file)
	}

	if verbose {
		writers = append(writers, out)
	}

	return io.MultiWriter(writers...), nil
}

func flag(dateTime bool) int {
	if dateTime {
		return stdlog.LstdFlags
	}

	return 0
}

package command

import (
	"io"
	"log"
)

type LogWriter struct {
	io.Writer
	logger *log.Logger
}

func NewLogWriter(l *log.Logger) *LogWriter {
	lw := &LogWriter{}
	lw.logger = l

	return lw
}

func (lw LogWriter) Write(b []byte) (n int, err error) {
	flags := lw.logger.Flags()

	lw.logger.SetFlags(0)
	lw.logger.Print(string(b))
	lw.logger.SetFlags(flags)

	return len(b), nil
}

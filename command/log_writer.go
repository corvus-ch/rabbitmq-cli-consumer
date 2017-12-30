package command

import (
	"io"
	"log"
)

type LogWriter struct {
	io.Writer
	Logger *log.Logger
}

func NewLogWriter(l *log.Logger) *LogWriter {
	lw := &LogWriter{}
	lw.Logger = l

	return lw
}

func (lw LogWriter) Write(b []byte) (n int, err error) {
	flags := lw.Logger.Flags()

	lw.Logger.SetFlags(0)
	lw.Logger.Print(string(b))
	lw.Logger.SetFlags(flags)

	return len(b), nil
}

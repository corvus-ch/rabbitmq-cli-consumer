package log

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/stretchr/testify/assert"
)

var createLoggerTests = []struct {
	name                string
	verbose             bool
	file                string
	expectFileContent   string
	expectBufferContent string
}{
	{"default", false, "default", "default", ""},
	{"verbose", true, "verbose", "verbose", "verbose"},
	{"noFile", true, "", "", "noFile"},
}

func TestCreateLoggerWriter(t *testing.T) {
	for _, test := range createLoggerTests {
		t.Run(test.name, func(t *testing.T) {
			w, f, buf, err := createWriter(test.file, test.verbose)
			if err != nil {
				t.Error(err)
			}
			defer f.Close()
			defer syscall.Unlink(f.Name())

			w.Write([]byte(test.name))

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Errorf("failed to read log output: %v", err)
			}

			assert.Equal(t, test.expectFileContent, string(b))
			assert.Equal(t, test.expectBufferContent, buf.String())
		})
	}
}

var loggersTests = []struct {
	name   string
	config string
	err    error
	flag   int
}{
	{
		"noErrorFile",
		"",
		fmt.Errorf("failed creating error log: open : no such file or directory"),
		log.Ldate | log.Ltime,
	},
	{
		"noOutFile",
		`[logs]
error = ./error.log
`,
		fmt.Errorf("failed creating info log: open : no such file or directory"),
		log.Ldate | log.Ltime,
	},
	{
		"success",
		`[logs]
error = ./error.log
info = ./info.log
`,
		nil,
		log.Ldate | log.Ltime,
	},
	{
		"noLogFiles",
		`[logs]
verbose = On
`,
		nil,
		log.Ldate | log.Ltime,
	},
	{
		"noDateTime",
		`[logs]
error = ./error.log
info = ./info.log
nodatetime = On
`,
		nil,
		0,
	},
	{
		"noLogFilesNoDateTime",
		`[logs]
verbose = On
nodatetime = On
`,
		nil,
		0,
	},
}

func TestLoggers(t *testing.T) {
	for _, test := range loggersTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.CreateFromString(test.config)
			l, _, _, err := NewFromConfig(cfg)
			assert.Equal(t, test.err, err)
			if l != nil {
				ll := reflect.ValueOf(l).Elem().FieldByName("loggers")
				assert.Equal(t, ll.Len(), 2)
				assert.Equal(t, int64(test.flag), reflect.Indirect(ll.Index(0)).FieldByName("flag").Int())
				assert.Equal(t, int64(test.flag), reflect.Indirect(ll.Index(1)).FieldByName("flag").Int())
			}
		})
	}
}

func createWriter(name string, verbose bool) (io.Writer, *os.File, *bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	f, err := ioutil.TempFile("", name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	if len(name) > 0 {
		name = f.Name()
	}

	w, err := newWriter(name, verbose, buf)

	return w, f, buf, err
}

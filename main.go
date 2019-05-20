package main

import (
	"context"
	"fmt"
	"github.com/corvus-ch/rabbitmq-cli-consumer/worker/exec"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bketelsen/logr"
	"github.com/codegangsta/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer/collector"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// flags is the list of global flags known to the application.
var flags []cli.Flag = []cli.Flag{
	cli.StringFlag{
		Name:   "url, u",
		Usage:  "Connect with RabbitMQ using `URL`",
		EnvVar: "AMQP_URL",
	},
	cli.StringFlag{
		Name:  "executable, e",
		Usage: "Location of executable",
	},
	cli.StringFlag{
		Name:  "configuration, c",
		Usage: "Location of configuration file",
	},
	cli.BoolFlag{
		Name:  "output, o",
		Usage: "Enable logging of output from executable",
	},
	cli.BoolFlag{
		Name:  "verbose, V",
		Usage: "Enable verbose mode (logs to stdout and stderr)",
	},
	cli.BoolFlag{
		Name:  "pipe, p",
		Usage: "Pipe the message via STDIN instead of passing it as an argument. The message metadata will be passed as JSON via fd3.",
	},
	cli.BoolFlag{
		Name:  "include, i",
		Usage: "Include metadata. Passes message as JSON data including headers, properties and message body. This flag will be ignored when `-pipe` is used.",
	},
	cli.BoolFlag{
		Name:  "strict-exit-code",
		Usage: "Strict exit code processing will rise a fatal error if exit code is different from allowed onces.",
	},
	cli.StringFlag{
		Name:  "queue-name, q",
		Usage: "Optional queue name to which can be passed in, without needing to define it in config, if set will override config queue name",
	},
	cli.BoolFlag{
		Name:  "no-datetime",
		Usage: "prevents the output of date and time in the logs.",
	},
	cli.BoolFlag{
		Name:  "no-declare",
		Usage: "prevents the queue from being declared.",
	},
	cli.BoolFlag{
		Name:  "metrics, m",
		Usage: "enables metric to be exposed.",
	},
	cli.StringFlag{
		Name:  "web.listen-address",
		Usage: "Address on which to expose metrics and web interface.",
		Value: ":9566",
	},
	cli.StringFlag{
		Name:  "web.telemetry-path",
		Usage: "Path under which to expose metrics.",
		Value: "/metrics",
	},
}

var ll logr.Logger

func main() {
	NewApp().Run(os.Args)
}

// NewApp creates a new application instance with just one single action.
func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "rabbitmq-cli-consumer"
	app.Usage = "Consume RabbitMQ easily to any cli program"
	app.Authors = []cli.Author{
		{"Richard van den Brand", "richard@vandenbrand.org"},
		{"Christian Häusler", "haeusler.christian@mac.com"},
	}
	app.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
	app.Flags = flags
	app.Action = Action
	app.ExitErrHandler = ExitErrHandler

	return app
}

// Action is the function being run when the application gets executed.
func Action(c *cli.Context) error {
	cfg, err := LoadConfiguration(c)
	if err != nil {
		return err
	}

	l, _, _, err := log.NewFromConfig(cfg)
	if err != nil {
		return err
	}
	ll = l

	// TODO Make reject/requeue conditions configurable.
	p := exec.New(c.String("executable"), []int{}, c.Bool("output"), ll)
	client, err := consumer.NewFromConfig(cfg, p, l)
	if err != nil {
		return err
	}
	defer client.Close()

	errs := make(chan error)

	if c.Bool("metrics") {
		ll.Infof("Registering metrics server at %v", c.String("web.listen-address"))
		go func() {
			errs <- setupAndServeMetrics(c.String("web.listen-address"), c.String("web.telemetry-path"))
		}()
	} else {
		ll.Infof("Metrics disabled.")
	}

	go func() {
		errs <- consume(client, l)
	}()

	return <-errs
}

func setupAndServeMetrics(addr string, path string) error {
	srv := &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	prometheus.MustRegister(collector.ProcessCounter)
	prometheus.MustRegister(collector.ProcessDuration)
	prometheus.MustRegister(collector.MessageDuration)

	http.Handle(path, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			 <head><title>rabbitmq-cli-consumer</title></head>
			 <body>
			 <h1>rabbitmq-cli-consumer</h1>
			 <p><a href='` + path + `'>Metrics</a></p>
			 </body>
			 </html>`))
	})

	if err := srv.ListenAndServe(); err != nil {
		return errors.Wrap(err, "failed to serve metrics")
	}

	return nil
}

func consume(client *consumer.Consumer, l logr.Logger) error {
	done := make(chan error)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		done <- client.Consume(ctx)
	}()

	select {
	case <-sig:
		l.Info("Cancel consumption of messages.")
		cancel()
		return checkConsumeError(<-done)

	case err := <-done:
		return checkConsumeError(err)
	}
}

func checkConsumeError(err error) error {
	switch err.(type) {
	case *amqp.Error:
		if strings.Contains(err.Error(), "Exception (320) Reason:") {
			return cli.NewExitError(fmt.Sprintf("connection closed: %v", err.(*amqp.Error).Reason), 10)
		}
		return err

	default:
		if strings.Contains(err.Error(), "failed to acknowledge message") {
			return cli.NewExitError(err, 11)
		}
		return err
	}
}

// ExitErrHandler is a global error handler registered with the application.
func ExitErrHandler(_ *cli.Context, err error) {
	if err == nil {
		return
	}

	code := 1

	if err.Error() != "" {
		if ll != nil {
			ll.Error(err)
		} else {
			stdlog.Printf("%+v\n", err)
		}
	}

	if exitErr, ok := err.(cli.ExitCoder); ok {
		code = exitErr.ExitCode()
	}

	os.Exit(code)
}

// LoadConfiguration checks the configuration flags, loads the config from file and updates the config according the flags.
func LoadConfiguration(c *cli.Context) (*config.Config, error) {
	file := c.String("configuration")
	url := c.String("url")
	queue := c.String("queue-name")

	if file == "" && url == "" && queue == "" && c.String("executable") == "" {
		cli.ShowAppHelp(c)
		return nil, cli.NewExitError("", 1)
	}

	cfg, err := configuration(file)
	if err != nil {
		return nil, fmt.Errorf("failed parsing configuration: %s", err)
	}

	if len(url) > 0 {
		cfg.RabbitMq.AmqpUrl = url
	}

	if queue != "" {
		cfg.RabbitMq.Queue = queue
	}

	if c.IsSet("no-datetime") {
		cfg.Logs.NoDateTime = c.Bool("no-datetime")
	}

	if c.IsSet("verbose") {
		cfg.Logs.Verbose = c.Bool("verbose")
	}

	if c.IsSet("strict-exit-code") {
		cfg.RabbitMq.Stricfailure = c.Bool("strict-exit-code")
	}

	if c.IsSet("no-declare") {
		cfg.QueueSettings.Nodeclare = c.Bool("no-declare")
	}

	return cfg, nil
}

func configuration(file string) (*config.Config, error) {
	if file == "" {
		return config.CreateFromString("")
	}

	return config.LoadAndParse(file)
}

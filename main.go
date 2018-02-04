package main

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/corvus-ch/rabbitmq-cli-consumer/command"
	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/corvus-ch/rabbitmq-cli-consumer/log"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// flags is the list of global flags known to the application.
var flags = []cli.Flag{
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
}

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

	l, infW, errW, err := log.NewFromConfig(cfg)
	if err != nil {
		return err
	}

	b := CreateBuilder(c.Bool("pipe"), cfg.RabbitMq.Compression, c.Bool("include"))
	builder, err := command.NewBuilder(b, c.String("executable"), c.Bool("output"), l, infW, errW)
	if err != nil {
		return fmt.Errorf("failed to create command builder: %v", err)
	}

	ack := consumer.NewAcknowledger(c.Bool("strict-exit-code"), cfg.RabbitMq.Onfailure)

	client, err := consumer.New(cfg, builder, ack, l)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed creating consumer: %s", err), 1)
	}

	client.Consume()

	return nil
}

// ExitErrHandler is a global error handler registered with the application.
func ExitErrHandler(_ *cli.Context, err error) {
	if err == nil {
		return
	}

	code := 1

	if err.Error() != "" {
		stdlog.Printf("%+v\n", err)
	}

	if exitErr, ok := err.(cli.ExitCoder); ok {
		code = exitErr.ExitCode()
	}

	os.Exit(code)
}

// CreateBuilder creates a new empty instance of command.Builder.
// The result must be passed to command.NewBuilder before it is ready to be used.
// If pipe is set to true, compression and metadata are ignored.
func CreateBuilder(pipe, compression, metadata bool) command.Builder {
	if pipe {
		return &command.PipeBuilder{}
	}

	return &command.ArgumentBuilder{
		Compressed:   compression,
		WithMetadata: metadata,
	}
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

	return cfg, nil
}

func configuration(file string) (*config.Config, error) {
	if file == "" {
		return config.CreateFromString("")
	}

	return config.LoadAndParse(file)
}

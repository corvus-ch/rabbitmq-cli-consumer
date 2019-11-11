# RabbitMQ cli consumer

[![Build Status](https://travis-ci.org/corvus-ch/rabbitmq-cli-consumer.svg?branch=master)](https://travis-ci.org/corvus-ch/rabbitmq-cli-consumer)
[![Maintainability](https://api.codeclimate.com/v1/badges/392b42c2fe09633dfd30/maintainability)](https://codeclimate.com/github/corvus-ch/rabbitmq-cli-consumer/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/392b42c2fe09633dfd30/test_coverage)](https://codeclimate.com/github/corvus-ch/rabbitmq-cli-consumer/test_coverage)

IMPORTANT: Looking for maintainer: https://github.com/corvus-ch/rabbitmq-cli-consumer/issues/81.

If you are a fellow PHP developer just like me you're probably aware of the
following fact: [PHP is meant to die][die].

When using RabbitMQ with pure PHP consumers you have to deal with stability
issues. Probably you are killing your consumers regularly. And try to solve the
problem with supervisord. Which also means on every deploy you have to restart
your consumers. A little bit dramatic if you ask me.

This is a fork of the work done by Richard van den Brand and provides a command
that aims to solve the above described problem for RabbitMQ workers by delegate
the long running part to a tool written in go which is much better suited for
this task. The PHP application then is only executed when there is an AMQP
message to process. This is comparable to how HTTP requests usually are handled
where the webs server waits for new incoming requests and calls your script once
for each request.

This fork came to be, when [Richard van den Brand][ricbra] did no longer had
the time to maintain his version. The main goals of the fork are:

- Following the principle of [The Twelve-Factor App][12factor] environment
  dependent settings should be configurable by environment variables.
- All logs, including the output of the called executable, will be available in
  STDOUT/STDERR.
- The AMQP message will be passed via STDIN (not as argument with its
  limitation in size)
- Have tests with a decent level of code coverage.

NOTE: If you previously used the consumer of [Richard van den Brand][ricbra],
this version should work as a drop in replacement. Do not migrate blindly but
do some testing before. Effort was made to remain backwards compatible, no
guarantees are made.

## Installation

You have the choice to either compile yourself or by installing via package or
binary.

## Binary

Binaries can be found at: https://github.com/corvus-ch/rabbitmq-cli-consumer/releases

### Compiling

This section assumes you're familiar with the Go language.

Use <code>go get</code> to get the source local:

```bash
$ go get github.com/corvus-ch/rabbitmq-cli-consumer
```

Change to the directory, e.g.:

```bash
$ cd $GOPATH/src/github.com/corvus-ch/rabbitmq-cli-consumer
```

Get the dependencies:

```bash
$ go get ./...
```

Then build and/or install:

```bash
$ go build
$ go install
```

## Usage

    rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable '/path/to/your/app argument --flag'

Run without arguments or with `-help` switch to show the helptext:

### Configuration

The file `example.conf` contains all available configuration options together
with its explanation.

In Go the zero value for a string is `""`. So, any values not configured in the
config file will result in a empty string. Now imagine you want to define an
empty name for one of the configuration settings. Yes, we now cannot determine
whether this value was empty on purpose or just left out. If you want to
configure an empty string you have to be explicit by using the value `<empty>`.

   rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable command.php --configuration example.conf

### Graceful shutdown

The consumer handles the signal SIGTERM. When SIGTERM is received, the AMQP
channel will be canceled, preventing any new messages from being consumed. This
allows to stop the consumer but let a currently running executable to finishing
and acknowledgement of the message.

## The executable

Your executable receives the message as the last argument. So consider the following:

   rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable command.php

The `command.php` file should look like this:

```php
#!/usr/bin/env php
<?php
// This contains first argument
$message = $argv[1];

// Decode to get original value
$original = base64_decode($message);

// Start processing
if (do_heavy_lifting($original)) {
    // All well, then return 0
    exit(0);
}

// Let rabbitmq-cli-consumer know someting went wrong, message will be requeued.
exit(1);

```

Or a Symfony2 example:

    rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable 'app/console event:processing --env=prod'

Command looks like this:

```php
<?php

namespace Vendor\EventBundle\Command;

use Symfony\Bundle\FrameworkBundle\Command\ContainerAwareCommand;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;

class TestCommand extends ContainerAwareCommand
{
    protected function configure()
    {
        $this
            ->addArgument('event', InputArgument::REQUIRED)
            ->setName('event:processing')
        ;

    }

    protected function execute(InputInterface $input, OutputInterface $output)
    {
        $message = base64_decode($input->getArgument('event'));

        $this->getContainer()->get('mailer')->send($message);

        exit(0);
    }
}
```

### Compression

Depending on what you're passing around on the queue, it may be wise to enable
compression support. If you don't you may encouter the infamous "Argument list
too long" error.

When compression is enabled, the message gets compressed with zlib maximum
compression before it's base64 encoded. We have to pay a performance penalty
for this. If you are serializing large php objects I suggest to turn it on.
Better safe then sorry.

In your config:

```ini
[rabbitmq]
compression = On

```

And in your php app:

```php
#!/usr/bin/env php
<?php
// This contains first argument
$message = $argv[1];

// Decode to get compressed value
$original = base64_decode($message);

// Uncompresss
if (! $original = gzuncompress($original)) {
    // Probably wanna throw some exception here
    exit(1);
}

// Start processing
if (do_heavy_lifting($original)) {
    // All well, then return 0
    exit(0);
}

// Let rabbitmq-cli-consumer know someting went wrong, message will be requeued.
exit(1);

```

### Including properties and message headers


If you need to access message headers and or properties, call the command with
the `--include` option set.

    rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable command.php --include

The script then will receive a json encoded data structure which looks like
the following.

```json
{
  "properties": {
    "application_headers": {
      "name": "value"
    },
    "content_type": "",
    "content_encoding": "",
    "delivery_mode": 1,
    "priority": 0,
    "correlation_id": "",
    "reply_to": "",
    "expiration": "",
    "message_id": "",
    "timestamp": "0001-01-01T00:00:00Z",
    "type": "",
    "user_id": "",
    "app_id": ""
  },
  "delivery_info": {
    "message_count": 0,
    "consumer_tag": "ctag-./rabbitmq-cli-consumer-1",
    "delivery_tag": 2,
    "redelivered": true,
    "exchange": "example",
    "routing_key": ""
  },
  "body": ""
}

```

Change your script according to the following example.

```php
#!/usr/bin/env php
<?php
// This contains first argument
$input = $argv[1];

// Decode to get original value also decrompress acording to your configuration.
$data = json_decode(base64_decode($input));

// Start processing
if (do_heavy_lifting($data->body, $data->properties)) {
    // All well, then return 0
    exit(0);
}

// Let rabbitmq-cli-consumer know someting went wrong, message will be requeued.
exit(1);
```

If you are using symfonies RabbitMQ bundle (`php-amqplib/rabbitmq-bundle`) you
can wrap the consumer with the following symfony command.

```php
<?php

namespace Vendor\EventBundle\Command;

use PhpAmqpLib\Message\AMQPMessage;
use Symfony\Bundle\FrameworkBundle\Command\ContainerAwareCommand;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;

class TestCommand extends ContainerAwareCommand
{
    protected function configure()
    {
        $this
            ->addArgument('event', InputArgument::REQUIRED)
            ->setName('event:processing')
        ;

    }

    protected function execute(InputInterface $input, OutputInterface $output)
    {
        $data = json_decode(base64_decode($input->getArgument('event')), true);
        $message = new AMQPMessage($data['body'], $data['properties']);

        /** @var \PhpAmqpLib\Message\AMQPMessage\ConsumerInterface $consumer */
        $consumer = $this->getContainer()->get('consumer');

        if (false == $consumer->execute($message)) {
            exit(1);
        }
    }
}
```

### Use pipe instead of arguments

When starting the consumer with the `--pipe` option, the AMQP message will be
passed on to the executable using STDIN for the message body and fd3 for the
metadata containing the properties and the delivery info encoded as JSON.

    rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable command.php --pipe


```php
#!/usr/bin/env php
<?php

// Read the metadata from fd3.
$metadata = file_get_contents("php://fd/3");
if (false === $metadata) {
  fwrite(STDERR, "failed to read metadata from fd3\n");
  exit(1);
}

// Decode the metadata.
$metadata = json_decode($metadata, true);
if (JSON_ERROR_NONE != json_last_error()) {
  fwrite(STDERR, "failed to decode metadata\n");
  fwrite(STDERR, json_last_error_msg() . PHP_EOL);
  exit(1);
}

// Read the body from STDIN.
$body = file_get_contents("php://stdin");
if (false === $body) {
  fwrite(STDERR, "failed to read body from STDIN\n");
  exit(1);
}

```

### Strict exit code processing

By default, any non-zero exit code will make consumer send a negative
acknowledgement and re-queue message back to the queue, in some cases it may
cause your consumer to fall into an infinite loop as re-queued message will be
getting back to consumer and it probably will fail again.

It's possible to get better control over message acknowledgement by setting up
strict exit code processing. In this mode consumer will acknowledge messages
only if executable process return an allowed exit code.

**Allowed exit codes**

| Exit Code | Action                                |
|:---------:|---------------------------------------|
| 0         | Acknowledgement                       |
| 3         | Reject                                |
| 4         | Reject and re-queue                   |
| 5         | Negative acknowledgement              |
| 6         | Negative acknowledgement and re-queue |

All other exit codes will cause consumer to fail.

Run consumer with `--strict-exit-code` option to enable strict exit code processing:

    rabbitmq-cli-consumer --verbose --url amqp://guest:guest@localhost --queue myqueue --executable command.php --strict-exit-code

Make sure your executable returns correct exit code

```php
#!/usr/bin/env php
<?php
// ...
try {
    if (do_heavy_lifting($data)) {
        // All well, then return 0
        exit(0);
    }
} catch(InvalidMessageBody $e) {
    exit(3); // Message is invalid, just reject and don't try to process again
} catch(TimeoutException $e) {
    exit(4); // Reject and try again
} catch(Exception $e) {
    exit(1); // Unexpected exception will cause consumer to stop consuming
}

```

## Metrics

Metrics are following the [Prometheus](https://prometheus.io/docs/introduction/overview/) conventions.
They are available via HTTP endpoint (http://127.0.0.1:9566/metrics), with default port `9566` and default path `/metrics`.

The following metrics are tracked:

| Metric                                           | Type      | Description |
| ------------------------------------------------ | --------- | ----------- |
| `rabbitmq_cli_consumer_process_total`            | Counter   | The total number of processes executed. Processes are aggregated by their exit code.  |
| `rabbitmq_cli_consumer_process_duration_seconds` | Histogram | The time spent by the consumer to process the message. |
| `rabbitmq_cli_consumer_message_duration_seconds` | Histogram | The time spent from publishing to finished processing the message. This requires the message to have the `timestamp` header set. |

## Contributing and license

This library is licenced under [MIT](LICENSE). For information about how to
contribute to this project, see [CONTRIBUTING.md].

[12factor]: https://12factor.net
[CONTRIBUTING.md]: https://github.com/corvus-ch/rabbitmq-cli-consumer/blob/master/CONTRIBUTING.md
[die]: https://software-gunslinger.tumblr.com/post/47131406821/php-is-meant-to-die
[ricbra]: https://github.com/ricbra

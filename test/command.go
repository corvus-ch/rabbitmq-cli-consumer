// +build integration

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Command to be executed by the consumer during integration tests.
//
// The input is written to an output file allowing to do some assertions in the tests.
func main() {
	outputFile := flag.String("output", "./command.log", "the output file")
	isCompressed := flag.Bool("comp", false, "whether the argument is compressed or not")
	isPipe := flag.Bool("pipe", false, "whether the argument passed via stdin (TRUE) or argument (FALSE)")
	flag.Parse()
	var f io.Writer
	f = os.Stdout
	if *outputFile != "-" {
		of, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file: %v\n", err)
			os.Exit(1)
		}
		defer of.Close()
		f = of
	}
	f.Write([]byte("Got executed\n"))

	var message []byte
	if *isPipe {
		var err error

		pipe := os.NewFile(3, "/proc/self/fd/3")
		defer pipe.Close()
		metadata, err := ioutil.ReadAll(pipe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read metadata from pipe: %v", err)
			os.Exit(1)
		}

		f.Write(metadata)
		f.Write([]byte("\n"))

		message, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read body from pipe: %v", err)
			os.Exit(1)
		}
	} else {
		message = []byte(os.Args[len(os.Args)-1])
	}

	f.Write(message)
	f.Write([]byte("\n"))


	if !*isPipe {
		var r io.Reader
		r = base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(message))
	if *isCompressed {
				zr, err := zlib.NewReader(r)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to create zlib reader: %v\n", err)
					os.Exit(1)
				}
				defer zr.Close()
				r = zr
	}
		original, err := ioutil.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to decompress input: %v\n", err)
			os.Exit(1)
		}
		f.Write(original)
		f.Write([]byte("\n"))
	}
}

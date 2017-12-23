// +build integration

package main

import (
	"fmt"
	"os"
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"encoding/base64"
	"flag"
	"io"
)

// Command to be executed by the consumer during integration tests.
//
// The input is written to an output file allowing to do some assertions in the tests.
func main() {
	outputFile := flag.String("output", "./command.log", "the output file")
	isCompressed := flag.Bool("comp", false, "whether the argument is compressed or not")
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

	message := []byte(os.Args[len(os.Args)-1])

	f.Write(message)
	f.Write([]byte("\n"))


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

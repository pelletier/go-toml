// Tomljson reads TOML and converts to JSON.
//
// Usage:
//   cat file.toml | tomljson > file.json
//   tomljson file1.toml > file.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `tomljson can be used in two ways:
Reading from stdin:
  cat file.toml | tomljson > file.json

Reading from a file:
  tomljson file.toml > file.json
`)
	}
	flag.Parse()
	os.Exit(processMain(flag.Args(), os.Stdin, os.Stdout, os.Stderr))
}

func processMain(files []string, defaultInput io.Reader, output io.Writer, errorOutput io.Writer) int {
	// read from stdin and print to stdout
	inputReader := defaultInput

	if len(files) > 0 {
		var err error
		inputReader, err = os.Open(files[0])
		if err != nil {
			printError(err, errorOutput)
			return -1
		}
	}
	s, err := reader(inputReader)
	if err != nil {
		printError(err, errorOutput)
		return -1
	}
	io.WriteString(output, s+"\n")
	return 0
}

func printError(err error, output io.Writer) {
	io.WriteString(output, err.Error()+"\n")
}

func reader(r io.Reader) (string, error) {
	var v interface{}

	d := toml.NewDecoder(r)
	err := d.Decode(&v)
	if err != nil {
		return "", err
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

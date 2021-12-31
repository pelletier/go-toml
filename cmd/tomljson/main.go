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

func processMain(files []string, input io.Reader, output, error io.Writer) int {
	err := run(files, input, output)
	if err != nil {
		fmt.Fprintln(error, err.Error())
		return -1
	}
	return 0
}

func run(files []string, input io.Reader, output io.Writer) error {
	if len(files) > 0 {
		f, err := os.Open(files[0])
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	}

	return convert(input, output)
}

func convert(r io.Reader, w io.Writer) error {
	var v interface{}

	d := toml.NewDecoder(r)
	err := d.Decode(&v)
	if err != nil {
		return err
	}

	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(v)
}

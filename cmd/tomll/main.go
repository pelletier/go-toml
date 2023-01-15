// Package tomll is a linter program for TOML.
//
// # Usage
//
// Reading from stdin, writing to stdout:
//
//	cat file.toml | tomll
//
// Reading and updating a list of files in place:
//
//	tomll a.toml b.toml c.toml
//
// # Installation
//
// Using Go:
//
//	go install github.com/pelletier/go-toml/v2/cmd/tomll@latest
package main

import (
	"flag"
	"io"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/cli"
)

const usage = `tomll can be used in two ways:

Reading from stdin, writing to stdout:
  cat file.toml | tomll > file.toml

Reading and updating a list of files in place:
  tomll a.toml b.toml c.toml

When given a list of files, tomll will modify all files in place without asking.

Flags:
-multiLineArray      sets up the linter to encode arrays with more than one element on multiple lines instead of one.
`

func main() {
	multiLineArray := flag.Bool("multiLineArray", false, "sets up the linter to encode arrays with more than one element on multiple lines insteadof one.")
	p := cli.Program{
		Usage:   usage,
		Fn:      convert,
		Inplace: true,
		Opts: cli.Options{"multiLineArray": multiLineArray},
	}
	p.Execute()
}

func convert(r io.Reader, w io.Writer, o cli.Options) error {
	var v interface{}

	multiLineArray := o["multiLineArray"].(bool)

	d := toml.NewDecoder(r)
	err := d.Decode(&v)
	if err != nil {
		return err
	}

	e := toml.NewEncoder(w)
	return e.SetArraysMultiline(multiLineArray).Encode(v)
}

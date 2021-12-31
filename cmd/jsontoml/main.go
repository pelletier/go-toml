// Jsontoml reads JSON and converts to TOML.
//
// Usage:
//   cat file.toml | jsontoml > file.json
//   jsontoml file1.toml > file.json
package main

import (
	"encoding/json"
	"io"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/cli"
)

func main() {
	usage := `jsontoml can be used in two ways:
Reading from stdin:
  cat file.json | jsontoml > file.toml

Reading from a file:
  jsontoml file.json > file.toml
`
	cli.Execute(usage, convert)
}

func convert(r io.Reader, w io.Writer) error {
	var v interface{}

	d := json.NewDecoder(r)
	err := d.Decode(&v)
	if err != nil {
		return err
	}

	e := toml.NewEncoder(w)
	return e.Encode(v)
}

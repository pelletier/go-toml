// Package cli provides common functions for command-line programs.
package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type ConvertFn func(r io.Reader, w io.Writer) error

type Program struct {
	Usage string
	Fn    ConvertFn
	// Inplace allows the command to take more than one file as argument and
	// perform conversion in place on each provided file.
	Inplace bool
}

func (p *Program) Execute() {
	flag.Usage = func() { fmt.Fprint(os.Stderr, p.Usage) }
	flag.Parse()
	os.Exit(p.main(flag.Args(), os.Stdin, os.Stdout, os.Stderr))
}

func (p *Program) main(files []string, input io.Reader, output, stderr io.Writer) int {
	err := p.run(files, input, output)
	if err != nil {
		var derr *toml.DecodeError
		if errors.As(err, &derr) {
			_, _ = fmt.Fprintln(stderr, derr.String())
			row, col := derr.Position()
			_, _ = fmt.Fprintln(stderr, "error occurred at row", row, "column", col)
		} else {
			_, _ = fmt.Fprintln(stderr, err.Error())
		}

		return -1
	}
	return 0
}

func (p *Program) run(files []string, input io.Reader, output io.Writer) error {
	if len(files) > 0 {
		if p.Inplace {
			return p.runAllFilesInPlace(files)
		}
		f, err := os.Open(files[0])
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		input = f
	}
	return p.Fn(input, output)
}

func (p *Program) runAllFilesInPlace(files []string) error {
	for _, path := range files {
		err := p.runFileInPlace(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Program) runFileInPlace(path string) error {
	in, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return err
	}

	out := new(bytes.Buffer)

	err = p.Fn(bytes.NewReader(in), out)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out.Bytes(), 0o600)
}

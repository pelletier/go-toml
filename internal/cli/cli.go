package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type ConvertFn func(r io.Reader, w io.Writer) error

func Execute(usage string, fn ConvertFn) {
	flag.Usage = func() { fmt.Fprintf(os.Stderr, usage) }
	flag.Parse()
	os.Exit(processMain(flag.Args(), os.Stdin, os.Stdout, os.Stderr, fn))
}

func processMain(files []string, input io.Reader, output, error io.Writer, f ConvertFn) int {
	err := run(files, input, output, f)
	if err != nil {
		fmt.Fprintln(error, err.Error())
		return -1
	}
	return 0
}

func run(files []string, input io.Reader, output io.Writer, convert ConvertFn) error {
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

// Tomll is a linter for TOML
//
// Usage:
//   cat file.toml | tomll > file_linted.toml
//   tomll file1.toml file2.toml # lint the two files in place
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
)

func main() {
	multiLineArray := flag.Bool("multiLineArray", false, "sets up the linter to encode arrays with more than one element on multiple lines instead of one.")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "tomll can be used in two ways:")
		fmt.Fprintln(os.Stderr, "Writing to STDIN and reading from STDOUT:")
		fmt.Fprintln(os.Stderr, "  cat file.toml | tomll > file.toml")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Reading and updating a list of files:")
		fmt.Fprintln(os.Stderr, "  tomll a.toml b.toml c.toml")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "When given a list of files, tomll will modify all files in place without asking.")
		fmt.Fprintln(os.Stderr, "When given a list of files, tomll will modify all files in place without asking.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "-multiLineArray      sets up the linter to encode arrays with more than one element on multiple lines instead of one.")
	}
	flag.Parse()

	// read from stdin and print to stdout
	if flag.NArg() == 0 {
		s, err := lintReader(os.Stdin, *multiLineArray)
		if err != nil {
			io.WriteString(os.Stderr, err.Error())
			os.Exit(-1)
		}
		io.WriteString(os.Stdout, s)
	} else {
		// otherwise modify a list of files
		for _, filename := range flag.Args() {
			s, err := lintFile(filename, *multiLineArray)
			if err != nil {
				io.WriteString(os.Stderr, err.Error())
				os.Exit(-1)
			}
			ioutil.WriteFile(filename, []byte(s), 0644)
		}
	}
}

func lintFile(filename string, multiLineArray bool) (string, error) {
	tree, err := toml.LoadFile(filename)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).ArraysWithOneElementPerLine(multiLineArray).Encode(tree); err != nil {
		panic(err)
	}

	return buf.String(), nil
}

func lintReader(r io.Reader, multiLineArray bool) (string, error) {
	tree, err := toml.LoadReader(r)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).ArraysWithOneElementPerLine(multiLineArray).Encode(tree); err != nil {
		panic(err)
	}
	return buf.String(), nil
}

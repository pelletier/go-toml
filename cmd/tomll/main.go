package main

import (
	"flag"
	"io"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
)

func main() {
	flag.Parse()
	// read from stdin and print to stdout
	if flag.NArg() == 0 {
		s, err := lintReader(os.Stdin)
		if err != nil {
			io.WriteString(os.Stderr, err.Error())
			os.Exit(-1)
		}
		io.WriteString(os.Stdout, s)
	} else {
		// otherwise modify a list of files
		for _, filename := range flag.Args() {
			s, err := lintFile(filename)
			if err != nil {
				io.WriteString(os.Stderr, err.Error())
				os.Exit(-1)
			}
			ioutil.WriteFile(filename, []byte(s), 0644)
		}
	}
}

func lintFile(filename string) (string, error) {
	tree, err := toml.LoadFile(filename)
	if err != nil {
		return "", err
	}
	return tree.String(), nil
}

func lintReader(r io.Reader) (string, error) {
	tree, err := toml.LoadReader(r)
	if err != nil {
		return "", err
	}
	return tree.String(), nil
}

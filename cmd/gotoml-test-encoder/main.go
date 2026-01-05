// Package gotoml-test-encoder is a minimal encoder program used to compare this library with other TOML implementations.
package main

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/pelletier/go-toml/v2/internal/testsuite"
)

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
	}

	err := testsuite.EncodeStdin()
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	log.Printf("Usage: %s < json-file\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}

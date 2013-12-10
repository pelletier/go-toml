package main

import (
	"io/ioutil"
	"os"
	"github.com/pelletier/go-toml"
)

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	_, err = toml.Load(string(bytes))
	if err == nil {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

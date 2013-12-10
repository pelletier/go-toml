#!/bin/bash

go get github.com/BurntSushi/toml-test # install test suite
go get github.com/BurntSushi/toml/toml-test-go # install my parser
go build -o test_program_bin github.com/pelletier/go-toml/test_program
$GOPATH/bin/toml-test ./test_program_bin # run tests on my parser

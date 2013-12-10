#!/bin/bash

go get github.com/BurntSushi/toml-test # install test suite
go get github.com/BurntSushi/toml/toml-test-go # install my parser
go build -o test_program_bin github.com/pelletier/go-toml/test_program

toml_test_wrapper() {
    if hash toml-test 2>/dev/null; then # test availability in $PATH
        toml-test "$@"
    else
        p="$HOME/gopath/bin/toml-test" # try in Travi's place
        if [ -f "$p" ]; then
            "$p" "$@"
        else
            "$GOPATH/bin/toml-test" "$@"
        fi
    fi
}

toml_test_wrapper ./test_program_bin # run tests on my parser

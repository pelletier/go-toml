#!/bin/bash

go get github.com/BurntSushi/toml-test # install test suite
go get github.com/BurntSushi/toml/toml-test-go # install my parser
go build -o test_program_bin github.com/pelletier/go-toml/test_program

toml_test_wrapper() {
    ret=0
    if hash toml-test 2>/dev/null; then # test availability in $PATH
        toml-test "$@"
        ret=$?
    else
        p="$HOME/gopath/bin/toml-test" # try in Travi's place
        if [ -f "$p" ]; then
            "$p" "$@"
            ret=$?
        else
            "$GOPATH/bin/toml-test" "$@"
            ret=$?
        fi
    fi
}

toml_test_wrapper ./test_program_bin | tee test_out
ret="$([ `tail -n 1 test_out | sed -E 's/^.+([0-9]+) failed$/\1/'` -eq 0 ])"
exit $ret

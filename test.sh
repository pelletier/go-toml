#!/bin/bash

# Run basic go unit tests
go test -v ./...
result=$?

# Run example-based toml tests
cd test_program && ./go-test.sh
result="$(( result || $? ))"

exit $result

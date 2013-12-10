#!/bin/bash

# Run basic go unit tests
go test -v ./...

# Run example-based toml tests
cd test_program && ./go-test.sh

#!/bin/bash
# fail out of the script if anything here fails
set -e

# set the path to the present working directory
export GOPATH=`pwd`

# Vendorize the BurntSushi test suite
# NOTE: this gets a specific release to avoid versioning issues
if [ ! -d 'src/github.com/BurntSushi/toml-test' ]; then
  mkdir -p src/github.com/BurntSushi
  git clone https://github.com/BurntSushi/toml-test.git src/github.com/BurntSushi/toml-test
fi
pushd src/github.com/BurntSushi/toml-test
git reset --hard '0.2.0'  # use the released version, NOT tip
popd
go build -o toml-test github.com/BurntSushi/toml-test

# vendorize the current lib for testing
# NOTE: this basically mocks an install without having to go back out to github for code
mkdir -p src/github.com/pelletier/go-toml/cmd
cp *.go *.toml src/github.com/pelletier/go-toml
cp cmd/*.go src/github.com/pelletier/go-toml/cmd
go build -o test_program_bin src/github.com/pelletier/go-toml/cmd/test_program.go

# Run basic unit tests and then the BurntSushi test suite
go test -v github.com/pelletier/go-toml
./toml-test ./test_program_bin | tee test_out

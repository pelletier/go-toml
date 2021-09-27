// Package testsuite runs tests from the github.com/BurntSushi/toml-test
// test suite.
//
// The data files are included within the toml-test package, so no file
// generation is required.
package testsuite

import (
	"encoding/json"
	"log"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Decode is a helper function for the toml-test binary interface.  TOML input
// is read from STDIN and a resulting tagged JSON representation is written to
// STDOUT.
func Decode() {
	var decoded map[string]interface{}

	if err := toml.NewDecoder(os.Stdin).Decode(&decoded); err != nil {
		log.Fatalf("Error decoding TOML: %s", err)
	}

	j := json.NewEncoder(os.Stdout)
	j.SetIndent("", "  ")
	if err := j.Encode(addTag("", decoded)); err != nil {
		log.Fatalf("Error encoding JSON: %s", err)
	}
}

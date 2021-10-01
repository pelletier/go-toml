//go:generate go run ./cmd/tomltestgen/main.go -o toml_testgen_test.go

// This is a support file for toml_testgen_test.go
package toml_test

import (
	"encoding/json"
	"testing"

	"github.com/pelletier/go-toml/v2/testsuite"
	"github.com/stretchr/testify/require"
)

func testgenInvalid(t *testing.T, input string) {
	t.Helper()
	t.Logf("Input TOML:\n%s", input)

	doc := map[string]interface{}{}
	err := testsuite.Unmarshal([]byte(input), &doc)

	if err == nil {
		t.Log(json.Marshal(doc))
		t.Fatalf("test did not fail")
	}
}

func testgenValid(t *testing.T, input string, jsonRef string) {
	t.Helper()
	t.Logf("Input TOML:\n%s", input)

	// TODO: change this to interface{}
	var doc map[string]interface{}

	err := testsuite.Unmarshal([]byte(input), &doc)
	if err != nil {
		t.Fatalf("failed parsing toml: %s", err)
	}
	j, err := testsuite.ValueToTaggedJSON(doc)
	require.NoError(t, err)
	require.Equal(t, jsonRef, string(j)+"\n")
}

//go:generate go run github.com/toml-lang/toml-test/cmd/toml-test@v1.6.0 -copy ./tests
//go:generate go run ./cmd/tomltestgen/main.go -r v1.6.0 -o toml_testgen_test.go

package toml_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
	"github.com/pelletier/go-toml/v2/internal/testsuite"
)

func testgenInvalid(t *testing.T, input string) {
	t.Helper()
	t.Logf("Input TOML:\n%s", input)

	doc := map[string]interface{}{}
	err := testsuite.Unmarshal([]byte(input), &doc)

	if err == nil {
		out, err := json.Marshal(doc)
		if err != nil {
			panic("could not marshal map to json")
		}
		t.Log("JSON output from unmarshal:", string(out))
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
		de := &toml.DecodeError{}
		if errors.As(err, &de) {
			t.Logf("%s\n%s", err, de)
		}
		t.Fatalf("failed parsing toml: %s", err)
	}
	j, err := testsuite.ValueToTaggedJSON(doc)
	assert.NoError(t, err)

	var ref interface{}
	err = json.Unmarshal([]byte(jsonRef), &ref)
	assert.NoError(t, err)

	var actual interface{}
	err = json.Unmarshal(j, &actual)
	assert.NoError(t, err)

	testsuite.CmpJSON(t, "", ref, actual)
}

// This is a support file for toml_testgen_test.go
package toml_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func testgenInvalid(t *testing.T, input string) {
	t.Helper()
	t.Logf("Input TOML:\n%s", input)

	doc := map[string]interface{}{}
	err := toml.Unmarshal([]byte(input), &doc)

	if err == nil {
		t.Log(json.Marshal(doc))
		t.Fatalf("test did not fail")
	}
}

func testgenValid(t *testing.T, input string, jsonRef string) {
	t.Helper()
	t.Logf("Input TOML:\n%s", input)

	doc := map[string]interface{}{}

	err := toml.Unmarshal([]byte(input), &doc)
	if err != nil {
		t.Fatalf("failed parsing toml: %s", err)
	}

	refDoc := testgenBuildRefDoc(jsonRef)

	require.Equal(t, refDoc, doc)

	out, err := toml.Marshal(doc)
	require.NoError(t, err)

	doc2 := map[string]interface{}{}
	err = toml.Unmarshal(out, &doc2)
	require.NoError(t, err)

	require.Equal(t, refDoc, doc2)
}

func testgenBuildRefDoc(jsonRef string) map[string]interface{} {
	descTree := map[string]interface{}{}

	err := json.Unmarshal([]byte(jsonRef), &descTree)
	if err != nil {
		panic(fmt.Sprintf("reference doc should be valid JSON: %s", err))
	}

	doc := testGenTranslateDesc(descTree)
	if doc == nil {
		return map[string]interface{}{}
	}

	return doc.(map[string]interface{})
}

//nolint:funlen,gocognit,cyclop
func testGenTranslateDesc(input interface{}) interface{} {
	a, ok := input.([]interface{})
	if ok {
		xs := make([]interface{}, len(a))
		for i, v := range a {
			xs[i] = testGenTranslateDesc(v)
		}

		return xs
	}

	d, ok := input.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("input should be valid map[string]: %v", input))
	}

	var (
		dtype  string
		dvalue interface{}
	)

	//nolint:nestif
	if len(d) == 2 {
		dtypeiface, ok := d["type"]
		if ok {
			dvalue, ok = d["value"]
			if ok {
				dtype = dtypeiface.(string)

				switch dtype {
				case "string":
					return dvalue.(string)
				case "float":
					v, err := strconv.ParseFloat(dvalue.(string), 64)
					if err != nil {
						panic(fmt.Sprintf("invalid float '%s': %s", dvalue, err))
					}

					return v
				case "integer":
					v, err := strconv.ParseInt(dvalue.(string), 10, 64)
					if err != nil {
						panic(fmt.Sprintf("invalid int '%s': %s", dvalue, err))
					}

					return v
				case "bool":
					return dvalue.(string) == "true"
				case "datetime":
					dt, err := time.Parse("2006-01-02T15:04:05Z", dvalue.(string))
					if err != nil {
						panic(fmt.Sprintf("invalid datetime '%s': %s", dvalue, err))
					}

					return dt
				case "array":
					if dvalue == nil {
						return nil
					}

					a := dvalue.([]interface{})

					xs := make([]interface{}, len(a))

					for i, v := range a {
						xs[i] = testGenTranslateDesc(v)
					}

					return xs
				}

				panic(fmt.Sprintf("unknown type: %s", dtype))
			}
		}
	}

	dest := map[string]interface{}{}
	for k, v := range d {
		dest[k] = testGenTranslateDesc(v)
	}

	return dest
}

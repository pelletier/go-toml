package main

import (
	"io/ioutil"
	"os"
	"github.com/pelletier/go-toml"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(2)
	}
	tree, err := toml.Load(string(bytes))
	if err != nil {
		os.Exit(1)
	}

	typedTree := translate((map[string]interface{})(*tree))

	if err := json.NewEncoder(os.Stdout).Encode(typedTree); err != nil {
		log.Fatalf("Error encoding JSON: %s", err)
	}

	os.Exit(0)
}

func translate(tomlData interface{}) interface{} {

	switch orig := tomlData.(type) {
	case map[string]interface{}:
		typed := make(map[string]interface{}, len(orig))
		for k, v := range orig {
			typed[k] = translate(v)
		}
		return typed
	case *toml.TomlTree:
		return translate((map[string]interface{})(*orig))
	case []map[string]interface{}:
		typed := make([]map[string]interface{}, len(orig))
		for i, v := range orig {
			typed[i] = translate(v).(map[string]interface{})
		}
		return typed
	case []interface{}:
		typed := make([]interface{}, len(orig))
		for i, v := range orig {
			typed[i] = translate(v)
		}
		return tag("array", typed)
	case time.Time:
		return tag("datetime", orig.Format("2006-01-02T15:04:05Z"))
	case bool:
		return tag("bool", fmt.Sprintf("%v", orig))
	case int64:
		return tag("integer", fmt.Sprintf("%d", orig))
	case float64:
		return tag("float", fmt.Sprintf("%v", orig))
	case string:
		return tag("string", orig)
	}

	panic(fmt.Sprintf("Unknown type: %T", tomlData))
}

func tag(typeName string, data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":  typeName,
		"value": data,
	}
}

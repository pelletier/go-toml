package toml

import (
	"testing"
	"time"
)

func validate(t *testing.T, path string, object interface{}) {
	switch o := object.(type) {
	case *TomlTree:
		for key, tree := range o.values {
			validate(t, path+"."+key, tree)
		}
	case []*TomlTree:
		for index, tree := range o {
			validate(t, path+"."+string(index), tree)
		}
	case *tomlValue:
		switch o.value.(type) {
		case int64, uint64, bool, string, float64, time.Time,
			[]int64, []uint64, []bool, []string, []float64, []time.Time:
			return // ok
		default:
			t.Fatalf("tomlValue at key %s containing incorrect type %T", path, o.value)
		}
	default:
		t.Fatalf("value at key %s is of incorrect type %T", path, object)
	}
	t.Log("validation ok", path)
}

func validateTree(t *testing.T, tree *TomlTree) {
	validate(t, "", tree)
}

func TestTomlTreeCreateToTree(t *testing.T) {
	data := map[string]interface{}{
		"a_string": "bar",
		"an_int":   42,
		"int8":     int8(2),
		"int16":    int16(2),
		"int32":    int32(2),
		"uint8":    uint8(2),
		"uint16":   uint16(2),
		"uint32":   uint32(2),
		"float32":  float32(2),
		"a_bool":   false,
		"nested": map[string]interface{}{
			"foo": "bar",
		},
		"array":       []string{"a", "b", "c"},
		"array_uint":  []uint{uint(1), uint(2)},
		"array_table": []map[string]interface{}{map[string]interface{}{"sub_map": 52}},
	}
	tree, err := TreeFromMap(data)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	validateTree(t, tree)
}

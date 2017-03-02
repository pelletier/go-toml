package toml

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func assertErrorString(t *testing.T, expected string, err error) {
	expectedErr := errors.New(expected)
	if err.Error() != expectedErr.Error() {
		t.Errorf("expecting error %s, but got %s instead", expected, err)
	}
}

func TestTomlTreeWriteToTomlString(t *testing.T) {
	toml, err := Load(`name = { first = "Tom", last = "Preston-Werner" }
points = { x = 1, y = 2 }`)

	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	tomlString, _ := toml.ToTomlString()
	reparsedTree, err := Load(tomlString)

	assertTree(t, reparsedTree, err, map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Tom",
			"last":  "Preston-Werner",
		},
		"points": map[string]interface{}{
			"x": int64(1),
			"y": int64(2),
		},
	})
}

func TestTomlTreeWriteToTomlStringSimple(t *testing.T) {
	tree, err := Load("[foo]\n\n[[foo.bar]]\na = 42\n\n[[foo.bar]]\na = 69\n")
	if err != nil {
		t.Errorf("Test failed to parse: %v", err)
		return
	}
	result, err := tree.ToTomlString()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expected := "\n[foo]\n\n  [[foo.bar]]\n    a = 42\n\n  [[foo.bar]]\n    a = 69\n"
	if result != expected {
		t.Errorf("Expected got '%s', expected '%s'", result, expected)
	}
}

func TestTomlTreeWriteToTomlStringKeysOrders(t *testing.T) {
	for i := 0; i < 100; i++ {
		tree, _ := Load(`
		foobar = true
		bar = "baz"
		foo = 1
		[qux]
		  foo = 1
		  bar = "baz2"`)

		stringRepr, _ := tree.ToTomlString()

		t.Log("Intermediate string representation:")
		t.Log(stringRepr)

		r := strings.NewReader(stringRepr)
		toml, err := LoadReader(r)

		if err != nil {
			t.Fatal("Unexpected error:", err)
		}

		assertTree(t, toml, err, map[string]interface{}{
			"foobar": true,
			"bar":    "baz",
			"foo":    1,
			"qux": map[string]interface{}{
				"foo": 1,
				"bar": "baz2",
			},
		})
	}
}

func testMaps(t *testing.T, actual, expected map[string]interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("trees aren't equal.\n", "Expected:\n", expected, "\nActual:\n", actual)
	}
}

func TestToTomlStringTypeConversionError(t *testing.T) {
	tree := TomlTree{
		values: map[string]interface{}{
			"thing": &tomlValue{[]string{"unsupported"}, Position{}},
		},
	}
	_, err := tree.ToTomlString()
	expected := errors.New("unsupported value type []string: [unsupported]")
	if err.Error() != expected.Error() {
		t.Errorf("expecting error %s, but got %s instead", expected, err)
	}
}

func TestTomlTreeWriteToMapSimple(t *testing.T) {
	tree, _ := Load("a = 42\nb = 17")

	expected := map[string]interface{}{
		"a": int64(42),
		"b": int64(17),
	}

	testMaps(t, tree.ToMap(), expected)
}

func TestTomlTreeWriteToInvalidTreeSimpleValue(t *testing.T) {
	tree := TomlTree{values: map[string]interface{}{"foo": int8(1)}}
	_, err := tree.ToTomlString()
	assertErrorString(t, "invalid key type at foo: int8", err)
}

func TestTomlTreeWriteToInvalidTreeTomlValue(t *testing.T) {
	tree := TomlTree{values: map[string]interface{}{"foo": &tomlValue{int8(1), Position{}}}}
	_, err := tree.ToTomlString()
	assertErrorString(t, "unsupported value type int8: 1", err)
}

func TestTomlTreeWriteToInvalidTreeTomlValueArray(t *testing.T) {
	tree := TomlTree{values: map[string]interface{}{"foo": &tomlValue{[]interface{}{int8(1)}, Position{}}}}
	_, err := tree.ToTomlString()
	assertErrorString(t, "unsupported value type int8: 1", err)
}

func TestTomlTreeWriteToMapExampleFile(t *testing.T) {
	tree, _ := LoadFile("example.toml")
	expected := map[string]interface{}{
		"title": "TOML Example",
		"owner": map[string]interface{}{
			"name":         "Tom Preston-Werner",
			"organization": "GitHub",
			"bio":          "GitHub Cofounder & CEO\nLikes tater tots and beer.",
			"dob":          time.Date(1979, time.May, 27, 7, 32, 0, 0, time.UTC),
		},
		"database": map[string]interface{}{
			"server":         "192.168.1.1",
			"ports":          []interface{}{int64(8001), int64(8001), int64(8002)},
			"connection_max": int64(5000),
			"enabled":        true,
		},
		"servers": map[string]interface{}{
			"alpha": map[string]interface{}{
				"ip": "10.0.0.1",
				"dc": "eqdc10",
			},
			"beta": map[string]interface{}{
				"ip": "10.0.0.2",
				"dc": "eqdc10",
			},
		},
		"clients": map[string]interface{}{
			"data": []interface{}{
				[]interface{}{"gamma", "delta"},
				[]interface{}{int64(1), int64(2)},
			},
		},
	}
	testMaps(t, tree.ToMap(), expected)
}

func TestTomlTreeWriteToMapWithTablesInMultipleChunks(t *testing.T) {
	tree, _ := Load(`
	[[menu.main]]
        a = "menu 1"
        b = "menu 2"
        [[menu.main]]
        c = "menu 3"
        d = "menu 4"`)
	expected := map[string]interface{}{
		"menu": map[string]interface{}{
			"main": []interface{}{
				map[string]interface{}{"a": "menu 1", "b": "menu 2"},
				map[string]interface{}{"c": "menu 3", "d": "menu 4"},
			},
		},
	}
	treeMap := tree.ToMap()

	testMaps(t, treeMap, expected)
}

func TestTomlTreeWriteToMapWithArrayOfInlineTables(t *testing.T) {
	tree, _ := Load(`
    	[params]
	language_tabs = [
    		{ key = "shell", name = "Shell" },
    		{ key = "ruby", name = "Ruby" },
    		{ key = "python", name = "Python" }
	]`)

	expected := map[string]interface{}{
		"params": map[string]interface{}{
			"language_tabs": []interface{}{
				map[string]interface{}{
					"key":  "shell",
					"name": "Shell",
				},
				map[string]interface{}{
					"key":  "ruby",
					"name": "Ruby",
				},
				map[string]interface{}{
					"key":  "python",
					"name": "Python",
				},
			},
		},
	}

	treeMap := tree.ToMap()
	testMaps(t, treeMap, expected)
}

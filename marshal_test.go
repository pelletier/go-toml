package toml

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

type basicMarshalTestStruct struct {
	String     string                      `toml:"string"`
	StringList []string                    `toml:"strlist"`
	Sub        basicMarshalTestSubStruct   `toml:"subdoc"`
	SubList    []basicMarshalTestSubStruct `toml:"sublist"`
}

type basicMarshalTestSubStruct struct {
	String2 string
}

var basicTestData = basicMarshalTestStruct{
	String:     "Hello",
	StringList: []string{"Howdy", "Hey There"},
	Sub:        basicMarshalTestSubStruct{"One"},
	SubList:    []basicMarshalTestSubStruct{{"Two"}, {"Three"}},
}

var basicTestToml = []byte(`string = "Hello"
strlist = ["Howdy","Hey There"]

[subdoc]
  string2 = "One"

[[sublist]]
  string2 = "Two"

[[sublist]]
  string2 = "Three"
`)

func TestBasicMarshal(t *testing.T) {
	result, err := Marshal(basicTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := basicTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestBasicUnmarshal(t *testing.T) {
	result := basicMarshalTestStruct{}
	err := Unmarshal(basicTestToml, &result)
	expected := basicTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad unmarshal: expected %v, got %v", expected, result)
	}
}

type testDoc struct {
	Title      string            `toml:"title"`
	Basics     testDocBasics     `toml:"basic"`
	BasicLists testDocBasicLists `toml:"basic_lists"`
	BasicMap   map[string]string `toml:"basic_map"`
	Subdocs    testDocSubs       `toml:"subdoc"`
	SubDocList []testSubDoc      `toml:"subdoclist"`
}

type testDocBasics struct {
	Bool   bool      `toml:"bool"`
	Date   time.Time `toml:"date"`
	Float  float32   `toml:"float"`
	Int    int       `toml:"int"`
	String string    `toml:"string"`
}

type testDocBasicLists struct {
	Bools   []bool      `toml:"bools"`
	Dates   []time.Time `toml:"dates"`
	Floats  []float32   `toml:"floats"`
	Ints    []int       `toml:"ints"`
	Strings []string    `toml:"strings"`
}

type testDocSubs struct {
	First  testSubDoc `toml:"first"`
	Second testSubDoc `toml:"second"`
}

type testSubDoc struct {
	Name string `toml:"name"`
}

var docData = testDoc{
	Title: "TOML Marshal Testing",
	Basics: testDocBasics{
		Bool:   true,
		Date:   time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
		Float:  123.4,
		Int:    5000,
		String: "Bite me",
	},
	BasicLists: testDocBasicLists{
		Bools: []bool{true, false, true},
		Dates: []time.Time{
			time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
			time.Date(1980, 5, 27, 7, 32, 0, 0, time.UTC),
		},
		Floats:  []float32{12.3, 45.6, 78.9},
		Ints:    []int{8001, 8001, 8002},
		Strings: []string{"One", "Two", "Three"},
	},
	BasicMap: map[string]string{
		"one": "one",
		"two": "two",
	},
	Subdocs: testDocSubs{
		First:  testSubDoc{"First"},
		Second: testSubDoc{"Second"},
	},
	SubDocList: []testSubDoc{
		testSubDoc{"List.First"},
		testSubDoc{"List.Second"},
	},
}

func TestDocMarshal(t *testing.T) {
	result, err := Marshal(docData)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := ioutil.ReadFile("marshal_test.toml")
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestDocUnmarshal(t *testing.T) {
	result := testDoc{}
	tomlData, _ := ioutil.ReadFile("marshal_test.toml")
	err := Unmarshal(tomlData, &result)
	expected := docData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		resStr, _ := json.MarshalIndent(result, "", "  ")
		expStr, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("Bad unmarshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expStr, resStr)
	}
}

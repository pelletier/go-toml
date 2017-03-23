package toml

import (
	"bytes"
	"reflect"
	"testing"
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

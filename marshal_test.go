package toml

import (
	"bytes"
	"reflect"
	"testing"
)

type basicMarshalTestStruct struct {
	String string                    `toml:"string"`
	Sub    basicMarshalTestSubStruct `toml:"subdoc"`
}

type basicMarshalTestSubStruct struct {
	String2 string
}

func TestBasicMarshal(t *testing.T) {
	x := basicMarshalTestStruct{
		String: "Hello",
		Sub: basicMarshalTestSubStruct{
			String2: "Howdy",
		},
	}
	expected := []byte(`string = "Hello"

[subdoc]
  string2 = "Howdy"
`)
	result, err := Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestBasicUnmarshal(t *testing.T) {
	data := []byte(`string = "Hello"

[subdoc]
string2 = "Howdy"
`)
	expected := basicMarshalTestStruct{
		String: "Hello",
		Sub: basicMarshalTestSubStruct{
			String2: "Howdy",
		},
	}
	result := basicMarshalTestStruct{}
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad unmarshal: expected %v, got %v", expected, result)
	}
}

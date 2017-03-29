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
  String2 = "One"

[[sublist]]
  String2 = "Two"

[[sublist]]
  String2 = "Three"
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
	Title       string            `toml:"title"`
	Basics      testDocBasics     `toml:"basic"`
	BasicLists  testDocBasicLists `toml:"basic_lists"`
	BasicMap    map[string]string `toml:"basic_map"`
	Subdocs     testDocSubs       `toml:"subdoc"`
	SubDocList  []testSubDoc      `toml:"subdoclist"`
	SubDocPtrs  []*testSubDoc     `toml:"subdocptrs"`
	err         int               `toml:"shouldntBeHere"`
	unexported  int               `toml:"shouldntBeHere"`
	Unexported2 int               `toml:"-"`
}

type testDocBasics struct {
	Bool       bool      `toml:"bool"`
	Date       time.Time `toml:"date"`
	Float      float32   `toml:"float"`
	Int        int       `toml:"int"`
	String     *string   `toml:"string"`
	unexported int       `toml:"shouldntBeHere"`
}

type testDocBasicLists struct {
	Bools   []bool      `toml:"bools"`
	Dates   []time.Time `toml:"dates"`
	Floats  []*float32  `toml:"floats"`
	Ints    []int       `toml:"ints"`
	Strings []string    `toml:"strings"`
}

type testDocSubs struct {
	First  testSubDoc  `toml:"first"`
	Second *testSubDoc `toml:"second"`
}

type testSubDoc struct {
	Name       string `toml:"name"`
	unexported int    `toml:"shouldntBeHere"`
}

var biteMe = "Bite me"
var float1 float32 = 12.3
var float2 float32 = 45.6
var float3 float32 = 78.9
var subdoc = testSubDoc{"Second", 0}

var docData = testDoc{
	Title:       "TOML Marshal Testing",
	unexported:  0,
	Unexported2: 0,
	Basics: testDocBasics{
		Bool:       true,
		Date:       time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
		Float:      123.4,
		Int:        5000,
		String:     &biteMe,
		unexported: 0,
	},
	BasicLists: testDocBasicLists{
		Bools: []bool{true, false, true},
		Dates: []time.Time{
			time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
			time.Date(1980, 5, 27, 7, 32, 0, 0, time.UTC),
		},
		Floats:  []*float32{&float1, &float2, &float3},
		Ints:    []int{8001, 8001, 8002},
		Strings: []string{"One", "Two", "Three"},
	},
	BasicMap: map[string]string{
		"one": "one",
		"two": "two",
	},
	Subdocs: testDocSubs{
		First:  testSubDoc{"First", 0},
		Second: &subdoc,
	},
	SubDocList: []testSubDoc{
		testSubDoc{"List.First", 0},
		testSubDoc{"List.Second", 0},
	},
	SubDocPtrs: []*testSubDoc{&subdoc},
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

type tomlTypeCheckTest struct {
	name string
	item interface{}
	typ  int //0=primitive, 1=otherslice, 2=treeslice, 3=tree
}

func TestTypeChecks(t *testing.T) {
	tests := []tomlTypeCheckTest{
		{"integer", 2, 0},
		{"time", time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC), 0},
		{"stringlist", []string{"hello", "hi"}, 1},
		{"timelist", []time.Time{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, 1},
		{"objectlist", []tomlTypeCheckTest{}, 2},
		{"object", tomlTypeCheckTest{}, 3},
	}

	for _, test := range tests {
		expected := []bool{false, false, false, false}
		expected[test.typ] = true
		result := []bool{
			isPrimitive(reflect.TypeOf(test.item)),
			isOtherSlice(reflect.TypeOf(test.item)),
			isTreeSlice(reflect.TypeOf(test.item)),
			isTree(reflect.TypeOf(test.item)),
		}
		if !reflect.DeepEqual(expected, result) {
			t.Errorf("Bad type check on %q: expected %v, got %v", test.name, expected, result)
		}
	}
}

type unexportedMarshalTestStruct struct {
	String      string                      `toml:"string"`
	StringList  []string                    `toml:"strlist"`
	Sub         basicMarshalTestSubStruct   `toml:"subdoc"`
	SubList     []basicMarshalTestSubStruct `toml:"sublist"`
	unexported  int                         `toml:"shouldntBeHere"`
	Unexported2 int                         `toml:"-"`
}

var unexportedTestData = unexportedMarshalTestStruct{
	String:      "Hello",
	StringList:  []string{"Howdy", "Hey There"},
	Sub:         basicMarshalTestSubStruct{"One"},
	SubList:     []basicMarshalTestSubStruct{{"Two"}, {"Three"}},
	unexported:  0,
	Unexported2: 0,
}

var unexportedTestToml = []byte(`string = "Hello"
strlist = ["Howdy","Hey There"]
unexported = 1
shouldntBeHere = 2

[subdoc]
  String2 = "One"

[[sublist]]
  String2 = "Two"

[[sublist]]
  String2 = "Three"
`)

func TestUnexportedUnmarshal(t *testing.T) {
	result := unexportedMarshalTestStruct{}
	err := Unmarshal(unexportedTestToml, &result)
	expected := unexportedTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad unexported unmarshal: expected %v, got %v", expected, result)
	}
}

type errStruct struct {
	Bool   bool      `toml:"bool"`
	Date   time.Time `toml:"date"`
	Float  float64   `toml:"float"`
	Int    int16     `toml:"int"`
	String *string   `toml:"string"`
}

var errTomls = []string{
	"bool = truly\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:3200Z\nfloat = 123.4\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123a4\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = j000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = 5000\nstring = Bite me",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = 5000\nstring = Bite me",
	"bool = 1\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1\nfloat = 123.4\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\n\"sorry\"\nint = 5000\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = \"sorry\"\nstring = \"Bite me\"",
	"bool = true\ndate = 1979-05-27T07:32:00Z\nfloat = 123.4\nint = 5000\nstring = 1",
}

type mapErr struct {
	Vals map[string]float64
}

func TestErrUnmarshal(t *testing.T) {
	for ind, toml := range errTomls {
		result := errStruct{}
		err := Unmarshal([]byte(toml), &result)
		if err == nil {
			t.Errorf("Expected err from case %d\n", ind)
		}
	}
	result2 := mapErr{}
	err := Unmarshal([]byte("[Vals]\nfred=\"1.2\""), &result2)
	if err == nil {
		t.Errorf("Expected err from map")
	}
}

type emptyMarshalTestStruct struct {
	Title      string                  `toml:"title"`
	Bool       bool                    `toml:"bool"`
	Int        int                     `toml:"int"`
	String     string                  `toml:"string"`
	StringList []string                `toml:"stringlist"`
	Ptr        *basicMarshalTestStruct `toml:"ptr"`
	Map        map[string]string       `toml:"map"`
}

var emptyTestData = emptyMarshalTestStruct{
	Title:      "Placeholder",
	Bool:       false,
	Int:        0,
	String:     "",
	StringList: []string{},
	Ptr:        nil,
	Map:        map[string]string{},
}

var emptyTestToml = []byte(`bool = false
int = 0
string = ""
stringlist = []
title = "Placeholder"

[map]
`)

type emptyMarshalTestStruct2 struct {
	Title      string                  `toml:"title"`
	Bool       bool                    `toml:"bool,omitempty"`
	Int        int                     `toml:"int, omitempty"`
	String     string                  `toml:"string,omitempty "`
	StringList []string                `toml:"stringlist,omitempty"`
	Ptr        *basicMarshalTestStruct `toml:"ptr,omitempty"`
	Map        map[string]string       `toml:"map,omitempty"`
}

var emptyTestData2 = emptyMarshalTestStruct2{
	Title:      "Placeholder",
	Bool:       false,
	Int:        0,
	String:     "",
	StringList: []string{},
	Ptr:        nil,
	Map:        map[string]string{},
}

var emptyTestToml2 = []byte(`title = "Placeholder"
`)

func TestEmptyMarshal(t *testing.T) {
	result, err := Marshal(emptyTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := emptyTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad empty marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestEmptyMarshalOmit(t *testing.T) {
	result, err := Marshal(emptyTestData2)
	if err != nil {
		t.Fatal(err)
	}
	expected := emptyTestToml2
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad empty omit marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestEmptyUnmarshal(t *testing.T) {
	result := emptyMarshalTestStruct{}
	err := Unmarshal(emptyTestToml, &result)
	expected := emptyTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad empty unmarshal: expected %v, got %v", expected, result)
	}
}

func TestEmptyUnmarshalOmit(t *testing.T) {
	result := emptyMarshalTestStruct2{}
	err := Unmarshal(emptyTestToml, &result)
	expected := emptyTestData2
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad empty omit unmarshal: expected %v, got %v", expected, result)
	}
}
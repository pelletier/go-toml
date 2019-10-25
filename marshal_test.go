package toml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type basicMarshalTestStruct struct {
	String     string                      `toml:"Zstring"`
	StringList []string                    `toml:"Ystrlist"`
	Sub        basicMarshalTestSubStruct   `toml:"Xsubdoc"`
	SubList    []basicMarshalTestSubStruct `toml:"Wsublist"`
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

var basicTestToml = []byte(`Ystrlist = ["Howdy","Hey There"]
Zstring = "Hello"

[[Wsublist]]
  String2 = "Two"

[[Wsublist]]
  String2 = "Three"

[Xsubdoc]
  String2 = "One"
`)

var basicTestTomlOrdered = []byte(`Zstring = "Hello"
Ystrlist = ["Howdy","Hey There"]

[Xsubdoc]
  String2 = "One"

[[Wsublist]]
  String2 = "Two"

[[Wsublist]]
  String2 = "Three"
`)

var marshalTestToml = []byte(`title = "TOML Marshal Testing"

[basic]
  bool = true
  date = 1979-05-27T07:32:00Z
  float = 123.4
  float64 = 123.456782132399
  int = 5000
  string = "Bite me"
  uint = 5001

[basic_lists]
  bools = [true,false,true]
  dates = [1979-05-27T07:32:00Z,1980-05-27T07:32:00Z]
  floats = [12.3,45.6,78.9]
  ints = [8001,8001,8002]
  strings = ["One","Two","Three"]
  uints = [5002,5003]

[basic_map]
  one = "one"
  two = "two"

[subdoc]

  [subdoc.first]
    name = "First"

  [subdoc.second]
    name = "Second"

[[subdoclist]]
  name = "List.First"

[[subdoclist]]
  name = "List.Second"

[[subdocptrs]]
  name = "Second"
`)

var marshalOrderPreserveToml = []byte(`title = "TOML Marshal Testing"

[basic_lists]
  floats = [12.3,45.6,78.9]
  bools = [true,false,true]
  dates = [1979-05-27T07:32:00Z,1980-05-27T07:32:00Z]
  ints = [8001,8001,8002]
  uints = [5002,5003]
  strings = ["One","Two","Three"]

[[subdocptrs]]
  name = "Second"

[basic_map]
  one = "one"
  two = "two"

[subdoc]

  [subdoc.second]
    name = "Second"

  [subdoc.first]
    name = "First"

[basic]
  uint = 5001
  bool = true
  float = 123.4
  float64 = 123.456782132399
  int = 5000
  string = "Bite me"
  date = 1979-05-27T07:32:00Z

[[subdoclist]]
  name = "List.First"

[[subdoclist]]
  name = "List.Second"
`)

var mashalOrderPreserveMapToml = []byte(`title = "TOML Marshal Testing"

[basic_map]
  one = "one"
  two = "two"

[long_map]
  a7 = "1"
  b3 = "2"
  c8 = "3"
  d4 = "4"
  e6 = "5"
  f5 = "6"
  g10 = "7"
  h1 = "8"
  i2 = "9"
  j9 = "10"
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

func TestBasicMarshalOrdered(t *testing.T) {
	var result bytes.Buffer
	err := NewEncoder(&result).Order(OrderPreserve).Encode(basicTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := basicTestTomlOrdered
	if !bytes.Equal(result.Bytes(), expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result.Bytes())
	}
}

func TestBasicMarshalWithPointer(t *testing.T) {
	result, err := Marshal(&basicTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := basicTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestBasicMarshalOrderedWithPointer(t *testing.T) {
	var result bytes.Buffer
	err := NewEncoder(&result).Order(OrderPreserve).Encode(&basicTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := basicTestTomlOrdered
	if !bytes.Equal(result.Bytes(), expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result.Bytes())
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
	BasicLists  testDocBasicLists `toml:"basic_lists"`
	SubDocPtrs  []*testSubDoc     `toml:"subdocptrs"`
	BasicMap    map[string]string `toml:"basic_map"`
	Subdocs     testDocSubs       `toml:"subdoc"`
	Basics      testDocBasics     `toml:"basic"`
	SubDocList  []testSubDoc      `toml:"subdoclist"`
	err         int               `toml:"shouldntBeHere"`
	unexported  int               `toml:"shouldntBeHere"`
	Unexported2 int               `toml:"-"`
}

type testMapDoc struct {
	Title    string            `toml:"title"`
	BasicMap map[string]string `toml:"basic_map"`
	LongMap  map[string]string `toml:"long_map"`
}

type testDocBasics struct {
	Uint       uint      `toml:"uint"`
	Bool       bool      `toml:"bool"`
	Float32    float32   `toml:"float"`
	Float64    float64   `toml:"float64"`
	Int        int       `toml:"int"`
	String     *string   `toml:"string"`
	Date       time.Time `toml:"date"`
	unexported int       `toml:"shouldntBeHere"`
}

type testDocBasicLists struct {
	Floats  []*float32  `toml:"floats"`
	Bools   []bool      `toml:"bools"`
	Dates   []time.Time `toml:"dates"`
	Ints    []int       `toml:"ints"`
	UInts   []uint      `toml:"uints"`
	Strings []string    `toml:"strings"`
}

type testDocSubs struct {
	Second *testSubDoc `toml:"second"`
	First  testSubDoc  `toml:"first"`
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
		Float32:    123.4,
		Float64:    123.456782132399,
		Int:        5000,
		Uint:       5001,
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
		UInts:   []uint{5002, 5003},
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
		{"List.First", 0},
		{"List.Second", 0},
	},
	SubDocPtrs: []*testSubDoc{&subdoc},
}

var mapTestDoc = testMapDoc{
	Title: "TOML Marshal Testing",
	BasicMap: map[string]string{
		"one": "one",
		"two": "two",
	},
	LongMap: map[string]string{
		"h1":  "8",
		"i2":  "9",
		"b3":  "2",
		"d4":  "4",
		"f5":  "6",
		"e6":  "5",
		"a7":  "1",
		"c8":  "3",
		"j9":  "10",
		"g10": "7",
	},
}

func TestDocMarshal(t *testing.T) {
	result, err := Marshal(docData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, marshalTestToml) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", marshalTestToml, result)
	}
}

func TestDocMarshalOrdered(t *testing.T) {
	var result bytes.Buffer
	err := NewEncoder(&result).Order(OrderPreserve).Encode(docData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Bytes(), marshalOrderPreserveToml) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", marshalOrderPreserveToml, result.Bytes())
	}
}

func TestDocMarshalMaps(t *testing.T) {
	result, err := Marshal(mapTestDoc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, mashalOrderPreserveMapToml) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", mashalOrderPreserveMapToml, result)
	}
}

func TestDocMarshalOrderedMaps(t *testing.T) {
	var result bytes.Buffer
	err := NewEncoder(&result).Order(OrderPreserve).Encode(mapTestDoc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Bytes(), mashalOrderPreserveMapToml) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", mashalOrderPreserveMapToml, result.Bytes())
	}
}

func TestDocMarshalPointer(t *testing.T) {
	result, err := Marshal(&docData)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result, marshalTestToml) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", marshalTestToml, result)
	}
}

func TestDocUnmarshal(t *testing.T) {
	result := testDoc{}
	err := Unmarshal(marshalTestToml, &result)
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

func TestDocPartialUnmarshal(t *testing.T) {
	file, err := ioutil.TempFile("", "test-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	err = ioutil.WriteFile(file.Name(), marshalTestToml, 0)
	if err != nil {
		t.Fatal(err)
	}

	tree, _ := LoadFile(file.Name())
	subTree := tree.Get("subdoc").(*Tree)

	result := testDocSubs{}
	err = subTree.Unmarshal(&result)
	expected := docData.Subdocs
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		resStr, _ := json.MarshalIndent(result, "", "  ")
		expStr, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("Bad partial unmartial: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expStr, resStr)
	}
}

type tomlTypeCheckTest struct {
	name string
	item interface{}
	typ  int //0=primitive, 1=otherslice, 2=treeslice, 3=tree
}

func TestTypeChecks(t *testing.T) {
	tests := []tomlTypeCheckTest{
		{"bool", true, 0},
		{"bool", false, 0},
		{"int", int(2), 0},
		{"int8", int8(2), 0},
		{"int16", int16(2), 0},
		{"int32", int32(2), 0},
		{"int64", int64(2), 0},
		{"uint", uint(2), 0},
		{"uint8", uint8(2), 0},
		{"uint16", uint16(2), 0},
		{"uint32", uint32(2), 0},
		{"uint64", uint64(2), 0},
		{"float32", float32(3.14), 0},
		{"float64", float64(3.14), 0},
		{"string", "lorem ipsum", 0},
		{"time", time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC), 0},
		{"stringlist", []string{"hello", "hi"}, 1},
		{"stringlistptr", &[]string{"hello", "hi"}, 1},
		{"stringarray", [2]string{"hello", "hi"}, 1},
		{"stringarrayptr", &[2]string{"hello", "hi"}, 1},
		{"timelist", []time.Time{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, 1},
		{"timelistptr", &[]time.Time{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, 1},
		{"timearray", [1]time.Time{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, 1},
		{"timearrayptr", &[1]time.Time{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, 1},
		{"objectlist", []tomlTypeCheckTest{}, 2},
		{"objectlistptr", &[]tomlTypeCheckTest{}, 2},
		{"objectarray", [2]tomlTypeCheckTest{{}, {}}, 2},
		{"objectlistptr", &[2]tomlTypeCheckTest{{}, {}}, 2},
		{"object", tomlTypeCheckTest{}, 3},
		{"objectptr", &tomlTypeCheckTest{}, 3},
	}

	for _, test := range tests {
		expected := []bool{false, false, false, false}
		expected[test.typ] = true
		result := []bool{
			isPrimitive(reflect.TypeOf(test.item)),
			isOtherSequence(reflect.TypeOf(test.item)),
			isTreeSequence(reflect.TypeOf(test.item)),
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

type intErr struct {
	Int1  int
	Int2  int8
	Int3  int16
	Int4  int32
	Int5  int64
	UInt1 uint
	UInt2 uint8
	UInt3 uint16
	UInt4 uint32
	UInt5 uint64
	Flt1  float32
	Flt2  float64
}

var intErrTomls = []string{
	"Int1 = []\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = []\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = []\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = []\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = []\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = []\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = []\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = []\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = []\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = []\nFlt1 = 1.0\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = []\nFlt2 = 2.0",
	"Int1 = 1\nInt2 = 2\nInt3 = 3\nInt4 = 4\nInt5 = 5\nUInt1 = 1\nUInt2 = 2\nUInt3 = 3\nUInt4 = 4\nUInt5 = 5\nFlt1 = 1.0\nFlt2 = []",
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
	for ind, toml := range intErrTomls {
		result3 := intErr{}
		err := Unmarshal([]byte(toml), &result3)
		if err == nil {
			t.Errorf("Expected int err from case %d\n", ind)
		}
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

type pointerMarshalTestStruct struct {
	Str       *string
	List      *[]string
	ListPtr   *[]*string
	Map       *map[string]string
	MapPtr    *map[string]*string
	EmptyStr  *string
	EmptyList *[]string
	EmptyMap  *map[string]string
	DblPtr    *[]*[]*string
}

var pointerStr = "Hello"
var pointerList = []string{"Hello back"}
var pointerListPtr = []*string{&pointerStr}
var pointerMap = map[string]string{"response": "Goodbye"}
var pointerMapPtr = map[string]*string{"alternate": &pointerStr}
var pointerTestData = pointerMarshalTestStruct{
	Str:       &pointerStr,
	List:      &pointerList,
	ListPtr:   &pointerListPtr,
	Map:       &pointerMap,
	MapPtr:    &pointerMapPtr,
	EmptyStr:  nil,
	EmptyList: nil,
	EmptyMap:  nil,
}

var pointerTestToml = []byte(`List = ["Hello back"]
ListPtr = ["Hello"]
Str = "Hello"

[Map]
  response = "Goodbye"

[MapPtr]
  alternate = "Hello"
`)

func TestPointerMarshal(t *testing.T) {
	result, err := Marshal(pointerTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := pointerTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad pointer marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestPointerUnmarshal(t *testing.T) {
	result := pointerMarshalTestStruct{}
	err := Unmarshal(pointerTestToml, &result)
	expected := pointerTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad pointer unmarshal: expected %v, got %v", expected, result)
	}
}

func TestUnmarshalTypeMismatch(t *testing.T) {
	result := pointerMarshalTestStruct{}
	err := Unmarshal([]byte("List = 123"), &result)
	if !strings.HasPrefix(err.Error(), "(1, 1): Can't convert 123(int64) to []string(slice)") {
		t.Errorf("Type mismatch must be reported: got %v", err.Error())
	}
}

type nestedMarshalTestStruct struct {
	String [][]string
	//Struct [][]basicMarshalTestSubStruct
	StringPtr *[]*[]*string
	// StructPtr *[]*[]*basicMarshalTestSubStruct
}

var str1 = "Three"
var str2 = "Four"
var strPtr = []*string{&str1, &str2}
var strPtr2 = []*[]*string{&strPtr}

var nestedTestData = nestedMarshalTestStruct{
	String:    [][]string{{"Five", "Six"}, {"One", "Two"}},
	StringPtr: &strPtr2,
}

var nestedTestToml = []byte(`String = [["Five","Six"],["One","Two"]]
StringPtr = [["Three","Four"]]
`)

func TestNestedMarshal(t *testing.T) {
	result, err := Marshal(nestedTestData)
	if err != nil {
		t.Fatal(err)
	}
	expected := nestedTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad nested marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestNestedUnmarshal(t *testing.T) {
	result := nestedMarshalTestStruct{}
	err := Unmarshal(nestedTestToml, &result)
	expected := nestedTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad nested unmarshal: expected %v, got %v", expected, result)
	}
}

type customMarshalerParent struct {
	Self    customMarshaler   `toml:"me"`
	Friends []customMarshaler `toml:"friends"`
}

type customMarshaler struct {
	FirsName string
	LastName string
}

func (c customMarshaler) MarshalTOML() ([]byte, error) {
	fullName := fmt.Sprintf("%s %s", c.FirsName, c.LastName)
	return []byte(fullName), nil
}

var customMarshalerData = customMarshaler{FirsName: "Sally", LastName: "Fields"}
var customMarshalerToml = []byte(`Sally Fields`)
var nestedCustomMarshalerData = customMarshalerParent{
	Self:    customMarshaler{FirsName: "Maiku", LastName: "Suteda"},
	Friends: []customMarshaler{customMarshalerData},
}
var nestedCustomMarshalerToml = []byte(`friends = ["Sally Fields"]
me = "Maiku Suteda"
`)

func TestCustomMarshaler(t *testing.T) {
	result, err := Marshal(customMarshalerData)
	if err != nil {
		t.Fatal(err)
	}
	expected := customMarshalerToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad custom marshaler: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestNestedCustomMarshaler(t *testing.T) {
	result, err := Marshal(nestedCustomMarshalerData)
	if err != nil {
		t.Fatal(err)
	}
	expected := nestedCustomMarshalerToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad nested custom marshaler: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

var commentTestToml = []byte(`
# it's a comment on type
[postgres]
  # isCommented = "dvalue"
  noComment = "cvalue"

  # A comment on AttrB with a
  # break line
  password = "bvalue"

  # A comment on AttrA
  user = "avalue"

  [[postgres.My]]

    # a comment on my on typeC
    My = "Foo"

  [[postgres.My]]

    # a comment on my on typeC
    My = "Baar"
`)

func TestMarshalComment(t *testing.T) {
	type TypeC struct {
		My string `comment:"a comment on my on typeC"`
	}
	type TypeB struct {
		AttrA string `toml:"user" comment:"A comment on AttrA"`
		AttrB string `toml:"password" comment:"A comment on AttrB with a\n break line"`
		AttrC string `toml:"noComment"`
		AttrD string `toml:"isCommented" commented:"true"`
		My    []TypeC
	}
	type TypeA struct {
		TypeB TypeB `toml:"postgres" comment:"it's a comment on type"`
	}

	ta := []TypeC{{My: "Foo"}, {My: "Baar"}}
	config := TypeA{TypeB{AttrA: "avalue", AttrB: "bvalue", AttrC: "cvalue", AttrD: "dvalue", My: ta}}
	result, err := Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := commentTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

type mapsTestStruct struct {
	Simple map[string]string
	Paths  map[string]string
	Other  map[string]float64
	X      struct {
		Y struct {
			Z map[string]bool
		}
	}
}

var mapsTestData = mapsTestStruct{
	Simple: map[string]string{
		"one plus one": "two",
		"next":         "three",
	},
	Paths: map[string]string{
		"/this/is/a/path": "/this/is/also/a/path",
		"/heloo.txt":      "/tmp/lololo.txt",
	},
	Other: map[string]float64{
		"testing": 3.9999,
	},
	X: struct{ Y struct{ Z map[string]bool } }{
		Y: struct{ Z map[string]bool }{
			Z: map[string]bool{
				"is.Nested": true,
			},
		},
	},
}
var mapsTestToml = []byte(`
[Other]
  "testing" = 3.9999

[Paths]
  "/heloo.txt" = "/tmp/lololo.txt"
  "/this/is/a/path" = "/this/is/also/a/path"

[Simple]
  "next" = "three"
  "one plus one" = "two"

[X]

  [X.Y]

    [X.Y.Z]
      "is.Nested" = true
`)

func TestEncodeQuotedMapKeys(t *testing.T) {
	var buf bytes.Buffer
	if err := NewEncoder(&buf).QuoteMapKeys(true).Encode(mapsTestData); err != nil {
		t.Fatal(err)
	}
	result := buf.Bytes()
	expected := mapsTestToml
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad maps marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestDecodeQuotedMapKeys(t *testing.T) {
	result := mapsTestStruct{}
	err := NewDecoder(bytes.NewBuffer(mapsTestToml)).Decode(&result)
	expected := mapsTestData
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad maps unmarshal: expected %v, got %v", expected, result)
	}
}

type structArrayNoTag struct {
	A struct {
		B []int64
		C []int64
	}
}

func TestMarshalArray(t *testing.T) {
	expected := []byte(`
[A]
  B = [1,2,3]
  C = [1]
`)

	m := structArrayNoTag{
		A: struct {
			B []int64
			C []int64
		}{
			B: []int64{1, 2, 3},
			C: []int64{1},
		},
	}

	b, err := Marshal(m)

	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, expected) {
		t.Errorf("Bad arrays marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, b)
	}
}

func TestMarshalArrayOnePerLine(t *testing.T) {
	expected := []byte(`
[A]
  B = [
    1,
    2,
    3,
  ]
  C = [1]
`)

	m := structArrayNoTag{
		A: struct {
			B []int64
			C []int64
		}{
			B: []int64{1, 2, 3},
			C: []int64{1},
		},
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf).ArraysWithOneElementPerLine(true)
	err := encoder.Encode(m)

	if err != nil {
		t.Fatal(err)
	}

	b := buf.Bytes()

	if !bytes.Equal(b, expected) {
		t.Errorf("Bad arrays marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, b)
	}
}

var customTagTestToml = []byte(`
[postgres]
  password = "bvalue"
  user = "avalue"

  [[postgres.My]]
    My = "Foo"

  [[postgres.My]]
    My = "Baar"
`)

func TestMarshalCustomTag(t *testing.T) {
	type TypeC struct {
		My string
	}
	type TypeB struct {
		AttrA string `file:"user"`
		AttrB string `file:"password"`
		My    []TypeC
	}
	type TypeA struct {
		TypeB TypeB `file:"postgres"`
	}

	ta := []TypeC{{My: "Foo"}, {My: "Baar"}}
	config := TypeA{TypeB{AttrA: "avalue", AttrB: "bvalue", My: ta}}
	var buf bytes.Buffer
	err := NewEncoder(&buf).SetTagName("file").Encode(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := customTagTestToml
	result := buf.Bytes()
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

var customCommentTagTestToml = []byte(`
# db connection
[postgres]

  # db pass
  password = "bvalue"

  # db user
  user = "avalue"
`)

func TestMarshalCustomComment(t *testing.T) {
	type TypeB struct {
		AttrA string `toml:"user" descr:"db user"`
		AttrB string `toml:"password" descr:"db pass"`
	}
	type TypeA struct {
		TypeB TypeB `toml:"postgres" descr:"db connection"`
	}

	config := TypeA{TypeB{AttrA: "avalue", AttrB: "bvalue"}}
	var buf bytes.Buffer
	err := NewEncoder(&buf).SetTagComment("descr").Encode(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := customCommentTagTestToml
	result := buf.Bytes()
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

var customCommentedTagTestToml = []byte(`
[postgres]
  # password = "bvalue"
  # user = "avalue"
`)

func TestMarshalCustomCommented(t *testing.T) {
	type TypeB struct {
		AttrA string `toml:"user" disable:"true"`
		AttrB string `toml:"password" disable:"true"`
	}
	type TypeA struct {
		TypeB TypeB `toml:"postgres"`
	}

	config := TypeA{TypeB{AttrA: "avalue", AttrB: "bvalue"}}
	var buf bytes.Buffer
	err := NewEncoder(&buf).SetTagCommented("disable").Encode(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := customCommentedTagTestToml
	result := buf.Bytes()
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

func TestMarshalDirectMultilineString(t *testing.T) {
	tree := newTree()
	tree.SetWithOptions("mykey", SetOptions{
		Multiline: true,
	}, "my\x11multiline\nstring\ba\tb\fc\rd\"e\\!")
	result, err := tree.Marshal()
	if err != nil {
		t.Fatal("marshal should not error:", err)
	}
	expected := []byte("mykey = \"\"\"\nmy\\u0011multiline\nstring\\ba\tb\\fc\rd\"e\\!\"\"\"\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

var customMultilineTagTestToml = []byte(`int_slice = [
  1,
  2,
  3,
]
`)

func TestMarshalCustomMultiline(t *testing.T) {
	type TypeA struct {
		AttrA []int `toml:"int_slice" mltln:"true"`
	}

	config := TypeA{AttrA: []int{1, 2, 3}}
	var buf bytes.Buffer
	err := NewEncoder(&buf).ArraysWithOneElementPerLine(true).SetTagMultiline("mltln").Encode(config)
	if err != nil {
		t.Fatal(err)
	}
	expected := customMultilineTagTestToml
	result := buf.Bytes()
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

var testDocBasicToml = []byte(`
[document]
  bool_val = true
  date_val = 1979-05-27T07:32:00Z
  float_val = 123.4
  int_val = 5000
  string_val = "Bite me"
  uint_val = 5001
`)

type testDocCustomTag struct {
	Doc testDocBasicsCustomTag `file:"document"`
}
type testDocBasicsCustomTag struct {
	Bool       bool      `file:"bool_val"`
	Date       time.Time `file:"date_val"`
	Float      float32   `file:"float_val"`
	Int        int       `file:"int_val"`
	Uint       uint      `file:"uint_val"`
	String     *string   `file:"string_val"`
	unexported int       `file:"shouldntBeHere"`
}

var testDocCustomTagData = testDocCustomTag{
	Doc: testDocBasicsCustomTag{
		Bool:       true,
		Date:       time.Date(1979, 5, 27, 7, 32, 0, 0, time.UTC),
		Float:      123.4,
		Int:        5000,
		Uint:       5001,
		String:     &biteMe,
		unexported: 0,
	},
}

func TestUnmarshalCustomTag(t *testing.T) {
	buf := bytes.NewBuffer(testDocBasicToml)

	result := testDocCustomTag{}
	err := NewDecoder(buf).SetTagName("file").Decode(&result)
	if err != nil {
		t.Fatal(err)
	}
	expected := testDocCustomTagData
	if !reflect.DeepEqual(result, expected) {
		resStr, _ := json.MarshalIndent(result, "", "  ")
		expStr, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("Bad unmarshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expStr, resStr)

	}
}

func TestUnmarshalMap(t *testing.T) {
	testToml := []byte(`
		a = 1
		b = 2
		c = 3
		`)
	var result map[string]int
	err := Unmarshal(testToml, &result)
	if err != nil {
		t.Errorf("Received unexpected error: %s", err)
		return
	}

	expected := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad unmarshal: expected %v, got %v", expected, result)
	}
}

func TestUnmarshalMapWithTypedKey(t *testing.T) {
	testToml := []byte(`
		a = 1
		b = 2
		c = 3
		`)

	type letter string
	var result map[letter]int
	err := Unmarshal(testToml, &result)
	if err != nil {
		t.Errorf("Received unexpected error: %s", err)
		return
	}

	expected := map[letter]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Bad unmarshal: expected %v, got %v", expected, result)
	}
}

func TestUnmarshalNonPointer(t *testing.T) {
	a := 1
	err := Unmarshal([]byte{}, a)
	if err == nil {
		t.Fatal("unmarshal should err when given a non pointer")
	}
}

func TestUnmarshalInvalidPointerKind(t *testing.T) {
	a := 1
	err := Unmarshal([]byte{}, &a)
	if err == nil {
		t.Fatal("unmarshal should err when given an invalid pointer type")
	}
}

func TestMarshalSlice(t *testing.T) {
	m := make([]int, 1)
	m[0] = 1

	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(&m)
	if err == nil {
		t.Error("expected error, got nil")
		return
	}
	if err.Error() != "Only pointer to struct can be marshaled to TOML" {
		t.Fail()
	}
}

func TestMarshalSlicePointer(t *testing.T) {
	m := make([]int, 1)
	m[0] = 1

	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(m)
	if err == nil {
		t.Error("expected error, got nil")
		return
	}
	if err.Error() != "Only a struct or map can be marshaled to TOML" {
		t.Fail()
	}
}

type testDuration struct {
	Nanosec   time.Duration  `toml:"nanosec"`
	Microsec1 time.Duration  `toml:"microsec1"`
	Microsec2 *time.Duration `toml:"microsec2"`
	Millisec  time.Duration  `toml:"millisec"`
	Sec       time.Duration  `toml:"sec"`
	Min       time.Duration  `toml:"min"`
	Hour      time.Duration  `toml:"hour"`
	Mixed     time.Duration  `toml:"mixed"`
	AString   string         `toml:"a_string"`
}

var testDurationToml = []byte(`
nanosec = "1ns"
microsec1 = "1us"
microsec2 = "1µs"
millisec = "1ms"
sec = "1s"
min = "1m"
hour = "1h"
mixed = "1h1m1s1ms1µs1ns"
a_string = "15s"
`)

func TestUnmarshalDuration(t *testing.T) {
	buf := bytes.NewBuffer(testDurationToml)

	result := testDuration{}
	err := NewDecoder(buf).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}
	ms := time.Duration(1) * time.Microsecond
	expected := testDuration{
		Nanosec:   1,
		Microsec1: time.Microsecond,
		Microsec2: &ms,
		Millisec:  time.Millisecond,
		Sec:       time.Second,
		Min:       time.Minute,
		Hour:      time.Hour,
		Mixed: time.Hour +
			time.Minute +
			time.Second +
			time.Millisecond +
			time.Microsecond +
			time.Nanosecond,
		AString: "15s",
	}
	if !reflect.DeepEqual(result, expected) {
		resStr, _ := json.MarshalIndent(result, "", "  ")
		expStr, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("Bad unmarshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expStr, resStr)

	}
}

var testDurationToml2 = []byte(`a_string = "15s"
hour = "1h0m0s"
microsec1 = "1µs"
microsec2 = "1µs"
millisec = "1ms"
min = "1m0s"
mixed = "1h1m1.001001001s"
nanosec = "1ns"
sec = "1s"
`)

func TestMarshalDuration(t *testing.T) {
	ms := time.Duration(1) * time.Microsecond
	data := testDuration{
		Nanosec:   1,
		Microsec1: time.Microsecond,
		Microsec2: &ms,
		Millisec:  time.Millisecond,
		Sec:       time.Second,
		Min:       time.Minute,
		Hour:      time.Hour,
		Mixed: time.Hour +
			time.Minute +
			time.Second +
			time.Millisecond +
			time.Microsecond +
			time.Nanosecond,
		AString: "15s",
	}

	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := testDurationToml2
	result := buf.Bytes()
	if !bytes.Equal(result, expected) {
		t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
	}
}

type testBadDuration struct {
	Val time.Duration `toml:"val"`
}

var testBadDurationToml = []byte(`val = "1z"`)

func TestUnmarshalBadDuration(t *testing.T) {
	buf := bytes.NewBuffer(testBadDurationToml)

	result := testBadDuration{}
	err := NewDecoder(buf).Decode(&result)
	if err == nil {
		t.Fatal()
	}
	if err.Error() != "(1, 1): Can't convert 1z(string) to time.Duration. time: unknown unit z in duration 1z" {
		t.Fatalf("unexpected error: %s", err)
	}
}

var testCamelCaseKeyToml = []byte(`fooBar = 10`)

func TestUnmarshalCamelCaseKey(t *testing.T) {
	var x struct {
		FooBar int
		B      int
	}

	if err := Unmarshal(testCamelCaseKeyToml, &x); err != nil {
		t.Fatal(err)
	}

	if x.FooBar != 10 {
		t.Fatal("Did not set camelCase'd key")
	}
}

func TestUnmarshalDefault(t *testing.T) {
	var doc struct {
		StringField  string  `default:"a"`
		BoolField    bool    `default:"true"`
		IntField     int     `default:"1"`
		Int64Field   int64   `default:"2"`
		Float64Field float64 `default:"3.1"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err != nil {
		t.Fatal(err)
	}
	if doc.BoolField != true {
		t.Errorf("BoolField should be true, not %t", doc.BoolField)
	}
	if doc.StringField != "a" {
		t.Errorf("StringField should be \"a\", not %s", doc.StringField)
	}
	if doc.IntField != 1 {
		t.Errorf("IntField should be 1, not %d", doc.IntField)
	}
	if doc.Int64Field != 2 {
		t.Errorf("Int64Field should be 2, not %d", doc.Int64Field)
	}
	if doc.Float64Field != 3.1 {
		t.Errorf("Float64Field should be 3.1, not %f", doc.Float64Field)
	}
}

func TestUnmarshalDefaultFailureBool(t *testing.T) {
	var doc struct {
		Field bool `default:"blah"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestUnmarshalDefaultFailureInt(t *testing.T) {
	var doc struct {
		Field int `default:"blah"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestUnmarshalDefaultFailureInt64(t *testing.T) {
	var doc struct {
		Field int64 `default:"blah"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestUnmarshalDefaultFailureFloat64(t *testing.T) {
	var doc struct {
		Field float64 `default:"blah"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestUnmarshalDefaultFailureUnsupported(t *testing.T) {
	var doc struct {
		Field struct{} `default:"blah"`
	}

	err := Unmarshal([]byte(``), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestUnmarshalNestedAnonymousStructs(t *testing.T) {
	type Nested struct {
		Value string `toml:"nested_field"`
	}
	type Deep struct {
		Nested
	}
	type Document struct {
		Deep
		Value string `toml:"own_field"`
	}

	var doc Document

	err := Unmarshal([]byte(`nested_field = "nested value"`+"\n"+`own_field = "own value"`), &doc)
	if err != nil {
		t.Fatal("should not error")
	}
	if doc.Value != "own value" || doc.Nested.Value != "nested value" {
		t.Fatal("unexpected values")
	}
}

func TestUnmarshalNestedAnonymousStructs_Controversial(t *testing.T) {
	type Nested struct {
		Value string `toml:"nested"`
	}
	type Deep struct {
		Nested
	}
	type Document struct {
		Deep
		Value string `toml:"own"`
	}

	var doc Document

	err := Unmarshal([]byte(`nested = "nested value"`+"\n"+`own = "own value"`), &doc)
	if err == nil {
		t.Fatal("should error")
	}
}

type unexportedFieldPreservationTest struct {
	Exported   string `toml:"exported"`
	unexported string
	Nested1    unexportedFieldPreservationTestNested    `toml:"nested1"`
	Nested2    *unexportedFieldPreservationTestNested   `toml:"nested2"`
	Nested3    *unexportedFieldPreservationTestNested   `toml:"nested3"`
	Slice1     []unexportedFieldPreservationTestNested  `toml:"slice1"`
	Slice2     []*unexportedFieldPreservationTestNested `toml:"slice2"`
}

type unexportedFieldPreservationTestNested struct {
	Exported1   string `toml:"exported1"`
	unexported1 string
}

func TestUnmarshalPreservesUnexportedFields(t *testing.T) {
	toml := `
	exported = "visible"
	unexported = "ignored"

	[nested1]
	exported1 = "visible1"
	unexported1 = "ignored1"

	[nested2]
	exported1 = "visible2"
	unexported1 = "ignored2"

	[nested3]
	exported1 = "visible3"
	unexported1 = "ignored3"

	[[slice1]]
	exported1 = "visible3"
	
	[[slice1]]
	exported1 = "visible4"

	[[slice2]]
	exported1 = "visible5"
	`

	t.Run("unexported field should not be set from toml", func(t *testing.T) {
		var actual unexportedFieldPreservationTest
		err := Unmarshal([]byte(toml), &actual)

		if err != nil {
			t.Fatal("did not expect an error")
		}

		expect := unexportedFieldPreservationTest{
			Exported:   "visible",
			unexported: "",
			Nested1:    unexportedFieldPreservationTestNested{"visible1", ""},
			Nested2:    &unexportedFieldPreservationTestNested{"visible2", ""},
			Nested3:    &unexportedFieldPreservationTestNested{"visible3", ""},
			Slice1: []unexportedFieldPreservationTestNested{
				{Exported1: "visible3"},
				{Exported1: "visible4"},
			},
			Slice2: []*unexportedFieldPreservationTestNested{
				{Exported1: "visible5"},
			},
		}

		if !reflect.DeepEqual(actual, expect) {
			t.Fatalf("%+v did not equal %+v", actual, expect)
		}
	})

	t.Run("unexported field should be preserved", func(t *testing.T) {
		actual := unexportedFieldPreservationTest{
			Exported:   "foo",
			unexported: "bar",
			Nested1:    unexportedFieldPreservationTestNested{"baz", "bax"},
			Nested2:    nil,
			Nested3:    &unexportedFieldPreservationTestNested{"baz", "bax"},
		}
		err := Unmarshal([]byte(toml), &actual)

		if err != nil {
			t.Fatal("did not expect an error")
		}

		expect := unexportedFieldPreservationTest{
			Exported:   "visible",
			unexported: "bar",
			Nested1:    unexportedFieldPreservationTestNested{"visible1", "bax"},
			Nested2:    &unexportedFieldPreservationTestNested{"visible2", ""},
			Nested3:    &unexportedFieldPreservationTestNested{"visible3", "bax"},
			Slice1: []unexportedFieldPreservationTestNested{
				{Exported1: "visible3"},
				{Exported1: "visible4"},
			},
			Slice2: []*unexportedFieldPreservationTestNested{
				{Exported1: "visible5"},
			},
		}

		if !reflect.DeepEqual(actual, expect) {
			t.Fatalf("%+v did not equal %+v", actual, expect)
		}
	})
}

func TestTreeMarshal(t *testing.T) {
	cases := [][]byte{
		basicTestToml,
		marshalTestToml,
		emptyTestToml,
		pointerTestToml,
	}
	for _, expected := range cases {
		t.Run("", func(t *testing.T) {
			tree, err := LoadBytes(expected)
			if err != nil {
				t.Fatal(err)
			}
			result, err := tree.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(result, expected) {
				t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", expected, result)
			}
		})
	}
}

func TestMarshalArrays(t *testing.T) {
	cases := []struct {
		Data     interface{}
		Expected string
	}{
		{
			Data: struct {
				XY [2]int
			}{
				XY: [2]int{1, 2},
			},
			Expected: `XY = [1,2]
`,
		},
		{
			Data: struct {
				XY [1][2]int
			}{
				XY: [1][2]int{{1, 2}},
			},
			Expected: `XY = [[1,2]]
`,
		},
		{
			Data: struct {
				XY [1][]int
			}{
				XY: [1][]int{{1, 2}},
			},
			Expected: `XY = [[1,2]]
`,
		},
		{
			Data: struct {
				XY [][2]int
			}{
				XY: [][2]int{{1, 2}},
			},
			Expected: `XY = [[1,2]]
`,
		},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			result, err := Marshal(tc.Data)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(result, []byte(tc.Expected)) {
				t.Errorf("Bad marshal: expected\n-----\n%s\n-----\ngot\n-----\n%s\n-----\n", []byte(tc.Expected), result)
			}
		})
	}
}

func TestUnmarshalLocalDate(t *testing.T) {
	t.Run("ToLocalDate", func(t *testing.T) {
		type dateStruct struct {
			Date LocalDate
		}

		toml := `date = 1979-05-27`

		var obj dateStruct

		err := Unmarshal([]byte(toml), &obj)

		if err != nil {
			t.Fatal(err)
		}

		if obj.Date.Year != 1979 {
			t.Errorf("expected year 1979, got %d", obj.Date.Year)
		}
		if obj.Date.Month != 5 {
			t.Errorf("expected month 5, got %d", obj.Date.Month)
		}
		if obj.Date.Day != 27 {
			t.Errorf("expected day 27, got %d", obj.Date.Day)
		}
	})

	t.Run("ToLocalDate", func(t *testing.T) {
		type dateStruct struct {
			Date time.Time
		}

		toml := `date = 1979-05-27`

		var obj dateStruct

		err := Unmarshal([]byte(toml), &obj)

		if err != nil {
			t.Fatal(err)
		}

		if obj.Date.Year() != 1979 {
			t.Errorf("expected year 1979, got %d", obj.Date.Year())
		}
		if obj.Date.Month() != 5 {
			t.Errorf("expected month 5, got %d", obj.Date.Month())
		}
		if obj.Date.Day() != 27 {
			t.Errorf("expected day 27, got %d", obj.Date.Day())
		}
	})
}

func TestMarshalLocalDate(t *testing.T) {
	type dateStruct struct {
		Date LocalDate
	}

	obj := dateStruct{Date: LocalDate{
		Year:  1979,
		Month: 5,
		Day:   27,
	}}

	b, err := Marshal(obj)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(b)
	expected := `Date = 1979-05-27
`

	if got != expected {
		t.Errorf("expected '%s', got '%s'", expected, got)
	}
}

func TestUnmarshalLocalDateTime(t *testing.T) {
	examples := []struct {
		name string
		in   string
		out  LocalDateTime
	}{
		{
			name: "normal",
			in:   "1979-05-27T07:32:00",
			out: LocalDateTime{
				Date: LocalDate{
					Year:  1979,
					Month: 5,
					Day:   27,
				},
				Time: LocalTime{
					Hour:       7,
					Minute:     32,
					Second:     0,
					Nanosecond: 0,
				},
			}},
		{
			name: "with nanoseconds",
			in:   "1979-05-27T00:32:00.999999",
			out: LocalDateTime{
				Date: LocalDate{
					Year:  1979,
					Month: 5,
					Day:   27,
				},
				Time: LocalTime{
					Hour:       0,
					Minute:     32,
					Second:     0,
					Nanosecond: 999999000,
				},
			},
		},
	}

	for i, example := range examples {
		toml := fmt.Sprintf(`date = %s`, example.in)

		t.Run(fmt.Sprintf("ToLocalDateTime_%d_%s", i, example.name), func(t *testing.T) {
			type dateStruct struct {
				Date LocalDateTime
			}

			var obj dateStruct

			err := Unmarshal([]byte(toml), &obj)

			if err != nil {
				t.Fatal(err)
			}

			if obj.Date != example.out {
				t.Errorf("expected '%s', got '%s'", example.out, obj.Date)
			}
		})

		t.Run(fmt.Sprintf("ToTime_%d_%s", i, example.name), func(t *testing.T) {
			type dateStruct struct {
				Date time.Time
			}

			var obj dateStruct

			err := Unmarshal([]byte(toml), &obj)

			if err != nil {
				t.Fatal(err)
			}

			if obj.Date.Year() != example.out.Date.Year {
				t.Errorf("expected year %d, got %d", example.out.Date.Year, obj.Date.Year())
			}
			if obj.Date.Month() != example.out.Date.Month {
				t.Errorf("expected month %d, got %d", example.out.Date.Month, obj.Date.Month())
			}
			if obj.Date.Day() != example.out.Date.Day {
				t.Errorf("expected day %d, got %d", example.out.Date.Day, obj.Date.Day())
			}
			if obj.Date.Hour() != example.out.Time.Hour {
				t.Errorf("expected hour %d, got %d", example.out.Time.Hour, obj.Date.Hour())
			}
			if obj.Date.Minute() != example.out.Time.Minute {
				t.Errorf("expected minute %d, got %d", example.out.Time.Minute, obj.Date.Minute())
			}
			if obj.Date.Second() != example.out.Time.Second {
				t.Errorf("expected second %d, got %d", example.out.Time.Second, obj.Date.Second())
			}
			if obj.Date.Nanosecond() != example.out.Time.Nanosecond {
				t.Errorf("expected nanoseconds %d, got %d", example.out.Time.Nanosecond, obj.Date.Nanosecond())
			}
		})
	}
}

func TestMarshalLocalDateTime(t *testing.T) {
	type dateStruct struct {
		DateTime LocalDateTime
	}

	examples := []struct {
		name string
		in   LocalDateTime
		out  string
	}{
		{
			name: "normal",
			out:  "DateTime = 1979-05-27T07:32:00\n",
			in: LocalDateTime{
				Date: LocalDate{
					Year:  1979,
					Month: 5,
					Day:   27,
				},
				Time: LocalTime{
					Hour:       7,
					Minute:     32,
					Second:     0,
					Nanosecond: 0,
				},
			}},
		{
			name: "with nanoseconds",
			out:  "DateTime = 1979-05-27T00:32:00.999999000\n",
			in: LocalDateTime{
				Date: LocalDate{
					Year:  1979,
					Month: 5,
					Day:   27,
				},
				Time: LocalTime{
					Hour:       0,
					Minute:     32,
					Second:     0,
					Nanosecond: 999999000,
				},
			},
		},
	}

	for i, example := range examples {
		t.Run(fmt.Sprintf("%d_%s", i, example.name), func(t *testing.T) {
			obj := dateStruct{
				DateTime: example.in,
			}
			b, err := Marshal(obj)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := string(b)

			if got != example.out {
				t.Errorf("expected '%s', got '%s'", example.out, got)
			}
		})
	}
}

func TestUnmarshalLocalTime(t *testing.T) {
	examples := []struct {
		name string
		in   string
		out  LocalTime
	}{
		{
			name: "normal",
			in:   "07:32:00",
			out: LocalTime{
				Hour:       7,
				Minute:     32,
				Second:     0,
				Nanosecond: 0,
			},
		},
		{
			name: "with nanoseconds",
			in:   "00:32:00.999999",
			out: LocalTime{
				Hour:       0,
				Minute:     32,
				Second:     0,
				Nanosecond: 999999000,
			},
		},
	}

	for i, example := range examples {
		toml := fmt.Sprintf(`Time = %s`, example.in)

		t.Run(fmt.Sprintf("ToLocalTime_%d_%s", i, example.name), func(t *testing.T) {
			type dateStruct struct {
				Time LocalTime
			}

			var obj dateStruct

			err := Unmarshal([]byte(toml), &obj)

			if err != nil {
				t.Fatal(err)
			}

			if obj.Time != example.out {
				t.Errorf("expected '%s', got '%s'", example.out, obj.Time)
			}
		})
	}
}

func TestMarshalLocalTime(t *testing.T) {
	type timeStruct struct {
		Time LocalTime
	}

	examples := []struct {
		name string
		in   LocalTime
		out  string
	}{
		{
			name: "normal",
			out:  "Time = 07:32:00\n",
			in: LocalTime{
				Hour:       7,
				Minute:     32,
				Second:     0,
				Nanosecond: 0,
			}},
		{
			name: "with nanoseconds",
			out:  "Time = 00:32:00.999999000\n",
			in: LocalTime{
				Hour:       0,
				Minute:     32,
				Second:     0,
				Nanosecond: 999999000,
			},
		},
	}

	for i, example := range examples {
		t.Run(fmt.Sprintf("%d_%s", i, example.name), func(t *testing.T) {
			obj := timeStruct{
				Time: example.in,
			}
			b, err := Marshal(obj)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := string(b)

			if got != example.out {
				t.Errorf("expected '%s', got '%s'", example.out, got)
			}
		})
	}
}

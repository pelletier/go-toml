package toml_test

import (
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalSimple(t *testing.T) {
	x := struct{ Foo string }{}
	err := toml.Unmarshal([]byte(`Foo = "hello"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.Foo)
}

func TestUnmarshalInt(t *testing.T) {
	x := struct{ Foo int }{}
	err := toml.Unmarshal([]byte(`Foo = 42`), &x)
	require.NoError(t, err)
	assert.Equal(t, 42, x.Foo)
}

func TestUnmarshalNestedStructs(t *testing.T) {
	x := struct{ Foo struct{ Bar string } }{}
	err := toml.Unmarshal([]byte(`Foo.Bar = "hello"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.Foo.Bar)
}

func TestUnmarshalNestedStructsMultipleExpressions(t *testing.T) {
	x := struct {
		A struct{ B string }
		C string
	}{}
	err := toml.Unmarshal([]byte(`A.B = "hello"
C = "test"`), &x)
	require.NoError(t, err)
	assert.Equal(t, "hello", x.A.B)
	assert.Equal(t, "test", x.C)
}

func TestUnmarshalTable(t *testing.T) {
	x := struct {
		Foo struct {
			A string
			B string
			C string
		}
		Bar struct {
			D string
		}
		E string
	}{}
	err := toml.Unmarshal([]byte(`

E = "E"
Foo.C = "C"

[Foo]
A = "A"
B = 'B'

[Bar]
D = "D"

`), &x)
	require.NoError(t, err)
	assert.Equal(t, "A", x.Foo.A)
	assert.Equal(t, "B", x.Foo.B)
	assert.Equal(t, "C", x.Foo.C)
	assert.Equal(t, "D", x.Bar.D)
	assert.Equal(t, "E", x.E)
}

func TestUnmarshalDoesNotEraseBaseStruct(t *testing.T) {
	x := struct {
		A string
		B string
	}{
		A: "preset",
	}
	err := toml.Unmarshal([]byte(`B = "data"`), &x)

	require.NoError(t, err)
	assert.Equal(t, "preset", x.A)
	assert.Equal(t, "data", x.B)
}

func TestArrayTableSimple(t *testing.T) {
	doc := `
[[Products]]
Name = "Hammer"

[[Products]]
Name = "Nail"
`

	type Product struct {
		Name string
	}

	type Data struct {
		Products []Product
	}

	x := Data{}
	err := toml.Unmarshal([]byte(doc), &x)

	require.NoError(t, err)

	expected := Data{
		Products: []Product{
			{
				Name: "Hammer",
			},
			{
				Name: "Nail",
			},
		},
	}

	assert.Equal(t, expected, x)
}

func TestUnmarshalArrayTablesMultiple(t *testing.T) {
	doc := `
[[Products]]
Name = "Hammer"
Sku = "738594937"

[[Products]]  # empty table within the array

[[Products]]
Name = "Nail"
Sku = "284758393"

Color = "gray"
`

	type Product struct {
		Name  string
		Sku   string
		Color string
	}

	type Data struct {
		Products []Product
	}

	x := Data{}
	err := toml.Unmarshal([]byte(doc), &x)

	require.NoError(t, err)

	expected := Data{
		Products: []Product{
			{
				Name: "Hammer",
				Sku:  "738594937",
			},
			{},
			{
				Name:  "Nail",
				Sku:   "284758393",
				Color: "gray",
			},
		},
	}

	assert.Equal(t, expected, x)
}

func TestUnmarshalArrayTablesNested(t *testing.T) {
	doc := `
[[Fruits]]
Name = "apple"

[Fruits.Physical]  # subtable
Color = "red"
Shape = "round"

[[Fruits.Varieties]]  # nested array of tables
Name = "red delicious"

[[Fruits.Varieties]]
Name = "granny smith"


[[Fruits]]
Name = "banana"

[[Fruits.Varieties]]
Name = "plantain"
`
	type Variety struct {
		Name string
	}

	type Physical struct {
		Color string
		Shape string
	}

	type Fruit struct {
		Name      string
		Physical  Physical
		Varieties []Variety
	}

	type Doc struct {
		Fruits []Fruit
	}

	x := Doc{}
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)

	expected := Doc{
		Fruits: []Fruit{
			{
				Name: "apple",
				Physical: Physical{
					Color: "red",
					Shape: "round",
				},
				Varieties: []Variety{
					{Name: "red delicious"},
					{Name: "granny smith"},
				},
			},
			{
				Name: "banana",
				Varieties: []Variety{
					{Name: "plantain"},
				},
			},
		},
	}

	assert.Equal(t, expected, x)
}

func TestUnmarshalArraySimple(t *testing.T) {
	x := struct {
		Foo []string
	}{}
	doc := `Foo = ["hello", "world"]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, x.Foo)
}

func TestUnmarshalArrayNestedInTable(t *testing.T) {
	x := struct {
		Wrapper struct {
			Foo []string
		}
	}{}
	doc := `[Wrapper]
Foo = ["hello", "world"]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, x.Wrapper.Foo)
}

func TestUnmarshalArrayMixed(t *testing.T) {
	x := struct {
		Wrapper struct {
			Foo []interface{}
		}
	}{}
	doc := `[Wrapper]
Foo = ["hello", true]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"hello", true}, x.Wrapper.Foo)
}

func TestUnmarshalArrayNested(t *testing.T) {
	x := struct {
		Foo [][]string
	}{}
	doc := `Foo = [["hello", "world"], ["a"], []]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"hello", "world"}, {"a"}, nil}, x.Foo)
}

func TestUnmarshalBool(t *testing.T) {
	x := struct {
		Truthy bool
		Falsey bool
	}{Falsey: true}
	doc := `Truthy = true
Falsey = false`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, true, x.Truthy)
	assert.Equal(t, false, x.Falsey)
}

func TestUnmarshalBoolArray(t *testing.T) {
	x := struct{ Bits []bool }{}
	doc := `Bits = [true, false, true, true]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true, true}, x.Bits)
}

func TestUnmarshalInlineTable(t *testing.T) {
	doc := `
	Name = { First = "Tom", Last = "Preston-Werner" }
	Point = { X = "1", Y = "2" }
	Animal = { Type.Name = "pug" }`

	type Name struct {
		First string
		Last  string
	}

	type Point struct {
		X string
		Y string
	}

	type Type struct {
		Name string
	}

	type Animal struct {
		Type Type
	}

	type Doc struct {
		Name   Name
		Point  Point
		Animal Animal
	}
	x := Doc{}
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)

	expected := Doc{
		Name: Name{
			First: "Tom",
			Last:  "Preston-Werner",
		},
		Point: Point{
			X: "1",
			Y: "2",
		},
		Animal: Animal{
			Type: Type{
				Name: "pug",
			},
		},
	}
	assert.Equal(t, expected, x)
}

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

func TestArraySimple(t *testing.T) {
	x := struct {
		Foo []string
	}{}
	doc := `Foo = ["hello", "world"]`
	err := toml.Unmarshal([]byte(doc), &x)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, x.Foo)
}

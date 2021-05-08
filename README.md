# go-toml v2

Go library for the [TOML](https://toml.io/en/) format.

This library supports [TOML v1.0.0](https://toml.io/en/v1.0.0).


## Development status

This is the upcoming major version of go-toml. It is currently in active
development. As of release v2.0.0-beta.1, the library has reached feature parity
with v1, and fixes a lot known bugs and performance issues along the way.

If you do not need the advanced document editing features of v1, you are
encouraged to try out this version.

👉 [Roadmap for v2](https://github.com/pelletier/go-toml/discussions/506).


## Documentation

Full API, examples, and implementation notes are available in the Go documentation.

[![Go Reference](https://pkg.go.dev/badge/github.com/pelletier/go-toml/v2.svg)](https://pkg.go.dev/github.com/pelletier/go-toml/v2)


## Import

```go
import "github.com/pelletier/go-toml/v2"
```

## Features

### Stdlib behavior

As much as possible, this library is designed to behave similarly as the
standard library's `encoding/json`.

### Performance

While go-toml favors usability, it is written with performance in mind. Most
operations should not be shockingly slow.

### Strict mode

`Decoder` can be set to "strict mode", which makes it error when some parts of
the TOML document was not prevent in the target structure. This is a great way
to check for typos. [See example in the documentation][strict].

[strict]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#example-Decoder.SetStrict

### Contextualized errors

When decoding errors occur, go-toml returns [`DecodeError`][decode-err]), which
contains a human readable contextualized version of the error. For example:

```
2| key1 = "value1"
3| key2 = "missing2"
 | ~~~~ missing field
4| key3 = "missing3"
5| key4 = "value4"
```

[decode-err]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#DecodeError

### Local date and time support

TOML supports native [local date/times][ldt]. It allows to represent a given
date, time, or date-time without relation to a timezone or offset. To support
this use-case, go-toml provides [`LocalDate`][tld], [`LocalTime`][tlt], and
[`LocalDateTime`][tldt]. Those types can be transformed to and from `time.Time`,
making them convenient yet unambiguous structures for their respective TOML
representation.

[ldt]: https://toml.io/en/v1.0.0#local-date-time
[tld]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#LocalDate
[tlt]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#LocalTime
[tldt]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#LocalDateTime

## Getting started

Given the following struct, let's see how to read it and write it as TOML:

```go
type MyConfig struct {
      Version int
      Name    string
      Tags    []string
}
```

### Unmarshaling

[`Unmarshal`][unmarshal] reads a TOML document and fills a Go structure with its
content. For example:

```go
doc := `
version = 2
name = "go-toml"
tags = ["go", "toml"]
`

var cfg MyConfig
err := toml.Unmarshal([]byte(doc), &cfg)
if err != nil {
      panic(err)
}
fmt.Println("version:", cfg.Version)
fmt.Println("name:", cfg.Name)
fmt.Println("tags:", cfg.Tags)

// Output:
// version: 2
// name: go-toml
// tags: [go toml]
```

[unmarshal]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#Unmarshal

### Marshaling

[`Marshal`][marshal] is the opposite of Unmarshal: it represents a Go structure
as a TOML document:

```go
cfg := MyConfig{
      Version: 2,
      Name:    "go-toml",
      Tags:    []string{"go", "toml"},
}

b, err := toml.Marshal(cfg)
if err != nil {
      panic(err)
}
fmt.Println(string(b))

// Output:
// Version = 2
// Name = 'go-toml'
// Tags = ['go', 'toml']
```

[marshal]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#Marshal

## Migrating from v1

This section describes the differences between v1 and v2, with some pointers on
how to get the original behavior when possible.

### Decoding / Unmarshal

#### Automatic field name guessing

When unmarshaling to a struct, if a key in the TOML document does not exactly
match the name of a struct field or any of the `toml`-tagged field, v1 tries
multiple variations of the key ([code][v1-keys]).

V2 instead does a case-insensitive matching, like `encoding/json`.

This could impact you if you are relying on casing to differentiate two fields,
and one of them is a not using the `toml` struct tag. The recommended solution
is to be specific about tag names for those fields using the `toml` struct tag.

[v1-keys]: https://github.com/pelletier/go-toml/blob/a2e52561804c6cd9392ebf0048ca64fe4af67a43/marshal.go#L775-L781

#### Ignore preexisting value in interface

When decoding into a non-nil `interface{}`, go-toml v1 uses the type of the
element in the interface to decode the object. For example:

```go
type inner struct {
  B interface{}
}
type doc struct {
  A interface{}
}

d := doc{
  A: inner{
    B: "Before",
  },
}

data := `
[A]                                                                                                                           
B = "After"
`

toml.Unmarshal([]byte(data), &d)
fmt.Printf("toml v1: %#v\n", d)

// toml v1: main.doc{A:main.inner{B:"After"}}
```

In this case, field `A` is of type `interface{}`, containing a `inner` struct.
V1 sees that type and uses it when decoding the object.

When decoding an object into an `interface{}`, V2 instead disregards whatever
value the `interface{}` may contain and replaces it with a
`map[string]interface{}`. With the same data structure as above, here is what
the result looks like:

```go
toml.Unmarshal([]byte(data), &d)
fmt.Printf("toml v2: %#v\n", d)

// toml v2: main.doc{A:map[string]interface {}{"B":"After"}}
```

This is to match `encoding/json`'s behavior. There is no way to make the v2
decoder behave like v1.

#### Values out of array bounds ignored

When decoding into an array, v1 returns an error when the number of elements
contained in the doc is superior to the capacity of the array. For example:

```go
type doc struct {
  A [2]string
}
d := doc{}
err := toml.Unmarshal([]byte(`A = ["one", "two", "many"]`), &d)
fmt.Println(err)

// (1, 1): unmarshal: TOML array length (3) exceeds destination array length (2)
```

In the same situation, v2 ignores the last value:

```go
err := toml.Unmarshal([]byte(`A = ["one", "two", "many"]`), &d)
fmt.Println("err:", err, "d:", d)
// err: <nil> d: {[one two]}
```

This is to match `encoding/json`'s behavior. There is no way to make the v2
decoder behave like v1.

#### Support for `toml.Unmarshaler` has been dropped

This method was not widely used, poorly defined, and added a lot of complexity.
A similar effect can be achieved by implementing the `encoding.TextUnmarshaler`
interface and use strings.

### Encoding / Marshal

#### Default struct fields order

V1 emits struct fields order alphabetically by default. V2 struct fields are
emitted in order they are defined. For example:

```go
type S struct {
	B string
	A string
}

data := S{
	B: "B",
	A: "A",
}

b, _ := tomlv1.Marshal(data)
fmt.Println("v1:\n" + string(b))

b, _ = tomlv2.Marshal(data)
fmt.Println("v2:\n" + string(b))

// Output:
// v1:
// A = "A"
// B = "B"

// v2:
// B = 'B'
// A = 'A'
```

There is no way to make v2 encoder behave like v1. A workaround could be to
manually sort the fields alphabetically in the struct definition.

#### No indentation by default

V1 automatically indents content of tables by default. V2 does not. However the
same behavior can be obtained using [`Encoder.SetIndentTables`][sit]. For example:


```go
data := map[string]interface{}{
	"table": map[string]string{
		"key": "value",
	},
}

b, _ := tomlv1.Marshal(data)
fmt.Println("v1:\n" + string(b))

b, _ = tomlv2.Marshal(data)
fmt.Println("v2:\n" + string(b))

buf := bytes.Buffer{}
enc := tomlv2.NewEncoder(&buf)
enc.SetIndentTables(true)
enc.Encode(data)
fmt.Println("v2 Encoder:\n" + string(buf.Bytes()))

// Output:
// v1:
// 
// [table]
//   key = "value"
// 
// v2:
// [table]
// key = 'value'
// 
// 
// v2 Encoder:
// [table]
//   key = 'value'
```

[sit]: https://pkg.go.dev/github.com/pelletier/go-toml/v2#Encoder.SetIndentTables

#### Keys and strings are single quoted

V1 always uses double quotes (`"`) around strings and keys that cannot be
represented bare (unquoted). V2 uses single quotes instead by default (`'`),
unless a character cannot be represented, then falls back to double quotes.

There is no way to make v2 encoder behave like v1.

#### `TextMarshaler` emits as a string, not TOML

Types that implement [`encoding.TextMarshaler`][tm] can emit arbitrary TOML in
v1. The encoder would append the result to the output directly. In v2 the result
is wrapped in a string. As a result, this interface cannot be implemented by the
root object.

There is no way to make v2 encoder behave like v1.

[tm]: https://golang.org/pkg/encoding/#TextMarshaler

## License

The MIT License (MIT). Read [LICENSE](LICENSE).

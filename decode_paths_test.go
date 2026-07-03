package toml

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2/internal/assert"
)

// TestUnmarshalManyKeysTable exercises the seen-tracker's spill to its hash
// index (tables with more keys than fit the sibling chains), for both the
// fused generic path and the reflection path, including duplicate detection
// after the spill.
func TestUnmarshalManyKeysTable(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, "key%d = %d\n", i, i)
	}
	doc := sb.String()

	t.Run("map", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(doc), &m))
		assert.Equal(t, 100, len(m))
		assert.Equal(t, interface{}(int64(99)), m["key99"])
	})

	t.Run("struct", func(t *testing.T) {
		var s struct {
			Key0  int64
			Key42 int64
			Key99 int64
		}
		assert.NoError(t, Unmarshal([]byte(doc), &s))
		assert.Equal(t, int64(42), s.Key42)
		assert.Equal(t, int64(99), s.Key99)
	})

	t.Run("duplicate after spill", func(t *testing.T) {
		m := map[string]interface{}{}
		err := Unmarshal([]byte(doc+"key12 = 1\n"), &m)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "already"), "unexpected error: %v", err)
	})

	t.Run("array table refresh under root", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 40; i++ {
			fmt.Fprintf(&sb, "key%d = %d\n", i, i)
		}
		for i := 0; i < 3; i++ {
			sb.WriteString("[[elem]]\nname = 'x'\n")
		}
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(sb.String()), &m))
		assert.Equal(t, 3, len(m["elem"].([]interface{})))
	})
}

// TestUnmarshalGenericValueShapes covers the generic decoding of every scalar
// kind (through interface{} struct fields, which use the AST path), long
// strings beyond the slab limit, and arrays larger than the slab cutoff.
func TestUnmarshalGenericValueShapes(t *testing.T) {
	long := strings.Repeat("x", 600)
	var arr strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&arr, "%d,", i)
	}
	doc := fmt.Sprintf(`
str = "hello"
longstr = "%s"
int = 42
bigint = 123456
negint = -7
float = 1.25
bool = true
date = 2021-03-30
time = 11:21:00
datetime = 2021-03-30T11:21:00Z
localdt = 2021-03-30T11:21:00
bigarray = [%s]
nested = { a.b = 1, c = [{ d = 2 }, { d = 3 }] }
`, long, arr.String())

	type target struct {
		Str      interface{}
		Longstr  interface{}
		Int      interface{}
		Bigint   interface{}
		Negint   interface{}
		Float    interface{}
		Bool     interface{}
		Date     interface{}
		Time     interface{}
		Datetime interface{}
		Localdt  interface{}
		Bigarray interface{}
		Nested   interface{}
	}

	check := func(t *testing.T, get func(string) interface{}) {
		t.Helper()
		assert.Equal(t, interface{}("hello"), get("str"))
		assert.Equal(t, interface{}(long), get("longstr"))
		assert.Equal(t, interface{}(int64(42)), get("int"))
		assert.Equal(t, interface{}(int64(123456)), get("bigint"))
		assert.Equal(t, interface{}(int64(-7)), get("negint"))
		assert.Equal(t, interface{}(1.25), get("float"))
		assert.Equal(t, interface{}(true), get("bool"))
		assert.Equal(t, interface{}(LocalDate{2021, 3, 30}), get("date"))
		assert.Equal(t, 40, len(get("bigarray").([]interface{})))
		nested := get("nested").(map[string]interface{})
		assert.Equal(t, interface{}(int64(1)), nested["a"].(map[string]interface{})["b"])
		cs := nested["c"].([]interface{})
		assert.Equal(t, interface{}(int64(3)), cs[1].(map[string]interface{})["d"])
		dt := get("datetime").(time.Time)
		assert.Equal(t, 2021, dt.Year())
	}

	t.Run("map target (fused path)", func(t *testing.T) {
		m := map[string]interface{}{}
		assert.NoError(t, Unmarshal([]byte(doc), &m))
		check(t, func(k string) interface{} { return m[k] })
	})

	t.Run("struct with interface fields (AST path)", func(t *testing.T) {
		var s target
		assert.NoError(t, Unmarshal([]byte(doc), &s))
		byName := map[string]interface{}{
			"str": s.Str, "longstr": s.Longstr, "int": s.Int,
			"bigint": s.Bigint, "negint": s.Negint, "float": s.Float,
			"bool": s.Bool, "date": s.Date, "time": s.Time,
			"datetime": s.Datetime, "localdt": s.Localdt,
			"bigarray": s.Bigarray, "nested": s.Nested,
		}
		check(t, func(k string) interface{} { return byName[k] })
	})
}

// TestUnmarshalFusedLineEndings covers CRLF and comment handling inside the
// natively decoded containers, and bare-CR rejection.
func TestUnmarshalFusedLineEndings(t *testing.T) {
	m := map[string]interface{}{}
	doc := "a = [ # comment\r\n1, # c\r\n2 ]\r\nb = { # comment\r\nx = 1, # c\r\n}\r\n"
	assert.NoError(t, Unmarshal([]byte(doc), &m))
	assert.Equal(t, 2, len(m["a"].([]interface{})))
	assert.Equal(t, interface{}(int64(1)), m["b"].(map[string]interface{})["x"])

	for _, bad := range []string{"a = [1,\r2]", "a = {x = 1,\ry = 2}"} {
		assert.Error(t, Unmarshal([]byte(bad), &map[string]interface{}{}))
	}
}

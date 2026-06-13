package toml_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/internal/assert"
)

type failKey struct{}

func (failKey) MarshalText() ([]byte, error) {
	return nil, errors.New("nope")
}

type ptrTextKey struct {
	V string
}

func (k *ptrTextKey) MarshalText() ([]byte, error) {
	return []byte(k.V), nil
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("disk full")
}

func TestMarshalRootErrors(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		_, err := toml.Marshal(nil)
		assert.Error(t, err)
	})
	t.Run("nil pointer", func(t *testing.T) {
		_, err := toml.Marshal((*int)(nil))
		assert.Error(t, err)
	})
	t.Run("scalar root", func(t *testing.T) {
		_, err := toml.Marshal(42)
		assert.Error(t, err)
	})
	t.Run("value-kind struct root", func(t *testing.T) {
		_, err := toml.Marshal(time.Time{})
		assert.Error(t, err)
	})
	t.Run("write error", func(t *testing.T) {
		err := toml.NewEncoder(failWriter{}).Encode(map[string]int{"a": 1})
		assert.Error(t, err)
	})
}

func TestMarshalNilTableValues(t *testing.T) {
	type sub struct {
		X int
	}
	t.Run("nil struct pointer map value", func(t *testing.T) {
		out, err := toml.Marshal(map[string]*sub{"a": nil})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "[a]"))
	})
	t.Run("nil scalar pointer map value", func(t *testing.T) {
		out, err := toml.Marshal(map[string]*int{"a": nil})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "a = 0"))
	})
	t.Run("nil pointer array element encodes zero value", func(t *testing.T) {
		out, err := toml.Marshal(map[string][]*sub{"a": {nil}})
		assert.NoError(t, err)
		assert.Equal(t, "a = [{X = 0}]\n", string(out))
	})
}

func TestMarshalMapKeys(t *testing.T) {
	t.Run("nil interface key", func(t *testing.T) {
		_, err := toml.Marshal(map[interface{}]int{nil: 1})
		assert.Error(t, err)
	})
	t.Run("text marshaler key", func(t *testing.T) {
		out, err := toml.Marshal(map[time.Time]int{{}: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "0001-01-01"))
	})
	t.Run("pointer receiver text marshaler key", func(t *testing.T) {
		out, err := toml.Marshal(map[ptrTextKey]int{{V: "k"}: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "k = 1"))
	})
	t.Run("failing text marshaler key", func(t *testing.T) {
		_, err := toml.Marshal(map[failKey]int{{}: 1})
		assert.Error(t, err)
	})
	t.Run("integer keys", func(t *testing.T) {
		out, err := toml.Marshal(map[int64]int{-1: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "-1 = 1"))
		out, err = toml.Marshal(map[uint64]int{7: 1})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "7 = 1"))
	})
}

type selfEmbed struct {
	*selfEmbed //nolint:unused // exercises recursive embedded type plans
	X          int
}

type planInner struct {
	X int
	Y int
}

type planOuter struct {
	planInner
	X int
}

type tagEmbed struct {
	planInner `toml:",inline"`
}

func TestMarshalEncodePlans(t *testing.T) {
	t.Run("recursive embedded type", func(t *testing.T) {
		v := selfEmbed{X: 1}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 1"))
	})
	t.Run("shadowed embedded field", func(t *testing.T) {
		v := planOuter{planInner: planInner{X: 5, Y: 2}}
		v.X = 9
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 9"))
		assert.True(t, strings.Contains(string(out), "Y = 2"))
		assert.False(t, strings.Contains(string(out), "X = 5"))
	})
	t.Run("tagged embedded without name", func(t *testing.T) {
		v := tagEmbed{planInner{X: 3, Y: 4}}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 3"))
	})
	t.Run("nil embedded pointer", func(t *testing.T) {
		v := selfEmbed{X: 2}
		out, err := toml.Marshal(&v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "X = 2"))
	})
}

type myInt int

type embedNonStruct struct {
	myInt
	A int
}

func TestMarshalEmbeddedNonStructUnexported(t *testing.T) {
	out, err := toml.Marshal(embedNonStruct{myInt: 1, A: 2})
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(out), "myInt"))
	assert.True(t, strings.Contains(string(out), "A = 2"))
}

func TestMarshalOmitemptyUncommonKind(t *testing.T) {
	v := struct {
		F func() `toml:",omitempty"`
	}{F: func() {}}
	_, err := toml.Marshal(v)
	assert.Error(t, err)
}

type ptrText struct {
	V string
}

func (p *ptrText) MarshalText() ([]byte, error) {
	return []byte(p.V), nil
}

func TestMarshalPtrReceiverTextMarshalerMapValue(t *testing.T) {
	out, err := toml.Marshal(map[string]ptrText{"a": {V: "x"}})
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), `a = 'x'`))
}

func TestMarshalJSONNumbers(t *testing.T) {
	enc := func(v interface{}) (string, error) {
		var buf strings.Builder
		err := toml.NewEncoder(&buf).SetMarshalJSONNumbers(true).Encode(v)
		return buf.String(), err
	}
	t.Run("invalid", func(t *testing.T) {
		_, err := enc(map[string]json.Number{"a": json.Number("abc")})
		assert.Error(t, err)
	})
	t.Run("empty is zero", func(t *testing.T) {
		out, err := enc(map[string]json.Number{"a": json.Number("")})
		assert.NoError(t, err)
		assert.Equal(t, "a = 0\n", out)
	})
	t.Run("float", func(t *testing.T) {
		out, err := enc(map[string]json.Number{"a": json.Number("1.5")})
		assert.NoError(t, err)
		assert.Equal(t, "a = 1.5\n", out)
	})
}

func TestMarshalElementErrors(t *testing.T) {
	t.Run("array element error", func(t *testing.T) {
		_, err := toml.Marshal(map[string][]interface{}{"a": {func() {}}})
		assert.Error(t, err)
	})
	t.Run("multiline array element error", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		err := enc.Encode(map[string][]interface{}{"a": {func() {}}})
		assert.Error(t, err)
	})
	t.Run("inline table element error", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetTablesInline(true)
		err := enc.Encode(map[string]map[string]interface{}{"a": {"b": func() {}}})
		assert.Error(t, err)
	})
	t.Run("multiline array of arrays", func(t *testing.T) {
		var buf strings.Builder
		enc := toml.NewEncoder(&buf)
		enc.SetArraysMultiline(true)
		err := enc.Encode(map[string][]interface{}{"a": {[]interface{}{int64(1), int64(2)}, "x"}})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(buf.String(), "1"))
	})
}

func TestMarshalStringEscapes(t *testing.T) {
	t.Run("invalid utf8 in basic string", func(t *testing.T) {
		out, err := toml.Marshal(map[string]string{"a": "\xff"})
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), "\\u00FF"))
	})
	t.Run("multiline escapes", func(t *testing.T) {
		v := struct {
			S string `toml:",multiline"`
		}{S: "a\"\"\"\"b\\c\bd\fe\rf\tg\x01h\xffi\nj"}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		s := string(out)
		assert.True(t, strings.Contains(s, "\\\"\\\"\\\"\\\""))
		assert.True(t, strings.Contains(s, "\\\\"))
		assert.True(t, strings.Contains(s, "\\b"))
		assert.True(t, strings.Contains(s, "\\f"))
		assert.True(t, strings.Contains(s, "\\r"))
		assert.True(t, strings.Contains(s, "\\u0001"))
		assert.True(t, strings.Contains(s, "\\u00FF"))
	})
	t.Run("multiline short quote runs kept", func(t *testing.T) {
		v := struct {
			S string `toml:",multiline"`
		}{S: `a""b`}
		out, err := toml.Marshal(v)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(out), `a""b`))
	})
}

type failPtrKey struct{ V int }

func (*failPtrKey) MarshalText() ([]byte, error) {
	return nil, errors.New("nope")
}

func TestMarshalMapKeysMore(t *testing.T) {
	t.Run("failing pointer receiver text marshaler key", func(t *testing.T) {
		_, err := toml.Marshal(map[failPtrKey]int{{V: 1}: 1})
		assert.Error(t, err)
	})
}

type commentedTagged struct {
	A int `commented:"true"`
}

func TestMarshalStandaloneCommentedTag(t *testing.T) {
	out, err := toml.Marshal(commentedTagged{A: 1})
	assert.NoError(t, err)
	assert.Equal(t, "# A = 1\n", string(out))
}

type nilEmbedInner struct {
	Z int
}

type nilEmbedOuter struct {
	*nilEmbedInner
	X int
}

func TestMarshalNilEmbeddedPointer(t *testing.T) {
	out, err := toml.Marshal(nilEmbedOuter{X: 1})
	assert.NoError(t, err)
	assert.Equal(t, "X = 1\n", string(out))
}

func TestMarshalInterfaceHoldingNilPointers(t *testing.T) {
	type sub struct{ X int }
	t.Run("nil struct pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*sub)(nil)})
		assert.NoError(t, err)
		// Interface-held nil pointers encode as the zero value of their
		// element type, like nil pointer map values.
		assert.Equal(t, "a = {X = 0}\n", string(out))
	})
	t.Run("nil map pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*map[string]int)(nil)})
		assert.NoError(t, err)
		assert.Equal(t, "a = {}\n", string(out))
	})
	t.Run("nil slice pointer in interface", func(t *testing.T) {
		out, err := toml.Marshal(map[string]interface{}{"a": (*[]int)(nil)})
		assert.NoError(t, err)
		assert.Equal(t, "a = []\n", string(out))
	})
}

func TestMarshalNestedTableValueError(t *testing.T) {
	_, err := toml.Marshal(map[string]map[string]interface{}{"a": {"b": func() {}}})
	assert.Error(t, err)
}

func TestMarshalInlineTableKeyError(t *testing.T) {
	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.SetTablesInline(true)
	err := enc.Encode(map[string]map[failKey]int{"a": {{}: 1}})
	assert.Error(t, err)
}

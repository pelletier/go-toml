package toml

import (
	"reflect"

	"github.com/pelletier/go-toml/v2/internal/reflectbuild"
)

func Unmarshal(data []byte, v interface{}) error {
	u := &unmarshaler{}
	u.builder, u.err = reflectbuild.NewBuilder(v)
	if u.err == nil {
		parseErr := parser{builder: u}.parse(data)
		if parseErr != nil {
			return parseErr
		}
	}
	return u.err
}

type unmarshaler struct {
	builder reflectbuild.Builder

	// First error that appeared during the construction of the object.
	// When set all callbacks are no-ops.
	err error

	// State that indicates the parser is processing a [[table-array]] name.
	// If false keys are interpreted as part of a key-value or [table].
	parsingTableArray bool

	// Table Arrays need a buffer of keys because we need to know which one is
	// the last one, as it may result in creating a new element in the array.
	arrayTableKey [][]byte

	// Flag to indicate that the next value is an an assignment.
	// Assignments are when the builder already points to the value, and should
	// be directly assigned. This is used to distinguish between assigning or
	// appending to arrays.
	assign bool
}

func (u *unmarshaler) Assignation() {
	u.assign = true
}

func (u *unmarshaler) ArrayBegin() {
	if u.err != nil {
		return
	}
	u.builder.Save()
	if u.assign {
		u.assign = false
	} else {
		u.err = u.builder.SliceNewElem()
	}
}

func (u *unmarshaler) ArrayEnd() {
	if u.err != nil {
		return
	}
	u.builder.Load()
}

func (u *unmarshaler) ArrayTableBegin() {
	if u.err != nil {
		return
	}

	u.parsingTableArray = true
}

func (u *unmarshaler) ArrayTableEnd() {
	if u.err != nil {
		return
	}

	u.builder.Reset()

	for _, v := range u.arrayTableKey[:len(u.arrayTableKey)-1] {
		u.err = u.builder.DigField(string(v))
		if u.err != nil {
			return
		}
		u.err = u.builder.SliceLastOrCreate()
	}

	v := u.arrayTableKey[len(u.arrayTableKey)-1]
	u.err = u.builder.DigField(string(v))
	if u.err != nil {
		return
	}
	u.err = u.builder.SliceNewElem()

	u.parsingTableArray = false
	u.arrayTableKey = u.arrayTableKey[:0]
}

func (u *unmarshaler) KeyValBegin() {
	u.builder.Save()
}

func (u *unmarshaler) KeyValEnd() {
	u.builder.Load()
}

func (u *unmarshaler) StringValue(v []byte) {
	if u.err != nil {
		return
	}
	if u.builder.IsSlice() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(string(v)))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.SetString(string(v))
	}
}

func (u *unmarshaler) BoolValue(b bool) {
	if u.err != nil {
		return
	}
	if u.builder.IsSlice() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(b))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.SetBool(b)
	}
}

func (u *unmarshaler) FloatValue(n float64) {
	if u.err != nil {
		return
	}
	if u.builder.IsSlice() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(n))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.SetFloat(n)
	}
}

func (u *unmarshaler) IntValue(n int64) {
	if u.err != nil {
		return
	}
	if u.builder.IsSlice() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(n))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.SetInt(n)
	}
}

func (u *unmarshaler) SimpleKey(v []byte) {
	if u.err != nil {
		return
	}

	if u.parsingTableArray {
		u.arrayTableKey = append(u.arrayTableKey, v)
	} else {
		if u.builder.Cursor().Kind() == reflect.Slice {
			u.err = u.builder.SliceLastOrCreate()
			if u.err != nil {
				return
			}
		}
		u.err = u.builder.DigField(string(v))
	}
}

func (u *unmarshaler) StandardTableBegin() {
	if u.err != nil {
		return
	}

	// tables are only top-level
	u.builder.Reset()
}

func (u *unmarshaler) StandardTableEnd() {
	if u.err != nil {
		return
	}
}

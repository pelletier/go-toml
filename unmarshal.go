package toml

import (
	"reflect"
	"time"

	"github.com/pelletier/go-toml/v2/internal/reflectbuild"
)

func Unmarshal(data []byte, v interface{}) error {
	u := &unmarshaler{}
	u.builder, u.err = reflectbuild.NewBuilder("toml", v)
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

	// State that indicates the parser is processing a [table] name.
	// Used to know whether the whole table should be skipped or just the
	// keyval if a field is missing.
	parsingTable bool

	// Counters that indicate that we are skipping TOML expressions. It happens
	// when the document contains values that are not in the target struct.
	// TODO: signal the parser that it can just scan to avoid processing the
	// unused data.
	skipKeyValCount uint
	skipTable       bool
}

func (u *unmarshaler) skipping() bool {
	return u.skipTable || u.skipKeyValCount > 0
}

func (u *unmarshaler) Assignation() {
	if u.skipping() || u.err != nil {
		return
	}
	u.assign = true
}

func (u *unmarshaler) ArrayBegin() {
	if u.skipping() || u.err != nil {
		return
	}
	u.builder.Save()
	u.err = u.builder.EnsureSlice()
	if u.err != nil {
		return
	}
	if u.assign {
		u.assign = false
	} else {
		u.err = u.builder.SliceNewElem()
	}
}

func (u *unmarshaler) ArrayEnd() {
	if u.skipping() || u.err != nil {
		return
	}
	u.builder.Load()
}

func (u *unmarshaler) ArrayTableBegin() {
	if u.skipping() || u.err != nil {
		return
	}

	u.parsingTableArray = true
}

func (u *unmarshaler) ArrayTableEnd() {
	if u.skipping() || u.err != nil {
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

func (u *unmarshaler) InlineTableBegin() {
	if u.skipping() || u.err != nil {
		return
	}

	// TODO

}

func (u *unmarshaler) InlineTableEnd() {
	if u.skipping() || u.err != nil {
		return
	}

	// TODO
}

func (u *unmarshaler) KeyValBegin() {
	if u.skipKeyValCount > 0 {
		u.skipKeyValCount++
		return
	}
	if u.skipping() || u.err != nil {
		return
	}
	u.builder.Save()
}

func (u *unmarshaler) KeyValEnd() {
	if u.skipKeyValCount > 0 {
		u.skipKeyValCount--
		return
	}
	if u.skipping() || u.err != nil {
		return
	}
	u.builder.Load()
}

func (u *unmarshaler) StringValue(v []byte) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		s := string(v)
		u.err = u.builder.SliceAppend(reflect.ValueOf(&s))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		s := string(v)
		u.err = u.builder.Set(reflect.ValueOf(&s))
	}
}

func (u *unmarshaler) BoolValue(b bool) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&b))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.SetBool(b)
	}
}

func (u *unmarshaler) FloatValue(n float64) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&n))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&n))
		//u.err = u.builder.SetFloat(n)
	}
}

func (u *unmarshaler) IntValue(n int64) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&n))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&n))
	}
}

func (u *unmarshaler) LocalDateValue(date LocalDate) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&date))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&date))
	}
}

func (u *unmarshaler) LocalDateTimeValue(dt LocalDateTime) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&dt))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&dt))
	}
}

func (u *unmarshaler) DateTimeValue(dt time.Time) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&dt))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&dt))
	}
}

func (u *unmarshaler) LocalTimeValue(localTime LocalTime) {
	if u.skipping() || u.err != nil {
		return
	}
	if u.builder.IsSliceOrPtr() {
		u.builder.Save()
		u.err = u.builder.SliceAppend(reflect.ValueOf(&localTime))
		if u.err != nil {
			return
		}
		u.builder.Load()
	} else {
		u.err = u.builder.Set(reflect.ValueOf(&localTime))
	}
}

func (u *unmarshaler) SimpleKey(v []byte) {
	if u.skipping() || u.err != nil {
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
		if u.err == nil {
			return
		}
		if _, ok := u.err.(reflectbuild.FieldNotFoundError); ok {
			u.err = nil
			if u.parsingTable {
				u.skipTable = true
			} else {
				u.skipKeyValCount = 1
			}
		}
		// TODO: figure out what to do with unexported fields
	}
}

func (u *unmarshaler) StandardTableBegin() {
	u.skipTable = false
	u.parsingTable = true
	if u.skipping() || u.err != nil {
		return
	}
	// tables are only top-level
	u.builder.Reset()
}

func (u *unmarshaler) StandardTableEnd() {
	u.parsingTable = false
	if u.skipping() || u.err != nil {
		return
	}

	u.builder.EnsureStructOrMap()
}

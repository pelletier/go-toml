package toml

import (
	"encoding"
	"reflect"
	"time"
)

// isZeroer is used to check whether a value is the zero value for its type,
// as defined by the type itself.
type isZeroer interface {
	IsZero() bool
}

var isZeroerType = reflect.TypeOf(new(isZeroer)).Elem()

var timeType = reflect.TypeOf(time.Time{})
var textMarshalerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
var textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()
var mapStringInterfaceType = reflect.TypeOf(map[string]interface{}(nil))
var sliceInterfaceType = reflect.TypeOf([]interface{}(nil))
var stringType = reflect.TypeOf("")

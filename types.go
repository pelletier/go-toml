package toml

import (
	"encoding"
	"reflect"
	"time"
)

var (
	timeType               = reflect.TypeOf((*time.Time)(nil)).Elem()
	textMarshalerType      = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	textUnmarshalerType    = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	mapStringInterfaceType = reflect.TypeOf(map[string]interface{}(nil))
	sliceInterfaceType     = reflect.TypeOf([]interface{}(nil))
	stringType             = reflect.TypeOf("")
)

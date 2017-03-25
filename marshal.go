package toml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

/*
TomlTree structural types and corresponding marshal types
-------------------------------------------------------------------------------
*TomlTree                        (*)struct, (*)map[string]interface{}
[]*TomlTree                      (*)[](*)struct, (*)[](*)map[string]interface{}
[]interface{} (as interface{})   (*)[]primitive, (*)[]([]interface{})
interface{}                      (*)primitive

TomlTree primitive types and  corresponding marshal types
-----------------------------------------------------------
uint64     uint, uint8-uint64, pointers to same
int64      int, int8-uint64, pointers to same
float64    float32, float64, pointers to same
string     string, pointers to same
bool       bool, pointers to same
time.Time  time.Time{}, pointers to same
*/

var timeType = reflect.TypeOf(time.Time{})

// Check if the given marshall type maps to a TomlTree primitive
func isPrimitive(mtype reflect.Type) bool {
	switch mtype.Kind() {
	case reflect.Ptr:
		return isPrimitive(mtype.Elem())
	case reflect.Bool:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.String:
		return true
	case reflect.Struct:
		return mtype == timeType
	default:
		return false
	}
}

// Check if the given marshall type maps to a TomlTree slice
func isTreeSlice(mtype reflect.Type) bool {
	switch mtype.Kind() {
	case reflect.Ptr:
		return isTreeSlice(mtype.Elem())
	case reflect.Slice:
		return !isOtherSlice(mtype)
	default:
		return false
	}
}

// Check if the given marshall type maps to a non-TomlTree slice
func isOtherSlice(mtype reflect.Type) bool {
	switch mtype.Kind() {
	case reflect.Ptr:
		return isOtherSlice(mtype.Elem())
	case reflect.Slice:
		return isPrimitive(mtype.Elem()) || mtype.Elem().Kind() == reflect.Slice
	default:
		return false
	}
}

// Check if the given marshall type maps to a TomlTree
func isTree(mtype reflect.Type) bool {
	switch mtype.Kind() {
	case reflect.Ptr:
		return isTree(mtype.Elem())
	case reflect.Map:
		return true
	case reflect.Struct:
		return !isPrimitive(mtype)
	default:
		return false
	}
}

// Marshal ...
func Marshal(v interface{}) ([]byte, error) {
	mtype := reflect.TypeOf(v)
	if mtype.Kind() != reflect.Struct {
		return []byte{}, errors.New("Only a Struct can be marshaled to TOML")
	}
	sval := reflect.ValueOf(v)
	t, err := valueToTree(mtype, sval)
	if err != nil {
		return []byte{}, err
	}
	s, err := t.ToTomlString()
	return []byte(s), err
}

// Convert given marshal struct or map value to toml tree
func valueToTree(mtype reflect.Type, mval reflect.Value) (*TomlTree, error) {
	if mtype.Kind() == reflect.Ptr {
		return valueToTree(mtype.Elem(), mval.Elem())
	}
	tval := newTomlTree()
	switch mtype.Kind() {
	case reflect.Struct:
		for i := 0; i < mtype.NumField(); i++ {
			mtypef, mvalf := mtype.Field(i), mval.Field(i)
			if mtypef.PkgPath == "" {
				val, err := valueToToml(mtypef.Type, mvalf)
				if err != nil {
					return nil, err
				}
				tval.Set(tomlName(mtypef), val)
			}
		}
	case reflect.Map:
		for _, key := range mval.MapKeys() {
			mvalf := mval.MapIndex(key)
			val, err := valueToToml(mtype.Elem(), mvalf)
			if err != nil {
				return nil, err
			}
			tval.Set(key.String(), val)
		}
	}
	return tval, nil
}

// Convert given marshal slice to slice of Toml trees
func valueToTreeSlice(mtype reflect.Type, mval reflect.Value) ([]*TomlTree, error) {
	if mtype.Kind() == reflect.Ptr {
		return valueToTreeSlice(mtype.Elem(), mval.Elem())
	}
	tval := make([]*TomlTree, mval.Len(), mval.Len())
	for i := 0; i < mval.Len(); i++ {
		val, err := valueToTree(mtype.Elem(), mval.Index(i))
		if err != nil {
			return nil, err
		}
		tval[i] = val
	}
	return tval, nil
}

// Convert given marshal slice to slice of toml values
func valueToOtherSlice(mtype reflect.Type, mval reflect.Value) (interface{}, error) {
	if mtype.Kind() == reflect.Ptr {
		return valueToOtherSlice(mtype.Elem(), mval.Elem())
	}
	tval := make([]interface{}, mval.Len(), mval.Len())
	for i := 0; i < mval.Len(); i++ {
		val, err := valueToToml(mtype.Elem(), mval.Index(i))
		if err != nil {
			return nil, err
		}
		tval[i] = val
	}
	return tval, nil
}

// Convert given marshal value to toml value
func valueToToml(mtype reflect.Type, mval reflect.Value) (interface{}, error) {
	if mtype.Kind() == reflect.Ptr {
		return valueToToml(mtype.Elem(), mval.Elem())
	}
	switch {
	case isTree(mtype):
		return valueToTree(mtype, mval)
	case isTreeSlice(mtype):
		return valueToTreeSlice(mtype, mval)
	case isOtherSlice(mtype):
		return valueToOtherSlice(mtype, mval)
	default:
		switch mtype.Kind() {
		case reflect.Bool:
			return mval.Bool(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return mval.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return mval.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return mval.Float(), nil
		case reflect.String:
			return mval.String(), nil
		case reflect.Struct:
			return mval.Interface().(time.Time), nil
		default:
			return nil, fmt.Errorf("Marshal can't handle %v(%v)", mtype, mtype.Kind())
		}
	}
}

//Unmarshal ...
func Unmarshal(data []byte, v interface{}) error {
	mtype := reflect.TypeOf(v)
	if mtype.Kind() != reflect.Ptr || mtype.Elem().Kind() != reflect.Struct {
		return errors.New("Only a pointer to Struct can be unmarshaled from TOML")
	}

	t, err := Load(string(data))
	if err != nil {
		return err
	}

	sval, err := valueFromTree(mtype.Elem(), t)
	if err != nil {
		return err
	}
	reflect.ValueOf(v).Elem().Set(sval)
	return nil
}

// Convert toml tree to marshal struct or map, using marshal type
func valueFromTree(mtype reflect.Type, tval *TomlTree) (reflect.Value, error) {
	if mtype.Kind() == reflect.Ptr {
		return unwrapPointer(mtype, tval)
	}
	var mval reflect.Value
	switch mtype.Kind() {
	case reflect.Struct:
		mval = reflect.New(mtype).Elem()
		for i := 0; i < mtype.NumField(); i++ {
			mtypef := mtype.Field(i)
			if mtypef.PkgPath == "" {
				key := tomlName(mtypef)
				exists := tval.Has(key)
				if exists {
					val := tval.Get(key)
					mvalf, err := valueFromToml(mtypef.Type, val)
					if err != nil {
						if err.Error()[0] == '(' {
							return mval, err
						}
						return mval, fmt.Errorf("%s: %s", tval.GetPosition(key), err)
					}
					mval.Field(i).Set(mvalf)
				}
			}
		}
	case reflect.Map:
		mval = reflect.MakeMap(mtype)
		for _, key := range tval.Keys() {
			val := tval.Get(key)
			mvalf, err := valueFromToml(mtype.Elem(), val)
			if err != nil {
				return mval, err
			}
			mval.SetMapIndex(reflect.ValueOf(key), mvalf)
		}
	}
	return mval, nil
}

// Convert toml value to marshal struct/map slice, using marshal type
func valueFromTreeSlice(mtype reflect.Type, tval []*TomlTree) (reflect.Value, error) {
	if mtype.Kind() == reflect.Ptr {
		return unwrapPointer(mtype, tval)
	}
	mval := reflect.MakeSlice(mtype, len(tval), len(tval))
	for i := 0; i < len(tval); i++ {
		val, err := valueFromTree(mtype.Elem(), tval[i])
		if err != nil {
			return mval, err
		}
		mval.Index(i).Set(val)
	}
	return mval, nil
}

// Convert toml value to marshal primitive slice, using marshal type
func valueFromOtherSlice(mtype reflect.Type, tval []interface{}) (reflect.Value, error) {
	if mtype.Kind() == reflect.Ptr {
		return unwrapPointer(mtype, tval)
	}
	mval := reflect.MakeSlice(mtype, len(tval), len(tval))
	for i := 0; i < len(tval); i++ {
		val, err := valueFromToml(mtype.Elem(), tval[i])
		if err != nil {
			return mval, err
		}
		mval.Index(i).Set(val)
	}
	return mval, nil
}

// Convert toml value to marshal value, using marshal type
func valueFromToml(mtype reflect.Type, tval interface{}) (reflect.Value, error) {
	if mtype.Kind() == reflect.Ptr {
		return unwrapPointer(mtype, tval)
	}
	switch {
	case isTree(mtype):
		return valueFromTree(mtype, tval.(*TomlTree))
	case isTreeSlice(mtype):
		return valueFromTreeSlice(mtype, tval.([]*TomlTree))
	case isOtherSlice(mtype):
		return valueFromOtherSlice(mtype, tval.([]interface{}))
	default:
		switch mtype.Kind() {
		case reflect.Bool:
			val, ok := tval.(bool)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to bool", tval, tval)
			}
			return reflect.ValueOf(val), nil
		case reflect.Int:
			val, ok := tval.(int64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to int", tval, tval)
			}
			return reflect.ValueOf(int(val)), nil
		case reflect.Int8:
			val, ok := tval.(int64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to int", tval, tval)
			}
			return reflect.ValueOf(int8(val)), nil
		case reflect.Int16:
			val, ok := tval.(int64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to int", tval, tval)
			}
			return reflect.ValueOf(int16(val)), nil
		case reflect.Int32:
			val, ok := tval.(int64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to int", tval, tval)
			}
			return reflect.ValueOf(int32(val)), nil
		case reflect.Int64:
			val, ok := tval.(int64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to int", tval, tval)
			}
			return reflect.ValueOf(val), nil
		case reflect.Uint:
			val, ok := tval.(uint64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to uint", tval, tval)
			}
			return reflect.ValueOf(uint(val)), nil
		case reflect.Uint8:
			val, ok := tval.(uint64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to uint", tval, tval)
			}
			return reflect.ValueOf(uint8(val)), nil
		case reflect.Uint16:
			val, ok := tval.(uint64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to uint", tval, tval)
			}
			return reflect.ValueOf(uint16(val)), nil
		case reflect.Uint32:
			val, ok := tval.(uint64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to uint", tval, tval)
			}
			return reflect.ValueOf(uint32(val)), nil
		case reflect.Uint64:
			val, ok := tval.(uint64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to uint", tval, tval)
			}
			return reflect.ValueOf(val), nil
		case reflect.Float32:
			val, ok := tval.(float64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to float", tval, tval)
			}
			return reflect.ValueOf(float32(val)), nil
		case reflect.Float64:
			val, ok := tval.(float64)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to float", tval, tval)
			}
			return reflect.ValueOf(val), nil
		case reflect.String:
			val, ok := tval.(string)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to string", tval, tval)
			}
			return reflect.ValueOf(val), nil
		case reflect.Struct:
			val, ok := tval.(time.Time)
			if !ok {
				return reflect.ValueOf(nil), fmt.Errorf("Can't convert %v(%T) to time", tval, tval)
			}
			return reflect.ValueOf(val), nil
		default:
			return reflect.ValueOf(nil), fmt.Errorf("Unmarshal can't handle %v(%v)", mtype, mtype.Kind())
		}
	}
}

func unwrapPointer(mtype reflect.Type, tval interface{}) (reflect.Value, error) {
	val, err := valueFromToml(mtype.Elem(), tval)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	mval := reflect.New(mtype.Elem())
	mval.Elem().Set(val)
	return mval, nil
}

func tomlName(vf reflect.StructField) string {
	name := vf.Tag.Get("toml")
	if name == "" {
		name = strings.ToLower(vf.Name)
	}
	return name
}

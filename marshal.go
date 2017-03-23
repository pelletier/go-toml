package toml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Marshal ...
func Marshal(v interface{}) ([]byte, error) {
	styp := reflect.TypeOf(v)
	if styp.Kind() != reflect.Struct {
		return []byte{}, errors.New("Only a Struct can be marshaled to TOML")
	}
	sval := reflect.ValueOf(v)
	t, err := reflectTreeFromStruct(styp, sval)
	if err != nil {
		return []byte{}, err
	}
	s, err := t.ToTomlString()
	return []byte(s), err
}

func reflectTreeFromStruct(styp reflect.Type, sval reflect.Value) (*TomlTree, error) {
	t := newTomlTree()
	for i := 0; i < styp.NumField(); i++ {
		stypf, svalf := styp.Field(i), sval.Field(i)
		val, err := reflectTreeFromValue(stypf.Type, svalf)
		if err != nil {
			return nil, err
		}
		t.Set(tomlName(stypf), val)
	}
	return t, nil
}

func reflectTreeFromMap(styp reflect.Type, sval reflect.Value) (*TomlTree, error) {
	t := newTomlTree()
	for _, key := range sval.MapKeys() {
		svalf := sval.MapIndex(key)
		val, err := reflectTreeFromValue(styp.Elem(), svalf)
		if err != nil {
			return nil, err
		}
		t.Set(key.String(), val)
	}
	return t, nil
}

func reflectTreeFromBasicSlice(styp reflect.Type, sval reflect.Value) ([]interface{}, error) {
	t := make([]interface{}, sval.Len(), sval.Len())
	for i := 0; i < sval.Len(); i++ {
		t[i] = upgradeValue(sval.Index(i).Interface())
	}
	return t, nil
}

func reflectTreeFromStructSlice(styp reflect.Type, sval reflect.Value) ([]*TomlTree, error) {
	t := make([]*TomlTree, sval.Len(), sval.Len())
	for i := 0; i < sval.Len(); i++ {
		val, err := reflectTreeFromStruct(styp.Elem(), sval.Index(i))
		if err != nil {
			return nil, err
		}
		t[i] = val
	}
	return t, nil
}

func reflectTreeFromValue(styp reflect.Type, sval reflect.Value) (interface{}, error) {
	var val interface{}
	var typ reflect.Type

	if styp.Kind() == reflect.Interface {
		typ = sval.Type()
	} else {
		typ = styp
	}

	switch typ.Kind() {
	case reflect.Bool:
		val = sval.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = sval.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val = sval.Uint()
	case reflect.Float32, reflect.Float64:
		val = sval.Float()
	case reflect.String:
		val = sval.String()
	case reflect.Interface:
		val = sval.Interface()
	case reflect.Struct:
		if tmp, ok := sval.Interface().(time.Time); ok {
			val = tmp
		} else {
			return reflectTreeFromStruct(typ, sval)
		}
	case reflect.Slice:
		switch typ.Elem().Kind() {
		case reflect.Struct:
			if styp.Elem().String() == "time.Time" {
				return reflectTreeFromBasicSlice(typ, sval)
			}
			return reflectTreeFromStructSlice(typ, sval)
		default:
			return reflectTreeFromBasicSlice(typ, sval)
		}
	case reflect.Map:
		return reflectTreeFromMap(typ, sval)
	case reflect.Array:
		fallthrough
	default:
		return nil, fmt.Errorf("Marshal can't handle %v(%v)", typ, typ.Kind())
	}
	return val, nil
}

func upgradeValue(val interface{}) interface{} {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case uint:
		return uint64(v)
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case float32:
		return float64(v)
	default:
		return v
	}
}

//Unmarshal ...
func Unmarshal(data []byte, v interface{}) error {
	styp := reflect.TypeOf(v)
	if styp.Kind() != reflect.Ptr || styp.Elem().Kind() != reflect.Struct {
		return errors.New("Only a pointer to Struct can be unmarshaled from TOML")
	}
	sval := reflect.ValueOf(v)

	t, err := Load(string(data))
	if err != nil {
		return err
	}

	return reflectTreeToStruct(styp.Elem(), sval.Elem(), t)
}

func reflectTreeToStruct(styp reflect.Type, sval reflect.Value, t *TomlTree) error {
	for i := 0; i < styp.NumField(); i++ {
		stypf, svalf := styp.Field(i), sval.Field(i)
		key := tomlName(stypf)
		exists := t.Has(key)
		if exists {
			val := t.Get(key)
			err := reflectTreeToValue(stypf.Type, svalf, val)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func reflectTreeToBasicSlice(styp reflect.Type, sval reflect.Value, slc []interface{}) error {
	sval.Set(reflect.MakeSlice(styp, len(slc), len(slc)))
	for i := 0; i < len(slc); i++ {
		svalf := sval.Index(i)
		err := reflectTreeToValue(styp.Elem(), svalf, slc[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func reflectTreeToStructSlice(styp reflect.Type, sval reflect.Value, slc []*TomlTree) error {
	sval.Set(reflect.MakeSlice(styp, len(slc), len(slc)))
	for i := 0; i < len(slc); i++ {
		svalf := sval.Index(i)
		err := reflectTreeToStruct(styp.Elem(), svalf, slc[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func reflectTreeToValue(styp reflect.Type, sval reflect.Value, val interface{}) error {
	switch styp.Kind() {
	case reflect.Bool:
		sval.SetBool(val.(bool))
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		sval.SetInt(val.(int64))
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		sval.SetUint(val.(uint64))
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		sval.SetFloat(val.(float64))
	case reflect.String:
		sval.SetString(val.(string))
	case reflect.Struct:
		reflectTreeToStruct(styp, sval, val.(*TomlTree))
	case reflect.Slice:
		switch val.(type) {
		case []*TomlTree:
			reflectTreeToStructSlice(styp, sval, val.([]*TomlTree))
		default:
			reflectTreeToBasicSlice(styp, sval, val.([]interface{}))
		}
	case reflect.Map:
		fallthrough
	case reflect.Array:
		fallthrough
	default:
	}
	return nil
}

func tomlName(vf reflect.StructField) string {
	name := vf.Tag.Get("toml")
	if name == "" {
		name = strings.ToLower(vf.Name)
	}
	return name
}

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
	case reflect.Struct:
		if styp.String() == "time.Time" {
			val = sval.Interface().(time.Time)
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

	t, err := Load(string(data))
	if err != nil {
		return err
	}

	sval, err := reflectTreeToStruct(styp.Elem(), t)
	if err != nil {
		return err
	}
	reflect.ValueOf(v).Elem().Set(sval)
	return nil
}

func reflectTreeToStruct(styp reflect.Type, t *TomlTree) (reflect.Value, error) {
	sval := reflect.New(styp).Elem()
	for i := 0; i < styp.NumField(); i++ {
		stypf := styp.Field(i)
		key := tomlName(stypf)
		exists := t.Has(key)
		if exists {
			val := t.Get(key)
			svalf, err := reflectTreeToValue(stypf.Type, val)
			if err != nil {
				return sval, err
			}
			sval.Field(i).Set(svalf)
		}
	}
	return sval, nil
}

func reflectTreeToMap(styp reflect.Type, t *TomlTree) (reflect.Value, error) {
	sval := reflect.MakeMap(styp)
	for _, key := range t.Keys() {
		val := t.Get(key)
		svalf, err := reflectTreeToValue(styp.Elem(), val)
		if err != nil {
			return sval, err
		}
		sval.SetMapIndex(reflect.ValueOf(key), svalf)
	}
	return sval, nil
}

func reflectTreeToBasicSlice(styp reflect.Type, slc []interface{}) (reflect.Value, error) {
	sval := reflect.MakeSlice(styp, len(slc), len(slc))
	for i := 0; i < len(slc); i++ {
		svalf, err := reflectTreeToValue(styp.Elem(), slc[i])
		if err != nil {
			return sval, err
		}
		sval.Index(i).Set(svalf)
	}
	return sval, nil
}

func reflectTreeToStructSlice(styp reflect.Type, slc []*TomlTree) (reflect.Value, error) {
	sval := reflect.MakeSlice(styp, len(slc), len(slc))
	for i := 0; i < len(slc); i++ {
		svalf, err := reflectTreeToStruct(styp.Elem(), slc[i])
		if err != nil {
			return sval, err
		}
		sval.Index(i).Set(svalf)
	}
	return sval, nil
}

func reflectTreeToValue(styp reflect.Type, val interface{}) (reflect.Value, error) {
	switch styp.Kind() {
	case reflect.Bool:
		return reflect.ValueOf(val.(bool)), nil
	case reflect.Int:
		return reflect.ValueOf(int(val.(int64))), nil
	case reflect.Int8:
		return reflect.ValueOf(int8(val.(int64))), nil
	case reflect.Int16:
		return reflect.ValueOf(int16(val.(int64))), nil
	case reflect.Int32:
		return reflect.ValueOf(int32(val.(int64))), nil
	case reflect.Int64:
		return reflect.ValueOf(val.(int64)), nil
	case reflect.Uint:
		return reflect.ValueOf(uint(val.(uint64))), nil
	case reflect.Uint8:
		return reflect.ValueOf(uint8(val.(uint64))), nil
	case reflect.Uint16:
		return reflect.ValueOf(uint16(val.(uint64))), nil
	case reflect.Uint32:
		return reflect.ValueOf(uint32(val.(uint64))), nil
	case reflect.Uint64:
		return reflect.ValueOf(val.(uint64)), nil
	case reflect.Float32:
		return reflect.ValueOf(float32(val.(float64))), nil
	case reflect.Float64:
		return reflect.ValueOf(val.(float64)), nil
	case reflect.String:
		return reflect.ValueOf(val.(string)), nil
	case reflect.Struct:
		if styp.String() == "time.Time" {
			return reflect.ValueOf(val.(time.Time)), nil
		} else {
			return reflectTreeToStruct(styp, val.(*TomlTree))
		}
	case reflect.Slice:
		switch val.(type) {
		case []*TomlTree:
			return reflectTreeToStructSlice(styp, val.([]*TomlTree))
		default:
			return reflectTreeToBasicSlice(styp, val.([]interface{}))
		}
	case reflect.Map:
		return reflectTreeToMap(styp, val.(*TomlTree))
	case reflect.Array:
		fallthrough
	default:
	}
	return reflect.Zero(reflect.TypeOf("")), nil
}

func tomlName(vf reflect.StructField) string {
	name := vf.Tag.Get("toml")
	if name == "" {
		name = strings.ToLower(vf.Name)
	}
	return name
}

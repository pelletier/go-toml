package toml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Marshal ...
func Marshal(v interface{}) ([]byte, error) {
	vt := reflect.TypeOf(v)
	if vt.Kind() != reflect.Struct {
		return []byte{}, errors.New("Only a Struct can be marshaled to TOML")
	}
	vv := reflect.ValueOf(v)
	t := newTomlTree()
	path := []string{}
	err := reflectTreeFromStruct(vt, vv, t, path)
	if err != nil {
		return []byte{}, err
	}
	s, err := t.ToTomlString()
	return []byte(s), err
}

func reflectTreeFromStruct(vt reflect.Type, vv reflect.Value, t *TomlTree, path []string) error {
	for i := 0; i < vt.NumField(); i++ {
		vtf, vvf := vt.Field(i), vv.Field(i)
		keys := append(path, tomlName(vtf))
		err := reflectTreeFromValue(vtf.Type, vvf, t, keys)
		if err != nil {
			return err
		}
	}
	return nil
}

func reflectTreeFromValue(vt reflect.Type, vv reflect.Value, t *TomlTree, path []string) error {
	switch vt.Kind() {
	case reflect.Bool:
		t.SetPath(path, vv.Bool())
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		t.SetPath(path, vv.Int())
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		t.SetPath(path, vv.Uint())
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		t.SetPath(path, vv.Float())
	case reflect.String:
		t.SetPath(path, vv.String())
	case reflect.Struct:
		reflectTreeFromStruct(vt, vv, t, path)
	case reflect.Map:
		fallthrough
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		fallthrough
	default:
		return fmt.Errorf("Marshal can't handle %v(%v)", vt, vt.Kind())
	}
	return nil
}

//Unmarshal ...
func Unmarshal(data []byte, v interface{}) error {
	vt := reflect.TypeOf(v)
	if vt.Kind() != reflect.Ptr || vt.Elem().Kind() != reflect.Struct {
		return errors.New("Only a pointer to Struct can be unmarshaled from TOML")
	}
	vv := reflect.ValueOf(v)

	t, err := Load(string(data))
	if err != nil {
		return err
	}

	return reflectTreeToStruct(vt.Elem(), vv.Elem(), t, []string{})
}

func reflectTreeToStruct(vt reflect.Type, vv reflect.Value, t *TomlTree, path []string) error {
	for i := 0; i < vt.NumField(); i++ {
		vtf, vvf := vt.Field(i), vv.Field(i)
		keys := append(path, tomlName(vtf))
		exists := t.HasPath(keys)
		if exists {
			err := reflectTreeToValue(vtf.Type, vvf, t, keys)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func reflectTreeToValue(vt reflect.Type, vv reflect.Value, t *TomlTree, path []string) error {
	switch vt.Kind() {
	case reflect.Bool:
		vv.SetBool(t.GetPath(path).(bool))
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		vv.SetInt(t.GetPath(path).(int64))
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		vv.SetUint(t.GetPath(path).(uint64))
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		vv.SetFloat(t.GetPath(path).(float64))
	case reflect.String:
		vv.SetString(t.GetPath(path).(string))
	case reflect.Struct:
		reflectTreeToStruct(vt, vv, t, path)
	case reflect.Map:
		fallthrough
	case reflect.Slice:
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

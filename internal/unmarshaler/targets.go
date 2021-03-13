package unmarshaler

import (
	"fmt"
	"reflect"
)

type target interface {
	// Ensure the target's reflect value is not nil.
	ensure()

	// Store a string at the target.
	setString(v string) error

	// Appends an arbitrary value to the container.
	pushValue(v reflect.Value) error

	// Dereferences the target.
	get() reflect.Value
}

// struct target just contain the reflect.Value of the target field.
type structTarget reflect.Value

func (t structTarget) get() reflect.Value {
	return reflect.Value(t)
}

func (t structTarget) ensure() {
	f := t.get()
	if !f.IsNil() {
		return
	}

	switch f.Kind() {
	case reflect.Slice:
		f.Set(reflect.MakeSlice(f.Type(), 0, 0))
	default:
		panic(fmt.Errorf("don't know how to ensure %s", f.Kind()))
	}
}

func (t structTarget) setString(v string) error {
	f := t.get()
	if f.Kind() != reflect.String {
		return fmt.Errorf("cannot assign string to a %s", f.String())
	}
	f.SetString(v)
	return nil
}

func (t structTarget) pushValue(v reflect.Value) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Slice:
		t.ensure()
		f.Set(reflect.Append(f, v))
	default:
		return fmt.Errorf("cannot push %s on a %s", v.Kind(), f.Kind())
	}

	return nil
}

func scope(v reflect.Value, name string) (target, error) {
	switch v.Kind() {
	case reflect.Struct:
		return scopeStruct(v, name)
	default:
		panic(fmt.Errorf("can't scope on a %s", v.Kind()))
	}
}

func scopeStruct(v reflect.Value, name string) (target, error) {
	// TODO: cache this
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			// only consider exported fields
			continue
		}
		if f.Anonymous {
			// TODO: handle embedded structs
		} else {
			// TODO: handle names variations
			if f.Name == name {
				return structTarget(v.Field(i)), nil
			}
		}
	}
	return nil, fmt.Errorf("field '%s' not found on %s", name, v.Type())
}

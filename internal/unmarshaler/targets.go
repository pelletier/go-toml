package unmarshaler

import (
	"fmt"
	"reflect"
)

type target interface {
	// Ensure the target's value is compatible with a slice and initialized.
	ensureSlice() error

	// Store a string at the target.
	setString(v string) error

	// Store a boolean at the target
	setBool(v bool) error

	// Store an int64 at the target
	setInt64(v int64) error

	// Store a float64 at the target
	setFloat64(v float64) error

	// Creates a new value of the container's element type, and returns a
	// target to it.
	pushNew() (target, error)

	// Dereferences the target.
	get() reflect.Value
}

// valueTarget just contains a reflect.Value that can be set.
// It is used for struct fields.
type valueTarget reflect.Value

func (t valueTarget) get() reflect.Value {
	return reflect.Value(t)
}

func (t valueTarget) ensureSlice() error {
	f := t.get()

	switch f.Type().Kind() {
	case reflect.Slice:
		if f.IsNil() {
			f.Set(reflect.MakeSlice(f.Type(), 0, 0))
		}
	case reflect.Interface:
		if f.IsNil() {
			f.Set(reflect.MakeSlice(reflect.TypeOf([]interface{}{}), 0, 0))
		} else {
			if f.Type().Elem().Kind() != reflect.Slice {
				return fmt.Errorf("interface is pointing to a %s, not a slice", f.Kind())
			}
		}
	default:
		return fmt.Errorf("cannot initialize a slice in %s", f.Kind())
	}
	return nil
}

func (t valueTarget) setString(v string) error {
	f := t.get()

	switch f.Kind() {
	case reflect.String:
		f.SetString(v)
	case reflect.Interface:
		f.Set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign string to a %s", f.String())
	}

	return nil
}

func (t valueTarget) setBool(v bool) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Bool:
		f.SetBool(v)
	case reflect.Interface:
		f.Set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign bool to a %s", f.String())
	}

	return nil
}

func (t valueTarget) setInt64(v int64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// TODO: overflow checks
		f.SetInt(v)
	case reflect.Interface:
		f.Set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign int64 to a %s", f.String())
	}

	return nil
}

func (t valueTarget) setFloat64(v float64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Float32, reflect.Float64:
		// TODO: overflow checks
		f.SetFloat(v)
	case reflect.Interface:
		f.Set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign float64 to a %s", f.String())
	}

	return nil
}

func (t valueTarget) pushNew() (target, error) {
	f := t.get()

	switch f.Kind() {
	case reflect.Slice:
		idx := f.Len()
		f.Set(reflect.Append(f, reflect.New(f.Type().Elem()).Elem()))
		return valueTarget(f.Index(idx)), nil
	case reflect.Interface:
		if f.IsNil() {
			panic("interface should have been initialized")
		}
		ifaceElem := f.Elem()
		if ifaceElem.Kind() != reflect.Slice {
			return nil, fmt.Errorf("cannot pushNew on a %s", f.Kind())
		}
		idx := ifaceElem.Len()
		newElem := reflect.New(ifaceElem.Type().Elem()).Elem()
		newSlice := reflect.Append(ifaceElem, newElem)
		f.Set(newSlice)
		return valueTarget(f.Elem().Index(idx)), nil
	default:
		return nil, fmt.Errorf("cannot pushNew on a %s", f.Kind())
	}
}

func scopeTarget(t target, name string) (target, error) {
	x := t.get()
	return scope(x, name)
}

func scope(v reflect.Value, name string) (target, error) {
	switch v.Kind() {
	case reflect.Struct:
		return scopeStruct(v, name)
	case reflect.Interface:
		if v.IsNil() {
			panic("not implemented") // TODO
		} else {
			return scope(v.Elem(), name)
		}
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
				return valueTarget(v.Field(i)), nil
			}
		}
	}
	return nil, fmt.Errorf("field '%s' not found on %s", name, v.Type())
}

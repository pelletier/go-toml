package toml

import (
	"fmt"
	"reflect"
)

type target interface {
	// Dereferences the target.
	get() reflect.Value

	// Store a string at the target.
	setString(v string) error

	// Store a boolean at the target
	setBool(v bool) error

	// Store an int64 at the target
	setInt64(v int64) error

	// Store a float64 at the target
	setFloat64(v float64) error

	// Stores any  value at the target
	set(v reflect.Value) error
}

// valueTarget just contains a reflect.Value that can be set.
// It is used for struct fields.
type valueTarget reflect.Value

func (t valueTarget) get() reflect.Value {
	return reflect.Value(t)
}

func (t valueTarget) set(v reflect.Value) error {
	reflect.Value(t).Set(v)
	return nil
}

func (t valueTarget) setString(v string) error {
	t.get().SetString(v)
	return nil
}

func (t valueTarget) setBool(v bool) error {
	t.get().SetBool(v)
	return nil
}

func (t valueTarget) setInt64(v int64) error {
	t.get().SetInt(v)
	return nil
}

func (t valueTarget) setFloat64(v float64) error {
	t.get().SetFloat(v)
	return nil
}

// mapTarget targets a specific key of a map.
type mapTarget struct {
	v reflect.Value
	k reflect.Value
}

func (t mapTarget) get() reflect.Value {
	return t.v.MapIndex(t.k)
}

func (t mapTarget) set(v reflect.Value) error {
	t.v.SetMapIndex(t.k, v)
	return nil
}

func (t mapTarget) setString(v string) error {
	return t.set(reflect.ValueOf(v))
}

func (t mapTarget) setBool(v bool) error {
	return t.set(reflect.ValueOf(v))
}

func (t mapTarget) setInt64(v int64) error {
	return t.set(reflect.ValueOf(v))
}

func (t mapTarget) setFloat64(v float64) error {
	return t.set(reflect.ValueOf(v))
}

func ensureSlice(t target) error {
	f := t.get()

	switch f.Type().Kind() {
	case reflect.Slice:
		if f.IsNil() {
			return t.set(reflect.MakeSlice(f.Type(), 0, 0))
		}
	case reflect.Interface:
		if f.IsNil() {
			return t.set(reflect.MakeSlice(reflect.TypeOf([]interface{}{}), 0, 0))
		}
		if f.Type().Elem().Kind() != reflect.Slice {
			return fmt.Errorf("interface is pointing to a %s, not a slice", f.Kind())
		}
	default:
		return fmt.Errorf("cannot initialize a slice in %s", f.Kind())
	}
	return nil
}

func setString(t target, v string) error {
	f := t.get()

	switch f.Kind() {
	case reflect.String:
		return t.setString(v)
	case reflect.Interface:
		return t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign string to a %s", f.Kind())
	}
}

func setBool(t target, v bool) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Bool:
		return t.setBool(v)
	case reflect.Interface:
		return t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign bool to a %s", f.String())
	}
}

func setInt64(t target, v int64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// TODO: overflow checks
		return t.setInt64(v)
	case reflect.Interface:
		return t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign int64 to a %s", f.String())
	}
}

func setFloat64(t target, v float64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Float32, reflect.Float64:
		// TODO: overflow checks
		return t.setFloat64(v)
	case reflect.Interface:
		return t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("cannot assign float64 to a %s", f.String())
	}
}

func pushNew(t target) (target, error) {
	f := t.get()

	switch f.Kind() {
	case reflect.Slice:
		idx := f.Len()
		err := t.set(reflect.Append(f, reflect.New(f.Type().Elem()).Elem()))
		if err != nil {
			return nil, err
		}
		return valueTarget(t.get().Index(idx)), nil
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
		err := t.set(newSlice)
		if err != nil {
			return nil, err
		}
		return valueTarget(t.get().Elem().Index(idx)), nil
	default:
		return nil, fmt.Errorf("cannot pushNew on a %s", f.Kind())
	}
}

func scopeTarget(t target, name string) (target, error) {
	x := t.get()
	return scope(x, name)
}

func scopeTableTarget(append bool, t target, name string) (target, error) {
	x := t.get()
	t, err := scope(x, name)
	if err != nil {
		return t, err
	}
	x = t.get()
	if x.Kind() == reflect.Slice {
		return scopeSlice(t, append)
	}
	return t, nil
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
	case reflect.Map:
		return scopeMap(v, name)
	default:
		panic(fmt.Errorf("can't scope on a %s", v.Kind()))
	}
}

func scopeSlice(t target, append bool) (target, error) {
	v := t.get()
	if append {
		newElem := reflect.New(v.Type().Elem())
		newSlice := reflect.Append(v, newElem.Elem())
		err := t.set(newSlice)
		if err != nil {
			return t, err
		}
		v = t.get()
	}
	return valueTarget(v.Index(v.Len() - 1)), nil
}

func scopeMap(v reflect.Value, name string) (target, error) {
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	k := reflect.ValueOf(name)
	if !v.MapIndex(k).IsValid() {
		newElem := reflect.New(v.Type().Elem())
		v.SetMapIndex(k, newElem.Elem())
	}

	return mapTarget{
		v: v,
		k: k,
	}, nil
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

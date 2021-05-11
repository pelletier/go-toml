package toml

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
)

type target interface {
	// Dereferences the target.
	get() reflect.Value

	// Store a string at the target.
	setString(v string)

	// Store a boolean at the target
	setBool(v bool)

	// Store an int64 at the target
	setInt64(v int64)

	// Store a float64 at the target
	setFloat64(v float64)

	// Stores any value at the target
	set(v reflect.Value)
}

// valueTarget just contains a reflect.Value that can be set.
// It is used for struct fields.
type valueTarget reflect.Value

func (t valueTarget) get() reflect.Value {
	return reflect.Value(t)
}

func (t valueTarget) set(v reflect.Value) {
	reflect.Value(t).Set(v)
}

func (t valueTarget) setString(v string) {
	t.get().SetString(v)
}

func (t valueTarget) setBool(v bool) {
	t.get().SetBool(v)
}

func (t valueTarget) setInt64(v int64) {
	t.get().SetInt(v)
}

func (t valueTarget) setFloat64(v float64) {
	t.get().SetFloat(v)
}

// interfaceTarget wraps an other target to dereference on get.
type interfaceTarget struct {
	x target
}

func (t interfaceTarget) get() reflect.Value {
	return t.x.get().Elem()
}

func (t interfaceTarget) set(v reflect.Value) {
	t.x.set(v)
}

func (t interfaceTarget) setString(v string) {
	panic("interface targets should always go through set")
}

func (t interfaceTarget) setBool(v bool) {
	panic("interface targets should always go through set")
}

func (t interfaceTarget) setInt64(v int64) {
	panic("interface targets should always go through set")
}

func (t interfaceTarget) setFloat64(v float64) {
	panic("interface targets should always go through set")
}

// mapTarget targets a specific key of a map.
type mapTarget struct {
	v reflect.Value
	k reflect.Value
}

func (t mapTarget) get() reflect.Value {
	return t.v.MapIndex(t.k)
}

func (t mapTarget) set(v reflect.Value) {
	t.v.SetMapIndex(t.k, v)
}

func (t mapTarget) setString(v string) {
	t.set(reflect.ValueOf(v))
}

func (t mapTarget) setBool(v bool) {
	t.set(reflect.ValueOf(v))
}

func (t mapTarget) setInt64(v int64) {
	t.set(reflect.ValueOf(v))
}

func (t mapTarget) setFloat64(v float64) {
	t.set(reflect.ValueOf(v))
}

// makes sure that the value pointed at by t is indexable (Slice, Array), or
// dereferences to an indexable (Ptr, Interface).
func ensureValueIndexable(t target) error {
	f := t.get()

	switch f.Type().Kind() {
	case reflect.Slice:
		if f.IsNil() {
			t.set(reflect.MakeSlice(f.Type(), 0, 0))
			return nil
		}
	case reflect.Interface:
		if f.IsNil() || f.Elem().Type() != sliceInterfaceType {
			t.set(reflect.MakeSlice(sliceInterfaceType, 0, 0))
			return nil
		}
	case reflect.Ptr:
		panic("pointer should have already been dereferenced")
	case reflect.Array:
		// arrays are always initialized.
	default:
		return fmt.Errorf("toml: cannot store array in a %s", f.Kind())
	}

	return nil
}

var (
	sliceInterfaceType     = reflect.TypeOf([]interface{}{})
	mapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})
)

func ensureMapIfInterface(x target) {
	v := x.get()

	if v.Kind() == reflect.Interface && v.IsNil() {
		newElement := reflect.MakeMap(mapStringInterfaceType)

		x.set(newElement)
	}
}

func setString(t target, v string) error {
	f := t.get()

	switch f.Kind() {
	case reflect.String:
		t.setString(v)
	case reflect.Interface:
		t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("toml: cannot assign string to a %s", f.Kind())
	}

	return nil
}

func setBool(t target, v bool) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Bool:
		t.setBool(v)
	case reflect.Interface:
		t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("toml: cannot assign boolean to a %s", f.Kind())
	}

	return nil
}

const (
	maxInt = int64(^uint(0) >> 1)
	minInt = -maxInt - 1
)

//nolint:funlen,gocognit,cyclop
func setInt64(t target, v int64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Int64:
		t.setInt64(v)
	case reflect.Int32:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return fmt.Errorf("toml: number %d does not fit in an int32", v)
		}

		t.set(reflect.ValueOf(int32(v)))
		return nil
	case reflect.Int16:
		if v < math.MinInt16 || v > math.MaxInt16 {
			return fmt.Errorf("toml: number %d does not fit in an int16", v)
		}

		t.set(reflect.ValueOf(int16(v)))
	case reflect.Int8:
		if v < math.MinInt8 || v > math.MaxInt8 {
			return fmt.Errorf("toml: number %d does not fit in an int8", v)
		}

		t.set(reflect.ValueOf(int8(v)))
	case reflect.Int:
		if v < minInt || v > maxInt {
			return fmt.Errorf("toml: number %d does not fit in an int", v)
		}

		t.set(reflect.ValueOf(int(v)))
	case reflect.Uint64:
		if v < 0 {
			return fmt.Errorf("toml: negative number %d does not fit in an uint64", v)
		}

		t.set(reflect.ValueOf(uint64(v)))
	case reflect.Uint32:
		if v < 0 || v > math.MaxUint32 {
			return fmt.Errorf("toml: negative number %d does not fit in an uint32", v)
		}

		t.set(reflect.ValueOf(uint32(v)))
	case reflect.Uint16:
		if v < 0 || v > math.MaxUint16 {
			return fmt.Errorf("toml: negative number %d does not fit in an uint16", v)
		}

		t.set(reflect.ValueOf(uint16(v)))
	case reflect.Uint8:
		if v < 0 || v > math.MaxUint8 {
			return fmt.Errorf("toml: negative number %d does not fit in an uint8", v)
		}

		t.set(reflect.ValueOf(uint8(v)))
	case reflect.Uint:
		if v < 0 {
			return fmt.Errorf("toml: negative number %d does not fit in an uint", v)
		}

		t.set(reflect.ValueOf(uint(v)))
	case reflect.Interface:
		t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("toml: integer cannot be assigned to %s", f.Kind())
	}

	return nil
}

func setFloat64(t target, v float64) error {
	f := t.get()

	switch f.Kind() {
	case reflect.Float64:
		t.setFloat64(v)
	case reflect.Float32:
		if v > math.MaxFloat32 {
			return fmt.Errorf("toml: number %f does not fit in a float32", v)
		}

		t.set(reflect.ValueOf(float32(v)))
	case reflect.Interface:
		t.set(reflect.ValueOf(v))
	default:
		return fmt.Errorf("toml: float cannot be assigned to %s", f.Kind())
	}

	return nil
}

// Returns the element at idx of the value pointed at by target, or an error if
// t does not point to an indexable.
// If the target points to an Array and idx is out of bounds, it returns
// (nil, nil) as this is not a fatal error (the unmarshaler will skip).
func elementAt(t target, idx int) target {
	f := t.get()

	switch f.Kind() {
	case reflect.Slice:
		//nolint:godox
		// TODO: use the idx function argument and avoid alloc if possible.
		idx := f.Len()

		t.set(reflect.Append(f, reflect.New(f.Type().Elem()).Elem()))

		return valueTarget(t.get().Index(idx))
	case reflect.Array:
		if idx >= f.Len() {
			return nil
		}

		return valueTarget(f.Index(idx))
	case reflect.Interface:
		// This function is called after ensureValueIndexable, so it's
		// guaranteed that f contains an initialized slice.
		ifaceElem := f.Elem()
		idx := ifaceElem.Len()
		newElem := reflect.New(ifaceElem.Type().Elem()).Elem()
		newSlice := reflect.Append(ifaceElem, newElem)

		t.set(newSlice)

		return valueTarget(t.get().Elem().Index(idx))
	default:
		// Why ensureValueIndexable let it go through?
		panic(fmt.Errorf("elementAt received unhandled value type: %s", f.Kind()))
	}
}

func (d *decoder) scopeTableTarget(shouldAppend bool, t target, name string) (target, bool, error) {
	x := t.get()

	switch x.Kind() {
	// Kinds that need to recurse
	case reflect.Interface:
		t := scopeInterface(shouldAppend, t)
		return d.scopeTableTarget(shouldAppend, t, name)
	case reflect.Ptr:
		t := scopePtr(t)
		return d.scopeTableTarget(shouldAppend, t, name)
	case reflect.Slice:
		t := scopeSlice(shouldAppend, t)
		shouldAppend = false
		return d.scopeTableTarget(shouldAppend, t, name)
	case reflect.Array:
		t, err := d.scopeArray(shouldAppend, t)
		if err != nil {
			return t, false, err
		}
		shouldAppend = false

		return d.scopeTableTarget(shouldAppend, t, name)

	// Terminal kinds
	case reflect.Struct:
		return scopeStruct(x, name)
	case reflect.Map:
		if x.IsNil() {
			t.set(reflect.MakeMap(x.Type()))
			x = t.get()
		}

		return scopeMap(x, name)
	default:
		panic(fmt.Sprintf("can't scope on a %s", x.Kind()))
	}
}

func scopeInterface(shouldAppend bool, t target) target {
	initInterface(shouldAppend, t)
	return interfaceTarget{t}
}

func scopePtr(t target) target {
	initPtr(t)
	return valueTarget(t.get().Elem())
}

func initPtr(t target) {
	x := t.get()
	if !x.IsNil() {
		return
	}

	t.set(reflect.New(x.Type().Elem()))
}

// initInterface makes sure that the interface pointed at by the target is not
// nil.
// Returns the target to the initialized value of the target.
func initInterface(shouldAppend bool, t target) {
	x := t.get()

	if x.Kind() != reflect.Interface {
		panic("this should only be called on interfaces")
	}

	if !x.IsNil() && (x.Elem().Type() == sliceInterfaceType || x.Elem().Type() == mapStringInterfaceType) {
		return
	}

	var newElement reflect.Value
	if shouldAppend {
		newElement = reflect.MakeSlice(sliceInterfaceType, 0, 0)
	} else {
		newElement = reflect.MakeMap(mapStringInterfaceType)
	}

	t.set(newElement)
}

func scopeSlice(shouldAppend bool, t target) target {
	v := t.get()

	if shouldAppend {
		newElem := reflect.New(v.Type().Elem())
		newSlice := reflect.Append(v, newElem.Elem())

		t.set(newSlice)

		v = t.get()
	}

	return valueTarget(v.Index(v.Len() - 1))
}

func (d *decoder) scopeArray(shouldAppend bool, t target) (target, error) {
	v := t.get()

	idx := d.arrayIndex(shouldAppend, v)

	if idx >= v.Len() {
		return nil, fmt.Errorf("toml: impossible to insert element beyond array's size: %d", v.Len())
	}

	return valueTarget(v.Index(idx)), nil
}

func scopeMap(v reflect.Value, name string) (target, bool, error) {
	k := reflect.ValueOf(name)

	keyType := v.Type().Key()
	if !k.Type().AssignableTo(keyType) {
		if !k.Type().ConvertibleTo(keyType) {
			return nil, false, fmt.Errorf("toml: cannot convert map key of type %s to expected type %s", k.Type(), keyType)
		}

		k = k.Convert(keyType)
	}

	if !v.MapIndex(k).IsValid() {
		newElem := reflect.New(v.Type().Elem())
		v.SetMapIndex(k, newElem.Elem())
	}

	return mapTarget{
		v: v,
		k: k,
	}, true, nil
}

type fieldPathsMap = map[string][]int

type fieldPathsCache struct {
	m map[reflect.Type]fieldPathsMap
	l sync.RWMutex
}

func (c *fieldPathsCache) get(t reflect.Type) (fieldPathsMap, bool) {
	c.l.RLock()
	paths, ok := c.m[t]
	c.l.RUnlock()

	return paths, ok
}

func (c *fieldPathsCache) set(t reflect.Type, m fieldPathsMap) {
	c.l.Lock()
	c.m[t] = m
	c.l.Unlock()
}

var globalFieldPathsCache = fieldPathsCache{
	m: map[reflect.Type]fieldPathsMap{},
	l: sync.RWMutex{},
}

func scopeStruct(v reflect.Value, name string) (target, bool, error) {
	//nolint:godox
	// TODO: cache this, and reduce allocations
	fieldPaths, ok := globalFieldPathsCache.get(v.Type())
	if !ok {
		fieldPaths = map[string][]int{}

		path := make([]int, 0, 16)

		var walk func(reflect.Value)
		walk = func(v reflect.Value) {
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				l := len(path)
				path = append(path, i)
				f := t.Field(i)

				if f.Anonymous {
					walk(v.Field(i))
				} else if f.PkgPath == "" {
					// only consider exported fields
					fieldName, ok := f.Tag.Lookup("toml")
					if !ok {
						fieldName = f.Name
					}

					pathCopy := make([]int, len(path))
					copy(pathCopy, path)

					fieldPaths[fieldName] = pathCopy
					// extra copy for the case-insensitive match
					fieldPaths[strings.ToLower(fieldName)] = pathCopy
				}
				path = path[:l]
			}
		}

		walk(v)

		globalFieldPathsCache.set(v.Type(), fieldPaths)
	}

	path, ok := fieldPaths[name]
	if !ok {
		path, ok = fieldPaths[strings.ToLower(name)]
	}

	if !ok {
		return nil, false, nil
	}

	return valueTarget(v.FieldByIndex(path)), true, nil
}

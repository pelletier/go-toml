// reflectbuild is a package that provides utility functions to build Go
// objects using reflection.
package reflectbuild

import (
	"fmt"
	"reflect"
	"strings"
)

// fieldGetters are functions that given a struct return a specific field
// (likely captured in their scope)
type fieldGetter func(s reflect.Value) reflect.Value

// collection of fieldGetters for a given struct type
type structFieldGetters map[string]fieldGetter

type target interface {
	get() reflect.Value
	set(value reflect.Value) error

	fmt.Stringer
}

type valueTarget reflect.Value

func (v valueTarget) get() reflect.Value {
	return reflect.Value(v)
}

func (v valueTarget) set(value reflect.Value) error {
	rv := reflect.Value(v)

	// value is guaranteed to be a pointer
	if value.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("set() should receive a pointer, not a '%s'", value.Kind()))
	}

	if rv.Kind() != reflect.Ptr {
		// TODO: check value is nil?
		value = value.Elem()
	}

	targetType := rv.Type()
	value, err := convert(targetType, value)
	if err != nil {
		return err
	}

	rv.Set(value)
	return nil
}

func (v valueTarget) String() string {
	return fmt.Sprintf("valueTarget: '%s' (%s)", reflect.Value(v), reflect.Value(v).Type())
}

type mapTarget struct {
	index reflect.Value
	m     reflect.Value
}

func (v mapTarget) get() reflect.Value {
	return v.m.MapIndex(v.index)
}

func (v mapTarget) set(value reflect.Value) error {
	// value is guaranteed to be a pointer

	if v.m.Type().Elem().Kind() != reflect.Ptr {
		// TODO: check value is nil?
		value = value.Elem()
	}

	targetType := v.m.Type().Elem()
	value, err := convert(targetType, value)
	if err != nil {
		return err
	}

	v.m.SetMapIndex(v.index, value)
	return nil
}

func (v mapTarget) String() string {
	return fmt.Sprintf("mapTarget: '%s'[%s]", v.m, v.index)
}

// Builder wraps a value and provides method to modify its structure.
// It is a stateful object that keeps a cursor of what part of the object is
// being modified.
// Create a Builder with NewBuilder.
type Builder struct {
	root reflect.Value
	// Root is always a pointer to a non-nil value.
	// Cursor is the top of the stack.
	stack []target
	// Struct field tag to use to retrieve name.
	nameTag string
	// Cache of functions to access specific fields.
	fieldGettersCache map[reflect.Type]structFieldGetters
}

func copyAndAppend(s []int, i int) []int {
	ns := make([]int, len(s)+1)
	copy(ns, s)
	ns[len(ns)-1] = i
	return ns
}

func (b *Builder) getOrGenerateFieldGettersRecursive(m structFieldGetters, idx []int, s reflect.Type) {
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.PkgPath != "" {
			// only consider exported fields
			continue
		}
		if f.Anonymous {
			b.getOrGenerateFieldGettersRecursive(m, copyAndAppend(idx, i), f.Type)
		} else {
			fieldName, ok := f.Tag.Lookup(b.nameTag)
			if !ok {
				fieldName = f.Name
			}

			if len(idx) == 0 {
				m[fieldName] = makeFieldGetterByIndex(i)
			} else {
				m[fieldName] = makeFieldGetterByIndexes(copyAndAppend(idx, i))
			}
		}
	}

	if b.fieldGettersCache == nil {
		b.fieldGettersCache = make(map[reflect.Type]structFieldGetters, 1)
	}

	b.fieldGettersCache[s] = m
}

func (b *Builder) getOrGenerateFieldGetters(s reflect.Type) structFieldGetters {
	if s.Kind() != reflect.Struct {
		panic("generateFieldGetters can only be called on a struct")
	}
	m, ok := b.fieldGettersCache[s]
	if ok {
		return m
	}

	m = make(structFieldGetters, s.NumField())
	b.getOrGenerateFieldGettersRecursive(m, nil, s)
	b.fieldGettersCache[s] = m
	return m
}

func makeFieldGetterByIndex(idx int) fieldGetter {
	return func(s reflect.Value) reflect.Value {
		return s.Field(idx)
	}
}

func makeFieldGetterByIndexes(idx []int) fieldGetter {
	return func(s reflect.Value) reflect.Value {
		return s.FieldByIndex(idx)
	}
}

func (b *Builder) fieldGetter(t reflect.Type, s string) (fieldGetter, error) {
	m := b.getOrGenerateFieldGetters(t)
	g, ok := m[s]
	if !ok {
		return nil, fmt.Errorf("field '%s' not accessible on '%s'", s, t)
	}
	return g, nil
}

// NewBuilder creates a Builder to construct v.
// If v is nil or not a pointer, an error will be returned.
func NewBuilder(tag string, v interface{}) (Builder, error) {
	if v == nil {
		return Builder{}, fmt.Errorf("cannot build a nil value")
	}

	rv := reflect.ValueOf(v)
	if rv.Type().Kind() != reflect.Ptr {
		return Builder{}, fmt.Errorf("cannot build a %s: need a pointer", rv.Type().Kind())
	}

	if rv.IsNil() {
		return Builder{}, fmt.Errorf("cannot build a nil value")
	}

	return Builder{
		root:    rv.Elem(),
		stack:   []target{valueTarget(rv.Elem())},
		nameTag: tag,
	}, nil
}

func (b *Builder) top() target {
	t := b.stack[len(b.stack)-1]
	fmt.Println("TOP:", t)
	return t
}

func (b *Builder) duplicate() {
	b.stack = append(b.stack, b.stack[len(b.stack)-1])
	// TODO: remove me. just here to make sure the method is included in the
	// binary for debug
	b.Dump()
}

func (b *Builder) pop() {
	b.stack = b.stack[:len(b.stack)-1]
	fmt.Println("POP: top:", b.stack[len(b.stack)-1])
}

func (b *Builder) len() int {
	return len(b.stack)
}

func (b *Builder) Dump() string {
	str := strings.Builder{}
	str.WriteByte('[')

	for i, x := range b.stack {
		if i > 0 {
			str.WriteString(" | ")
		}
		fmt.Fprintf(&str, "%s", x)
	}

	str.WriteByte(']')
	return str.String()
}

func (b *Builder) replace(v target) {
	fmt.Println("REPLACING:", v)
	b.stack[len(b.stack)-1] = v
}

var mapStringInterfaceType = reflect.TypeOf(map[string]interface{}{})

// DigField pushes the cursor into a field of the current struct.
// Dereferences all pointers found along the way.
// Errors if the current value is not a struct, or the field does not exist.
func (b *Builder) DigField(s string) error {
	t := b.top()
	v := t.get()

	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.Kind() == reflect.Interface {
			fmt.Println("STOP")
		}

		if v.IsNil() {
			if v.Kind() == reflect.Ptr {
				thing := reflect.New(v.Type().Elem())
				v.Set(thing)
			} else {
				v.Set(reflect.MakeMap(mapStringInterfaceType))
			}
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Map {
		// if map is nil, allocate it
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}

		// TODO: handle error when map is not indexed by strings
		key := reflect.ValueOf(s)

		key, err := convert(v.Type().Key(), key)
		if err != nil {
			return err
		}

		b.replace(mapTarget{
			index: key,
			m:     v,
		})
	} else {
		err := checkKind(v.Type(), reflect.Struct)
		if err != nil {
			return err
		}

		g, err := b.fieldGetter(v.Type(), s)
		if err != nil {
			return FieldNotFoundError{FieldName: s, Struct: v}
		}

		f := g(v)
		if !f.IsValid() {
			return FieldNotFoundError{FieldName: s, Struct: v}
		}

		b.replace(valueTarget(f))
	}

	return nil
}

// Save stores a copy of the current cursor position.
// It can be restored using Back().
// Save points are stored as a stack.
func (b *Builder) Save() {
	b.duplicate()
}

// Reset brings the cursor back to the root object.
func (b *Builder) Reset() {
	b.stack = b.stack[:1]
	b.stack[0] = valueTarget(b.root)
}

// Load is the opposite of Save. It discards the current cursor and loads the
// last saved cursor.
// Panics if no cursor has been saved.
func (b *Builder) Load() {
	if b.len() < 2 {
		panic(fmt.Errorf("tried to Back() when cursor was already at root"))
	}
	b.pop()
}

// Cursor returns the value pointed at by the cursor.
func (b *Builder) Cursor() reflect.Value {
	return b.top().get()
}

func (b *Builder) IsSlice() bool {
	return b.top().get().Kind() == reflect.Slice
}

func (b *Builder) IsSliceOrPtr() bool {
	return b.top().get().Kind() == reflect.Slice || (b.top().get().Kind() == reflect.Ptr && b.top().get().Type().Elem().Kind() == reflect.Slice)
}

// Last moves the cursor to the last value of the current value.
// For a slice or an array, it is the last element they contain, if any.
// For anything else, it's a no-op.
func (b *Builder) Last() {
	switch b.Cursor().Kind() {
	case reflect.Slice, reflect.Array:
		length := b.Cursor().Len()
		if length > 0 {
			x := b.Cursor().Index(length - 1)
			b.replace(valueTarget(x)) // TODO: create a "sliceTarget" ?
		}
	}
}

// SliceLastOrCreate moves the cursor to the last element of the slice if any.
// Otherwise creates a new element in that slice and moves to it.
func (b *Builder) SliceLastOrCreate() error {
	t := b.top()
	v := t.get()
	err := checkKind(v.Type(), reflect.Slice)
	if err != nil {
		return err
	}

	if v.Len() == 0 {
		return b.SliceNewElem()
	}
	b.Last()
	return nil
}

// SliceNewElem operates on a slice. It creates a new object (of type contained
// by the slice), append it to the slice, and moves the cursor to the new
// object.
func (b *Builder) SliceNewElem() error {
	t := b.top()
	v := t.get()

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	err := checkKind(v.Type(), reflect.Slice)
	if err != nil {
		return err
	}
	elem := reflect.New(v.Type().Elem())
	newSlice := reflect.Append(v, elem.Elem())
	v.Set(newSlice)
	b.replace(valueTarget(v.Index(v.Len() - 1))) // TODO: "sliceTarget"?
	return nil
}

func assertPtr(v reflect.Value) {
	if v.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("value '%s' should be a ptr, not '%s'", v, v.Kind()))
	}
}

func (b *Builder) SliceAppend(value reflect.Value) error {
	assertPtr(value)

	t := b.top()
	v := t.get()

	// pointer to a slice
	if v.Kind() == reflect.Ptr {
		// if the pointer is nil we need to allocate the slice
		if v.IsNil() {
			x := reflect.New(v.Type().Elem())
			v.Set(x)
		}
		// target the slice itself
		v = v.Elem()
	}

	err := checkKind(v.Type(), reflect.Slice)
	if err != nil {
		return err
	}

	if v.Type().Elem().Kind() == reflect.Ptr {
		// if it is a slice of pointers, we can just append
	} else {
		// otherwise we need to reference the value
		value = value.Elem()
	}

	if v.Type().Elem() != value.Type() {
		//nv, err := convert(v.Type().Elem(), value)
		//if err != nil {
		return fmt.Errorf("cannot assign '%s' to '%s'", value.Type(), v.Type().Elem())
		//}
		//value = nv
	}

	newSlice := reflect.Append(v, value)
	v.Set(newSlice)
	b.replace(valueTarget(v.Index(v.Len() - 1))) // TODO: "sliceTarget" ?
	return nil
}

// convert value so that it can be assigned to t.
//
// Conversion rules:
//
// * Pointers are de-referenced as needed.
// * Integer types are converted between each other as long as they don't
//   overflow.
// * Float types are converted between each other as long as they don't
//   overflow.
//
// TODO: this function acts as a switchboard. Runtime has enough information to
// generate per-type functions avoiding the double type switches.
func convert(t reflect.Type, value reflect.Value) (reflect.Value, error) {
	result := value

	if value.Type().AssignableTo(t) {
		return result, nil
	}

	if value.Kind() == reflect.Ptr {
		if t.Kind() != reflect.Ptr {
			return reflect.Value{}, fmt.Errorf("cannot convert pointer to non-pointer")
		}

		value = value.Elem()
		t = t.Elem()
	}

	var err error
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err = convertInt(t, value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err = convertUint(t, value)
	case reflect.Float32, reflect.Float64:
		value, err = convertFloat(t, value)
	default:
		err = fmt.Errorf("not converting a %s into a %s", value.Kind(), t.Kind())
	}

	if err != nil {
		return value, err
	}

	result = reflect.New(t)
	result.Elem().Set(value.Convert(t))
	return result.Elem(), nil
}

func convertInt(t reflect.Type, value reflect.Value) (reflect.Value, error) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Convert(t), nil // reflect.TypeOf(int64(0))
	default:
		return value, fmt.Errorf("cannot convert %s to integer (%s)", value.Kind(), t.Kind())
	}
}

func convertUint(t reflect.Type, value reflect.Value) (reflect.Value, error) {
	switch value.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Convert(t), nil // reflect.TypeOf(int64(0))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x := value.Int()
		if x < 0 {
			return value, fmt.Errorf("cannot store negative integer '%d' into %s", x, t.Kind())
		}
		return value.Convert(t), nil
	default:
		return value, fmt.Errorf("cannot convert %s to unsigned integer (%s)", value.Kind(), t.Kind())
	}
}

func convertFloat(t reflect.Type, value reflect.Value) (reflect.Value, error) {
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		return value.Convert(t), nil
	default:
		return value, fmt.Errorf("cannot convert %s to integer (%s)", value.Kind(), t.Kind())
	}
}

// Set the value at the cursor to the given string.
// Errors if a string cannot be assigned to the current value.
func (b *Builder) SetString(s string) error {
	t := b.top()
	v := t.get()

	if v.Kind() == reflect.Ptr {
		v.Set(reflect.ValueOf(&s))
		return nil
	}
	return t.set(reflect.ValueOf(s))
}

// Set the value at the cursor to the given boolean.
// Errors if a boolean cannot be assigned to the current value.
func (b *Builder) SetBool(value bool) error {
	t := b.top()
	v := t.get()

	err := checkKind(v.Type(), reflect.Bool)
	if err != nil {
		return err
	}

	v.SetBool(value)
	return nil
}

func (b *Builder) SetFloat(n float64) error {
	t := b.top()
	v := t.get()

	err := checkKindFloat(v.Type())
	if err != nil {
		return err
	}

	v.SetFloat(n)
	return nil
}

func (b *Builder) Set(v reflect.Value) error {
	assertPtr(v)
	t := b.top()
	return t.set(v)
}

// EnsureSlice makes sure that the cursor points to a non-nil slice.
func (b *Builder) EnsureSlice() error {
	t := b.top()
	v := t.get()

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return IncorrectKindError{
			Reason:   "EnsureSlice",
			Actual:   v.Kind(),
			Expected: []reflect.Kind{reflect.Slice},
		}
	}

	if v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}

	return nil
}

// EnsureStructOrMap makes sure that the cursor points to an initialized
// struct or map.
func (b *Builder) EnsureStructOrMap() error {
	t := b.top()
	v := t.get()

	switch v.Kind() {
	case reflect.Struct:
	case reflect.Map:
		if v.IsNil() {
			x := reflect.New(v.Type())
			x.Elem().Set(reflect.MakeMap(v.Type()))
			return t.set(x)
		}
	default:
		return IncorrectKindError{
			Reason:   "EnsureStructOrMap",
			Actual:   v.Kind(),
			Expected: []reflect.Kind{reflect.Struct, reflect.Map},
		}
	}
	return nil
}

func checkKindInt(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return nil
	}

	return IncorrectKindError{
		Reason:   "CheckKindInt",
		Actual:   rt.Kind(),
		Expected: []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64},
	}
}

func checkKindFloat(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Float32, reflect.Float64:
		return nil
	}

	return IncorrectKindError{
		Reason:   "CheckKindFloat",
		Actual:   rt.Kind(),
		Expected: []reflect.Kind{reflect.Float64},
	}
}

func checkKind(rt reflect.Type, expected reflect.Kind) error {
	if rt.Kind() != expected {
		return IncorrectKindError{
			Reason:   "CheckKind",
			Actual:   rt.Kind(),
			Expected: []reflect.Kind{expected},
		}
	}
	return nil
}

type IncorrectKindError struct {
	Reason   string
	Actual   reflect.Kind
	Expected []reflect.Kind
}

func (e IncorrectKindError) Error() string {
	b := strings.Builder{}
	b.WriteString("incorrect kind: ")

	if len(e.Expected) < 2 {
		b.WriteString(fmt.Sprintf("expected '%s', got '%s'", e.Expected[0], e.Actual))
	} else {
		b.WriteString(fmt.Sprintf("expected any of '%s', got '%s'", e.Expected, e.Actual))
	}

	if e.Reason != "" {
		b.WriteString(": ")
		b.WriteString(e.Reason)
	}

	return b.String()
}

type FieldNotFoundError struct {
	Struct    reflect.Value
	FieldName string
}

func (e FieldNotFoundError) Error() string {
	return fmt.Sprintf("field not found: '%s' on '%s'", e.FieldName, e.Struct.Type())
}

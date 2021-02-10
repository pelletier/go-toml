// reflectbuild is a package that provides utility functions to build Go
// objects using reflection.
package reflectbuild

import (
	"fmt"
	"reflect"
	"strings"
)

// Builder wraps a value and provides method to modify its structure.
// It is a stateful object that keeps a cursor of what part of the object is
// being modified.
// Create a Builder with NewBuilder.
type Builder struct {
	root reflect.Value
	// Root is always a pointer to a non-nil value.
	// Cursor is the top of the stack.
	stack []reflect.Value
}

// NewBuilder creates a Builder to construct v.
// If v is nil or not a pointer, an error will be returned.
func NewBuilder(v interface{}) (Builder, error) {
	if v == nil {
		return Builder{}, fmt.Errorf("cannot build a nil value")
	}

	rv := reflect.ValueOf(v)
	if rv.Type().Kind() != reflect.Ptr {
		return Builder{}, fmt.Errorf("cannot build a %s: need a pointer", rv.Type().Kind())
	}

	return Builder{
		root:  rv.Elem(),
		stack: []reflect.Value{rv.Elem()},
	}, nil
}

func (b *Builder) top() reflect.Value {
	return b.stack[len(b.stack)-1]
}

func (b *Builder) push(v reflect.Value) {
	b.stack = append(b.stack, v)
	// TODO: remove me. just here to make sure the method is included in the
	// binary for debug
	b.Dump()
}

func (b *Builder) pop() {
	b.stack = b.stack[:len(b.stack)-1]
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
		fmt.Fprintf(&str, "%s (%s)", x.Type(), x)
	}

	str.WriteByte(']')
	return str.String()
}

func (b *Builder) replace(v reflect.Value) {
	b.stack[len(b.stack)-1] = v
}

// DigField pushes the cursor into a field of the current struct.
// Dereferences all pointers found along the way.
// Errors if the current value is not a struct, or the field does not exist.
func (b *Builder) DigField(s string) error {
	t := b.top()

	for t.Kind() == reflect.Interface || t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	err := checkKind(t.Type(), reflect.Struct)
	if err != nil {
		return err
	}

	f := t.FieldByName(s)
	if !f.IsValid() {
		return FieldNotFoundError{FieldName: s, Struct: t}
	}

	b.replace(f)

	return nil
}

// Save stores a copy of the current cursor position.
// It can be restored using Back().
// Save points are stored as a stack.
func (b *Builder) Save() {
	b.push(b.top())
}

// Reset brings the cursor back to the root object.
func (b *Builder) Reset() {
	b.stack = b.stack[:1]
	b.stack[0] = b.root
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
	return b.top()
}

func (b *Builder) IsSlice() bool {
	return b.top().Kind() == reflect.Slice
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
			b.replace(x)
		}
	}
}

// SliceLastOrCreate moves the cursor to the last element of the slice if any.
// Otherwise creates a new element in that slice and moves to it.
func (b *Builder) SliceLastOrCreate() error {
	t := b.top()
	err := checkKind(t.Type(), reflect.Slice)
	if err != nil {
		return err
	}

	if t.Len() == 0 {
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
	err := checkKind(t.Type(), reflect.Slice)
	if err != nil {
		return err
	}
	elem := reflect.New(t.Type().Elem())
	newSlice := reflect.Append(t, elem.Elem())
	t.Set(newSlice)
	b.replace(t.Index(t.Len() - 1))
	return nil
}

func (b *Builder) SliceAppend(v reflect.Value) error {
	t := b.top()
	err := checkKind(t.Type(), reflect.Slice)
	if err != nil {
		return err
	}
	newSlice := reflect.Append(t, v)
	t.Set(newSlice)
	b.replace(t.Index(t.Len() - 1))
	return nil
}

// Set the value at the cursor to the given string.
// Errors if a string cannot be assigned to the current value.
func (b *Builder) SetString(s string) error {
	t := b.top()

	err := checkKind(t.Type(), reflect.String)
	if err != nil {
		return err
	}

	t.SetString(s)
	return nil
}

// Set the value at the cursor to the given boolean.
// Errors if a boolean cannot be assigned to the current value.
func (b *Builder) SetBool(v bool) error {
	t := b.top()

	err := checkKind(t.Type(), reflect.Bool)
	if err != nil {
		return err
	}

	t.SetBool(v)
	return nil
}

func (b *Builder) SetFloat(n float64) error {
	t := b.top()

	err := checkKindFloat(t.Type())
	if err != nil {
		return err
	}

	t.SetFloat(n)
	return nil
}

func (b *Builder) SetInt(n int64) error {
	t := b.top()

	err := checkKindInt(t.Type())
	if err != nil {
		return err
	}

	t.SetInt(n)
	return nil
}

func checkKindInt(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return nil
	}

	return IncorrectKindError{
		Actual:   rt.Kind(),
		Expected: reflect.Int,
	}
}

func checkKindFloat(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Float32, reflect.Float64:
		return nil
	}

	return IncorrectKindError{
		Actual:   rt.Kind(),
		Expected: reflect.Float64,
	}
}

func checkKind(rt reflect.Type, expected reflect.Kind) error {
	if rt.Kind() != expected {
		return IncorrectKindError{
			Actual:   rt.Kind(),
			Expected: expected,
		}
	}
	return nil
}

type IncorrectKindError struct {
	Actual   reflect.Kind
	Expected reflect.Kind
}

func (e IncorrectKindError) Error() string {
	return fmt.Sprintf("incorrect kind: expected '%s', got '%s'", e.Expected, e.Actual)
}

type FieldNotFoundError struct {
	Struct    reflect.Value
	FieldName string
}

func (e FieldNotFoundError) Error() string {
	return fmt.Sprintf("field not found: '%s' on '%s'", e.FieldName, e.Struct.Type())
}

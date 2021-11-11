//go:build go1.18
// +build go1.18

package danger

import (
	"reflect"
	"unsafe"
)

func ExtendSlice(t reflect.Type, s *Slice, n int) Slice {
	arrayType := reflect.ArrayOf(n, t.Elem())
	arrayData := reflect.New(arrayType)
	reflect.Copy(arrayData.Elem(), reflect.NewAt(t, unsafe.Pointer(s)).Elem())
	return Slice{
		Data: unsafe.Pointer(arrayData.Pointer()),
		Len:  s.Len,
		Cap:  n,
	}
}

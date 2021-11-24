//go:build !go1.18
// +build !go1.18

package danger

import (
	"reflect"
	"unsafe"
)

//go:linkname unsafe_NewArray reflect.unsafe_NewArray
func unsafe_NewArray(rtype unsafe.Pointer, length int) unsafe.Pointer

//go:linkname typedslicecopy reflect.typedslicecopy
//go:noescape
func typedslicecopy(elemType unsafe.Pointer, dst, src Slice) int

func ExtendSlice(t reflect.Type, s *Slice, n int) Slice {
	elemTypeRef := t.Elem()
	elemTypePtr := ((*iface)(unsafe.Pointer(&elemTypeRef))).ptr

	d := Slice{
		Data: unsafe_NewArray(elemTypePtr, n),
		Len:  s.Len,
		Cap:  n,
	}

	typedslicecopy(elemTypePtr, d, *s)
	return d
}

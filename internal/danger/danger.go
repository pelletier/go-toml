// Package danger provides optimized unsafe functions.
package danger

import (
	"fmt"
	"unsafe"
)

const maxInt = uintptr(int(^uint(0) >> 1))

func SubsliceOffset(data []byte, subslice []byte) int {
	datap := uintptr(unsafe.Pointer(unsafe.SliceData(data)))   // #nosec G103
	hlp := uintptr(unsafe.Pointer(unsafe.SliceData(subslice))) // #nosec G103

	if hlp < datap {
		panic(fmt.Errorf("subslice address (%d) is before data address (%d)", hlp, datap))
	}
	offset := hlp - datap

	if offset > maxInt {
		panic(fmt.Errorf("slice offset larger than int (%d)", offset))
	}

	intoffset := int(offset)

	if intoffset > len(data) {
		panic(fmt.Errorf("slice offset (%d) is farther than data length (%d)", intoffset, len(data)))
	}

	if intoffset+len(subslice) > len(data) {
		panic(fmt.Errorf("slice ends (%d+%d) is farther than data length (%d)", intoffset, len(subslice), len(data)))
	}

	return intoffset
}

func BytesRange(start []byte, end []byte) []byte {
	if start == nil || end == nil {
		panic("cannot call BytesRange with nil")
	}

	startp := uintptr(unsafe.Pointer(unsafe.SliceData(start))) // #nosec G103
	endp := uintptr(unsafe.Pointer(unsafe.SliceData(end)))     // #nosec G103

	if startp > endp {
		panic(fmt.Errorf("start pointer address (%d) is after end pointer address (%d)", startp, endp))
	}

	l := len(start)
	endLen := int(endp-startp) + len(end)
	if endLen > l {
		l = endLen
	}

	if l > cap(start) {
		panic(fmt.Errorf("range length is larger than capacity"))
	}

	return start[:l]
}

func Stride(ptr unsafe.Pointer, size uintptr, offset int) unsafe.Pointer {
	return unsafe.Add(ptr, size*uintptr(offset))
}

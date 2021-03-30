package errors

import (
	"fmt"
	"reflect"
	"unsafe"
)

const maxInt = uintptr(int(^uint(0) >> 1))


func UnsafeSubsliceOffset(data []byte, subslice []byte) int {
	datap := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	hlp := (*reflect.SliceHeader)(unsafe.Pointer(&subslice))


	if hlp.Data < datap.Data {
		panic(fmt.Errorf("subslice address (%d) is before data address (%d)", hlp.Data, datap.Data))
	}
	offset := hlp.Data - datap.Data

	if offset > maxInt {
		panic(fmt.Errorf("slice offset larger than int (%d)", offset))
	}

	intoffset := int(offset)

	if intoffset >= datap.Len {
		panic(fmt.Errorf("slice offset (%d) is farther than data length (%d)", intoffset, datap.Len))
	}

	if intoffset + hlp.Len > datap.Len {
		panic(fmt.Errorf("slice ends (%d+%d) is farther than data length (%d)", intoffset, hlp.Len, datap.Len))
	}

	return intoffset
}

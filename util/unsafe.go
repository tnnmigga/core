package util

import (
	"reflect"
	"unsafe"
)

func Address(fn any) uint64 {
	value := reflect.ValueOf(fn)
	ptr := unsafe.Pointer(value.Pointer())
	addr := uintptr(ptr)
	return uint64(addr)
}

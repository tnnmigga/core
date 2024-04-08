package util

import (
	"reflect"
	"unsafe"
)

// 获取任意类型的地址包括函数在内
func Address(a any) uint64 {
	value := reflect.ValueOf(a)
	ptr := unsafe.Pointer(value.Pointer())
	addr := uintptr(ptr)
	return uint64(addr)
}

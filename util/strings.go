package util

import "unsafe"

// 零拷贝字节数组转字符串
// 如果不确定是否存在并发则不使用
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 零拷贝字符串转字节数组
// 如果不确定是否存在并发则不使用
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

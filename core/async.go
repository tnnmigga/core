package core

import "github.com/tnnmigga/nett/idef"

// 异步回调的方式执行函数
// 启动一个新的goruntine执行同步阻塞的代码
// 执行完将结果返到模块线程往后执行
// 匿名函数捕获的变量需要防范并发读写问题
func Async[T any](m idef.IModule, f func() (T, error), cb func(T, error)) {
	m.Async(func() (any, error) {
		res, err := f()
		return res, err
	}, func(a any, err error) {
		cb(a.(T), err)
	})
}
